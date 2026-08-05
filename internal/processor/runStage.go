package processor

import (
	"slices"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/schollz/progressbar/v3"

	"github.com/sbanik/shrink-my-photos/internal/config"
	"github.com/sbanik/shrink-my-photos/internal/detector"
	"github.com/sbanik/shrink-my-photos/internal/helper"
)


// runStage scans the target volume, identifies screenshots, copies them to the staging directory, and builds/updates manifest.json
func RunStage(cfg *config.Config) int {
	_ = os.MkdirAll(cfg.StagedFolder, 0755)

	fmt.Printf("Scanning volume path: %s...\n", cfg.VolumePath)

	scanBar := progressbar.NewOptions(-1,
		progressbar.OptionSetDescription("Scanning files"),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionShowCount(),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionThrottle(65),
		progressbar.OptionClearOnFinish(),
	)

	manifest, err := helper.LoadManifest(cfg.ManifestPath)
	if err != nil {
		manifest = &helper.Manifest{Records: make(map[string]*helper.FileRecord)}
	}

	var totalStagedBytes int64
	var stagedCount int64
	var mu sync.Mutex

	jobs := make(chan string, 100)
	var wg sync.WaitGroup

	// Worker pool for concurrent staging
	for w := 0; w < cfg.Workers; w++ {
		wg.Go(func() {
			for srcPath := range jobs {
				origName := filepath.Base(srcPath)

				mu.Lock()
				destPath := getUniqueDestination(cfg.StagedFolder, origName)
				// Create empty placeholder file immediately to reserve destination name
				_ = os.WriteFile(destPath, []byte{}, 0644)
				mu.Unlock()

				fi, err := os.Stat(srcPath)
				var origSize int64
				if err == nil {
					origSize = fi.Size()
				}

				if err := helper.CopyFile(srcPath, destPath); err == nil {
					mu.Lock()
					manifest.Records[destPath] = &helper.FileRecord{
						OriginalPath: srcPath,
						StagedPath:   destPath,
						OriginalSize: origSize,
						Status:       "staged",
					}
					mu.Unlock()
					atomic.AddInt64(&totalStagedBytes, origSize)
					atomic.AddInt64(&stagedCount, 1)
				} else {
					log.Printf("Failed to stage %s -> %s: %v", srcPath, destPath, err)
					_ = os.Remove(destPath)
				}
			}
		})
	}

	_ = filepath.WalkDir(cfg.VolumePath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		_ = scanBar.Add(1)

		if !d.IsDir() && isTypeAllowed(filepath.Ext(path), cfg.AllowedTypes) {
			if detector.IsScreenshot(path) {
				jobs <- path
			}
		}
		return nil
	})

	close(jobs)
	wg.Wait()

	_ = scanBar.Finish()
	fmt.Println()

	if stagedCount == 0 {
		fmt.Println("No screenshots detected on target volume.")
		return 0
	}

	helper.SaveManifest(cfg.ManifestPath, manifest)

	totalMB := float64(totalStagedBytes) / (1024 * 1024)
	minSavedMB := totalMB * 0.60
	maxSavedMB := totalMB * 0.80

	fmt.Println("\n=======================================================")
	fmt.Printf("Staged Screenshots        : %d files (%.2f MB)\n", stagedCount, totalMB)
	fmt.Printf("Estimated Space Savings   : %.2f MB - %.2f MB (60%%-80%%)\n", minSavedMB, maxSavedMB)
	fmt.Println("=======================================================")

	if cfg.Mode == "stage" {
		fmt.Printf("Run conversion when ready using:\n./shrinker -mode=convert -out %s\n", cfg.OutDir)
		fmt.Println("=======================================================")
	}

	return int(stagedCount)
}

func isTypeAllowed(ext string, allowed []string) bool {
	ext = strings.ToLower(ext)
	return slices.Contains(allowed, ext)
}

func getUniqueDestination(dir, fileName string) string {
	dest := filepath.Join(dir, fileName)
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		return dest
	}

	ext := filepath.Ext(fileName)
	base := strings.TrimSuffix(fileName, ext)
	counter := 1

	for {
		candidate := filepath.Join(dir, fmt.Sprintf("%s_%d%s", base, counter, ext))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
		counter++
	}
}