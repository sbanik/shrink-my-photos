package processor

import (
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/schollz/progressbar/v3"

	"github.com/sbanik/shrink-my-photos/internal/config"
	"github.com/sbanik/shrink-my-photos/internal/helper"
)

// runConvert converts all staged images to WebP concurrently and calculates total storage saved
func RunConvert(cfg *config.Config) {
	manifest, err := helper.LoadManifest(cfg.ManifestPath)
	if err != nil {
		fmt.Printf("Error: Unable to load manifest from %s: %v\n", cfg.ManifestPath, err)
		return
	}

	var pending []*helper.FileRecord
	for _, rec := range manifest.Records {
		if _, err := os.Stat(rec.StagedPath); err == nil && rec.Status == "staged" {
			pending = append(pending, rec)
		}
	}

	if len(pending) == 0 {
		fmt.Println("No staged images remaining for conversion.")
		return
	}

	fmt.Println()
	barConvert := progressbar.Default(int64(len(pending)), "Converting to WebP")
	sem := make(chan struct{}, cfg.Workers)
	var wg sync.WaitGroup

	var convertedCount, failedCount, totalBytesSaved int64

	for _, rec := range pending {
		wg.Add(1)
		go func(r *helper.FileRecord) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() {
				<-sem
				_ = barConvert.Add(1)
			}()

			ext := filepath.Ext(r.StagedPath)
			webpPath := r.StagedPath[:len(r.StagedPath)-len(ext)] + ".webp"
			r.WebPPath = webpPath

			if err := helper.ConvertToWebP(r.StagedPath, webpPath, float32(cfg.Quality)); err != nil {
				r.Status = "failed"
				atomic.AddInt64(&failedCount, 1)
				log.Printf("Conversion failed for %s: %v", r.StagedPath, err)
				return
			}

			webpInfo, err := os.Stat(webpPath)
			if err == nil {
				r.WebPSize = webpInfo.Size()
				bytesSaved := r.OriginalSize - r.WebPSize
				if bytesSaved > 0 {
					atomic.AddInt64(&totalBytesSaved, bytesSaved)
				}
			}

			_ = os.Remove(r.StagedPath)
			r.Status = "converted"
			atomic.AddInt64(&convertedCount, 1)
		}(rec)
	}

	wg.Wait()
	helper.SaveManifest(cfg.ManifestPath, manifest)

	fmt.Println("\n========================================")
	fmt.Println("        PROCESS COMPLETE REPORT         ")
	fmt.Println("========================================")
	fmt.Printf("Successfully Converted : %d\n", convertedCount)
	fmt.Printf("Failed Conversions     : %d\n", failedCount)
	fmt.Printf("Total Storage Saved    : %.2f MB\n", float64(totalBytesSaved)/(1024*1024))
	fmt.Println("========================================")
	fmt.Println("Original files remain untouched on your volume.")
	fmt.Printf("To delete original files later, run:\n./shrinker -mode=delete -out %s\n", cfg.OutDir)
	fmt.Println("========================================")
}

