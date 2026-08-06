package processor

import (
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
	"github.com/sbanik/shrink-my-photos/internal/helper"
)

// RunConvert converts all staged images to WebP concurrently, saves them to ProcessedFolder, and calculates total storage saved
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

	// Ensure the processed output directory exists before conversion
	if err := os.MkdirAll(cfg.ProcessedFolder, 0755); err != nil {
		fmt.Printf("Error: Unable to create processed directory %s: %v\n", cfg.ProcessedFolder, err)
		return
	}

	fmt.Println()
	barConvert := progressbar.Default(int64(len(pending)), "Converting to WebP")
	sem := make(chan struct{}, cfg.Workers)
	var wg sync.WaitGroup

	var convertedCount, skippedCount, failedCount, totalBytesSaved int64

	for _, rec := range pending {
		wg.Add(1)
		go func(r *helper.FileRecord) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() {
				<-sem
				_ = barConvert.Add(1)
			}()

			filename := filepath.Base(r.StagedPath)
			ext := filepath.Ext(filename)
			baseName := strings.TrimSuffix(filename, ext)

			// Route output file to ProcessedFolder
			webpPath := filepath.Join(cfg.ProcessedFolder, baseName+".webp")
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

				// Check against configured savings threshold
				savingsRatio := float64(bytesSaved) / float64(r.OriginalSize)
				if savingsRatio < cfg.MinSavings {
					r.Status = "skipped_low_savings"
					_ = os.Remove(webpPath) // Remove inefficient WebP output
					atomic.AddInt64(&skippedCount, 1)
					return
				}

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
	_ = helper.SaveManifest(cfg.ManifestPath, manifest)

	fmt.Println("\n========================================")
	fmt.Println("        PROCESS COMPLETE REPORT         ")
	fmt.Println("========================================")
	fmt.Printf("Successfully Converted : %d\n", convertedCount)
	fmt.Printf("Skipped (Low Savings)  : %d\n", skippedCount)
	fmt.Printf("Failed Conversions     : %d\n", failedCount)
	fmt.Printf("Total Storage Saved    : %.2f MB\n", float64(totalBytesSaved)/(1024*1024))
	fmt.Println("========================================")
}