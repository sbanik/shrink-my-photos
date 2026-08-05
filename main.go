package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/chai2010/webp"
	"github.com/joho/godotenv"
	"github.com/schollz/progressbar/v3"
)

type FileRecord struct {
	OriginalPath string `json:"original_path"`
	StagedPath   string `json:"staged_path"`
	WebPPath     string `json:"webp_path"`
	Status       string `json:"status"` // "staged", "completed", "failed"
}

type Manifest struct {
	Records map[string]*FileRecord `json:"records"`
}

func main() {
	// 1. Load environment variables from .env if available
	_ = godotenv.Load()

	// 2. Resolve default flag values from environment variables
	envVolume := os.Getenv("VOLUME_PATH")
	envOut := os.Getenv("OUT_DIR")

	envQuality := 80.0
	if q, err := strconv.ParseFloat(os.Getenv("QUALITY"), 64); err == nil {
		envQuality = q
	}

	envWorkers := runtime.NumCPU()
	if w, err := strconv.Atoi(os.Getenv("WORKERS")); err == nil && w > 0 {
		envWorkers = w
	}

	// 3. Define CLI flags using environment values as defaults
	volumePath := flag.String("volume", envVolume, "Source volume path to scan (e.g. /Volumes/MySSD)")
	outDir := flag.String("out", envOut, "Output directory path for screenshots")
	quality := flag.Float64("quality", envQuality, "WebP compression quality (0-100)")
	workers := flag.Int("workers", envWorkers, "Parallel workers")
	flag.Parse()

	if *volumePath == "" || *outDir == "" {
		fmt.Println("Error: Both source volume and output directory paths are required.")
		fmt.Println("Set VOLUME_PATH and OUT_DIR in your .env file, or pass -volume and -out flags.")
		os.Exit(1)
	}

	stagedFolder := filepath.Join(*outDir, "screenshots")
	manifestPath := filepath.Join(*outDir, "manifest.json")
	_ = os.MkdirAll(stagedFolder, 0755)

	// ------------------------------------------------------------------
	// STEP 1: DETECT & STAGE SCREENSHOTS
	// ------------------------------------------------------------------
	fmt.Println("Scanning volume for iPhone screenshots...")
	var detected []string

	_ = filepath.WalkDir(*volumePath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if !d.IsDir() && (ext == ".png" || ext == ".jpg" || ext == ".jpeg") {
			if isScreenshot(path) {
				detected = append(detected, path)
			}
		}
		return nil
	})

	if len(detected) == 0 {
		fmt.Println("No iPhone screenshots found on volume.")
		return
	}

	manifest := Manifest{Records: make(map[string]*FileRecord)}
	barStage := progressbar.Default(int64(len(detected)), "Staging screenshots")

	for i, srcPath := range detected {
		fileName := fmt.Sprintf("screenshot_%d_%s", i+1, filepath.Base(srcPath))
		destPath := filepath.Join(stagedFolder, fileName)

		if err := copyFile(srcPath, destPath); err == nil {
			manifest.Records[destPath] = &FileRecord{
				OriginalPath: srcPath,
				StagedPath:   destPath,
				Status:       "staged",
			}
		}
		_ = barStage.Add(1)
	}

	saveManifest(manifestPath, &manifest)

	// ------------------------------------------------------------------
	// STEP 2: INTERACTIVE PAUSE FOR MANUAL VERIFICATION
	// ------------------------------------------------------------------
	fmt.Println("\n=======================================================")
	fmt.Printf("Staged %d screenshots to:\n%s\n\n", len(manifest.Records), stagedFolder)
	fmt.Println("--> Open the folder above to review or delete any unwanted images now.")
	fmt.Print("--> Press ENTER when ready to convert to WebP and delete originals... ")

	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')

	// ------------------------------------------------------------------
	// STEP 3: CONVERT WEBP & DELETE ORIGINALS
	// ------------------------------------------------------------------
	var pending []*FileRecord
	for _, rec := range manifest.Records {
		if _, err := os.Stat(rec.StagedPath); err == nil && rec.Status == "staged" {
			pending = append(pending, rec)
		}
	}

	if len(pending) == 0 {
		fmt.Println("No staged files remaining to convert.")
		return
	}

	fmt.Println()
	barConvert := progressbar.Default(int64(len(pending)), "Converting & deleting originals")
	sem := make(chan struct{}, *workers)
	var wg sync.WaitGroup

	var convertedCount, failedCount int64

	for _, rec := range pending {
		wg.Add(1)
		go func(r *FileRecord) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() {
				<-sem
				_ = barConvert.Add(1)
			}()

			ext := filepath.Ext(r.StagedPath)
			webpPath := r.StagedPath[:len(r.StagedPath)-len(ext)] + ".webp"
			r.WebPPath = webpPath

			// Convert staged file to WebP
			if err := convertToWebP(r.StagedPath, webpPath, float32(*quality)); err != nil {
				r.Status = "failed"
				atomic.AddInt64(&failedCount, 1)
				return
			}

			// Delete staged file
			_ = os.Remove(r.StagedPath)

			// Delete original source image safely AFTER conversion completion
			if err := os.Remove(r.OriginalPath); err != nil {
				r.Status = "failed"
				atomic.AddInt64(&failedCount, 1)
				return
			}

			r.Status = "completed"
			atomic.AddInt64(&convertedCount, 1)
		}(rec)
	}

	wg.Wait()
	saveManifest(manifestPath, &manifest)

	fmt.Println("\n========================================")
	fmt.Println("        PROCESS COMPLETE REPORT         ")
	fmt.Println("========================================")
	fmt.Printf("Successfully Converted & Deleted Originals : %d\n", convertedCount)
	fmt.Printf("Failed / Preserved Originals             : %d\n", failedCount)
	fmt.Println("========================================")
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func convertToWebP(src, dst string, quality float32) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return err
	}

	outFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer outFile.Close()

	return webp.Encode(outFile, img, &webp.Options{Lossless: false, Quality: quality})
}

func saveManifest(path string, m *Manifest) {
	data, _ := json.MarshalIndent(m, "", "  ")
	_ = os.WriteFile(path, data, 0644)
}