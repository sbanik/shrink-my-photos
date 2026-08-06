package processor

import (
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/schollz/progressbar/v3"

	"github.com/sbanik/shrink-my-photos/internal/config"
	"github.com/sbanik/shrink-my-photos/internal/detector"
	"github.com/sbanik/shrink-my-photos/internal/helper"
)

// RunStage scans target volume, identifies screenshots, copies new ones to staging, and updates manifest.json
func RunStage(cfg *config.Config) int {
	// Clean output folder if requested
	if cfg.Clean {
		fmt.Println("Cleaning existing staged folder and manifest...")
		_ = os.RemoveAll(cfg.StagedFolder)
		_ = os.Remove(cfg.ManifestPath)
	}

	_ = os.MkdirAll(cfg.StagedFolder, 0755)
	_ = os.MkdirAll(cfg.DuplicatesFolder, 0755)

	fmt.Printf("Scanning volume path: %s...\n", cfg.VolumePath)

	// Step 1: Discover screenshots
	matchedFiles := scanForScreenshots(cfg.VolumePath, cfg.AllowedTypes)
	if len(matchedFiles) == 0 {
		fmt.Println("No screenshots detected on target volume.")
		return 0
	}

	fmt.Printf("Found %d screenshot(s). Staging files...\n", len(matchedFiles))

	manifest, err := helper.LoadManifest(cfg.ManifestPath)
	if err != nil || cfg.Clean {
		manifest = &helper.Manifest{Records: make(map[string]*helper.FileRecord)}
	}

	// Step 2: Concurrently stage files (skipping already existing ones)
	stagedCount, totalStagedBytes, skippedCount := stageFiles(cfg, matchedFiles, manifest)

	helper.SaveManifest(cfg.ManifestPath, manifest)

	totalMB := float64(totalStagedBytes) / (1024 * 1024)
	minSavedMB := totalMB * 0.60
	maxSavedMB := totalMB * 0.80

	fmt.Println("\n=======================================================")
	fmt.Printf("Staged Screenshots        : %d new, %d skipped (Total: %.2f MB)\n", stagedCount, skippedCount, totalMB)
	fmt.Printf("Estimated Space Savings   : %.2f MB - %.2f MB (60%%-80%%)\n", minSavedMB, maxSavedMB)
	fmt.Println("=======================================================")

	if cfg.Mode == "stage" {
		fmt.Printf("Run conversion when ready using:\n./shrinker -mode=convert -out %s\n", cfg.OutDir)
		fmt.Println("=======================================================")
	}

	return int(stagedCount)
}

// scanForScreenshots handles directory traversal, type filtering, and screenshot detection (Loop 1)
func scanForScreenshots(volumePath string, allowedTypes []string) []string {
	var matchedFiles []string

	scanBar := progressbar.NewOptions(-1,
		progressbar.OptionSetDescription("Scanning directory"),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionShowCount(),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionThrottle(65),
		progressbar.OptionClearOnFinish(),
	)

	_ = filepath.WalkDir(volumePath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		_ = scanBar.Add(1)

		if !d.IsDir() && isTypeAllowed(filepath.Ext(path), allowedTypes) {
			if detector.IsScreenshot(path) {
				matchedFiles = append(matchedFiles, path)
			}
		}
		return nil
	})

	_ = scanBar.Finish()
	fmt.Println()

	return matchedFiles
}

// stageFiles copies matched screenshots to the staging folder in parallel, skipping duplicates, and updates manifest records (Loop 2)
func stageFiles(cfg *config.Config, matchedFiles []string, manifest *helper.Manifest) (int64, int64, int64) {
	stageBar := progressbar.NewOptions(len(matchedFiles),
		progressbar.OptionSetDescription("Staging images"),
		progressbar.OptionShowCount(),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionThrottle(65),
		progressbar.OptionClearOnFinish(),
	)

	var totalStagedBytes int64
	var stagedCount int64
	var skippedCount int64
	var mu sync.Mutex

	jobs := make(chan string, len(matchedFiles))
	for _, file := range matchedFiles {
		jobs <- file
	}
	close(jobs)

	var wg sync.WaitGroup

	for w := 0; w < cfg.Workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for srcPath := range jobs {
				origName := filepath.Base(srcPath)
				expectedDest := filepath.Join(cfg.StagedFolder, origName)

				fi, err := os.Stat(srcPath)
				var origSize int64
				if err == nil {
					origSize = fi.Size()
				}

				// Skip if target file already exists with valid data
				if destFi, err := os.Stat(expectedDest); err == nil && destFi.Size() > 0 {
					mu.Lock()
					manifest.Records[expectedDest] = &helper.FileRecord{
						OriginalPath: srcPath,
						StagedPath:   expectedDest,
						OriginalSize: origSize,
						Status:       "staged",
					}
					mu.Unlock()
					atomic.AddInt64(&totalStagedBytes, origSize)
					atomic.AddInt64(&skippedCount, 1)
					_ = stageBar.Add(1)
					continue
				}

				mu.Lock()
				destPath := getUniqueDestination(cfg.StagedFolder, origName)
				// Reserve target path name instantly
				_ = os.WriteFile(destPath, []byte{}, 0644)
				mu.Unlock()

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
				_ = stageBar.Add(1)
			}
		}()
	}

	wg.Wait()
	_ = stageBar.Finish()
	fmt.Println()

	return stagedCount, totalStagedBytes, skippedCount
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