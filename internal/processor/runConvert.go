package processor

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/schollz/progressbar/v3"

	"github.com/sbanik/shrink-my-photos/internal/config"
	"github.com/sbanik/shrink-my-photos/internal/helper"
)

// RunConvert writes WebP files below VOLUME_PATH/processed while retaining the
// source directory structure. Original files remain in place.
func RunConvert(cfg *config.Config) {
	manifest, err := helper.LoadManifest(cfg.ManifestPath)
	if err != nil {
		fmt.Printf("Unable to load manifest from %s: %v\n", cfg.ManifestPath, err)
		return
	}

	var pending []*helper.FileRecord
	for _, record := range manifest.Records {
		if record.Status != "pending" {
			continue
		}
		if _, err := os.Stat(record.OriginalPath); err == nil {
			pending = append(pending, record)
		}
	}
	if len(pending) == 0 {
		fmt.Println("No pending images remain for conversion.")
		cleanupDiscardedFolders(cfg.VolumePath)
		return
	}

	bar := progressbar.NewOptions(
		len(pending),
		progressbar.OptionSetDescription("Converting to WebP"),
		progressbar.OptionShowCount(),
		progressbar.OptionSetPredictTime(true),
	)
	sem := make(chan struct{}, cfg.Workers)
	var wg sync.WaitGroup
	maxOutputSize := cfg.MaxOutputSize
	if maxOutputSize <= 0 {
		maxOutputSize = 400 * 1024
	}
	var converted, failed, overTarget, overMaximum, storageChange int64

	for _, record := range pending {
		wg.Add(1)
		go func(record *helper.FileRecord) {
			defer wg.Done()
			defer func() { _ = bar.Add(1) }()
			sem <- struct{}{}
			defer func() { <-sem }()

			relativePath, err := filepath.Rel(cfg.VolumePath, record.OriginalPath)
			if err != nil || relativePath == "." || filepath.IsAbs(relativePath) || startsOutsideRoot(relativePath) {
				record.Status = "failed"
				atomic.AddInt64(&failed, 1)
				log.Printf("Refusing to write output outside processed folder for %s", record.OriginalPath)
				return
			}
			webpPath := filepath.Join(cfg.ProcessedFolder, replaceExtension(relativePath, ".webp"))
			size, quality, err := helper.ConvertToWebPBounded(record.OriginalPath, webpPath, float32(cfg.Quality), cfg.QualitySpecified, cfg.TargetSize, maxOutputSize, cfg.SmallFileSize)
			if err != nil {
				record.Status = "failed"
				atomic.AddInt64(&failed, 1)
				log.Printf("Conversion failed for %s: %v", record.OriginalPath, err)
				return
			}
			record.WebPPath = webpPath
			record.WebPSize = size
			record.Status = "converted"
			atomic.AddInt64(&storageChange, record.OriginalSize-size)
			if size > cfg.TargetSize {
				atomic.AddInt64(&overTarget, 1)
				log.Printf("%s exceeds the ideal target at quality %.0f", record.OriginalPath, quality)
			}
			if size > maxOutputSize {
				atomic.AddInt64(&overMaximum, 1)
				log.Printf("%s remains above the 400 KB maximum at quality %.0f", record.OriginalPath, quality)
			}
			atomic.AddInt64(&converted, 1)
		}(record)
	}
	wg.Wait()
	if err := helper.SaveManifest(cfg.ManifestPath, manifest); err != nil {
		fmt.Printf("Could not save manifest: %v\n", err)
	}
	cleanupDiscardedFolders(cfg.VolumePath)
	fmt.Printf("\nConversion complete: %d converted, %d failed, %d above ideal target, %d above 400 KB maximum.\n", converted, failed, overTarget, overMaximum)
	fmt.Printf("Potential storage savings after replacing originals: %s\n", helper.FormatBytes(storageChange))
}

func replaceExtension(path, extension string) string {
	return path[:len(path)-len(filepath.Ext(path))] + extension
}

func startsOutsideRoot(relativePath string) bool {
	return relativePath == ".." || len(relativePath) > 3 && relativePath[:3] == ".."+string(os.PathSeparator)
}

func cleanupDiscardedFolders(root string) {
	var dirs []string
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err == nil && info != nil && info.IsDir() && info.Name() == "discarded" {
			dirs = append(dirs, path)
		}
		return nil
	})
	for index := len(dirs) - 1; index >= 0; index-- {
		_ = os.Remove(dirs[index])
	}
}
