package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/chai2010/webp"
	"github.com/schollz/progressbar/v3"

	// Module internal package imports
	"github.com/sbanik/shrink-my-photos/internal/config"
	"github.com/sbanik/shrink-my-photos/internal/detector"
)

type FileRecord struct {
	OriginalPath string `json:"original_path"`
	StagedPath   string `json:"staged_path"`
	WebPPath     string `json:"webp_path"`
	OriginalSize int64  `json:"original_size_bytes"`
	WebPSize     int64  `json:"webp_size_bytes"`
	Status       string `json:"status"` // "staged", "converted", "completed", "failed"
}

type Manifest struct {
	Records map[string]*FileRecord `json:"records"`
}

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Configuration Error: %v\n", err)
		os.Exit(1)
	}

	// Setup logger
	logFile, err := os.OpenFile(cfg.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		defer logFile.Close()
		log.SetOutput(logFile)
		log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	}

	// ------------------------------------------------------------------
	// MODE 1: DELETE ORIGINALS ONLY
	// ------------------------------------------------------------------
	if cfg.DeleteOriginals {
		runDeleteOriginals(cfg.ManifestPath)
		return
	}

	// ------------------------------------------------------------------
	// MODE 2: STAGE & CONVERT (PRESERVE ORIGINALS)
	// ------------------------------------------------------------------
	if cfg.StagedFolder == "" {
		fmt.Println("Error: Source volume path (-volume) is required for detection and staging.")
		os.Exit(1)
	}

	_ = os.MkdirAll(cfg.StagedFolder, 0755)

	// STEP 1: DETECT & STAGE SCREENSHOTS
	fmt.Println("Scanning volume for screenshots...")
	var detected []string

	_ = filepath.WalkDir(cfg.StagedFolder, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if !d.IsDir() && (ext == ".png" || ext == ".jpg" || ext == ".jpeg") {
			if detector.IsScreenshot(path) {
				detected = append(detected, path)
			}
		}
		return nil
	})

	if len(detected) == 0 {
		fmt.Println("No screenshots found on target volume.")
		return
	}

	manifest := Manifest{Records: make(map[string]*FileRecord)}
	barStage := progressbar.Default(int64(len(detected)), "Staging screenshots")

	for i, srcPath := range detected {
		fileName := fmt.Sprintf("screenshot_%d_%s", i+1, filepath.Base(srcPath))
		destPath := filepath.Join(cfg.StagedFolder, fileName)

		fi, err := os.Stat(srcPath)
		var origSize int64
		if err == nil {
			origSize = fi.Size()
		}

		if err := copyFile(srcPath, destPath); err == nil {
			manifest.Records[destPath] = &FileRecord{
				OriginalPath: srcPath,
				StagedPath:   destPath,
				OriginalSize: origSize,
				Status:       "staged",
			}
		} else {
			log.Printf("Failed to stage %s -> %s: %v", srcPath, destPath, err)
		}
		_ = barStage.Add(1)
	}

	saveManifest(cfg.ManifestPath, &manifest)

	// STEP 2: INTERACTIVE PAUSE FOR MANUAL VERIFICATION
	fmt.Println("\n=======================================================")
	fmt.Printf("Staged %d screenshots to:\n%s\n\n", len(manifest.Records), cfg.StagedFolder)
	fmt.Println("--> Review or delete any unwanted images in that folder now.")
	fmt.Print("--> Press ENTER when ready to convert images to WebP... ")

	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')

	// STEP 3: CONVERT WEBP (LEAVE ORIGINALS INTACT)
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
	barConvert := progressbar.Default(int64(len(pending)), "Converting to WebP")
	sem := make(chan struct{}, cfg.Workers)
	var wg sync.WaitGroup

	var convertedCount, failedCount, totalBytesSaved int64

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

			if err := convertToWebP(r.StagedPath, webpPath, float32(cfg.Quality)); err != nil {
				r.Status = "failed"
				atomic.AddInt64(&failedCount, 1)
				log.Printf("Conversion error for %s: %v", r.StagedPath, err)
				return
			}

			// Stat the generated WebP file to calculate size savings
			webpInfo, err := os.Stat(webpPath)
			if err == nil {
				r.WebPSize = webpInfo.Size()
				bytesSaved := r.OriginalSize - r.WebPSize
				if bytesSaved > 0 {
					atomic.AddInt64(&totalBytesSaved, bytesSaved)
				}
			}

			// Clean up staged PNG copy
			_ = os.Remove(r.StagedPath)

			r.Status = "converted"
			atomic.AddInt64(&convertedCount, 1)
		}(rec)
	}

	wg.Wait()
	saveManifest(cfg.ManifestPath, &manifest)

	fmt.Println("\n========================================")
	fmt.Println("        PROCESS COMPLETE REPORT         ")
	fmt.Println("========================================")
	fmt.Printf("Successfully Converted : %d\n", convertedCount)
	fmt.Printf("Failed Conversions     : %d\n", failedCount)
	fmt.Printf("Total Storage Saved    : %.2f MB\n", float64(totalBytesSaved)/(1024*1024))
	fmt.Println("========================================")
	fmt.Println("Original files remain untouched on your volume.")
	fmt.Printf("To delete original files later, run:\n./shrinker -out %s -delete-originals\n", cfg.OutDir)
	fmt.Println("========================================")
}

// ------------------------------------------------------------------
// HELPER: SEPARATE CLEANUP / DELETION COMMAND
// ------------------------------------------------------------------
func runDeleteOriginals(manifestPath string) {
	manifest, err := loadManifest(manifestPath)
	if err != nil {
		fmt.Printf("Error: Unable to load manifest at %s: %v\n", manifestPath, err)
		os.Exit(1)
	}

	var toDelete []*FileRecord
	for _, rec := range manifest.Records {
		if rec.Status == "converted" {
			toDelete = append(toDelete, rec)
		}
	}

	if len(toDelete) == 0 {
		fmt.Println("No converted screenshots marked for deletion in manifest.")
		return
	}

	bar := progressbar.Default(int64(len(toDelete)), "Deleting original files")
	var deletedCount, failedCount int64

	for _, rec := range toDelete {
		if err := os.Remove(rec.OriginalPath); err == nil || os.IsNotExist(err) {
			rec.Status = "completed"
			atomic.AddInt64(&deletedCount, 1)
		} else {
			rec.Status = "failed"
			atomic.AddInt64(&failedCount, 1)
			log.Printf("Failed to delete original file %s: %v", rec.OriginalPath, err)
		}
		_ = bar.Add(1)
	}

	saveManifest(manifestPath, manifest)

	fmt.Println("\n========================================")
	fmt.Println("        CLEANUP SUMMARY REPORT          ")
	fmt.Println("========================================")
	fmt.Printf("Successfully Deleted Originals : %d\n", deletedCount)
	fmt.Printf("Failed Deletions              : %d\n", failedCount)
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

func loadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	err = json.Unmarshal(data, &m)
	return &m, err
}
