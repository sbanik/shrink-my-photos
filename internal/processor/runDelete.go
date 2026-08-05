package processor

import (
	"fmt"
	"log"
	"os"
	"sync/atomic"

	"github.com/sbanik/shrink-my-photos/internal/helper"
	"github.com/schollz/progressbar/v3"
)

// runDeleteOriginals reads manifest.json and safely removes converted original files from external media
func RunDeleteOriginals(manifestPath string) {
	manifest, err := helper.LoadManifest(manifestPath)
	if err != nil {
		fmt.Printf("Error: Unable to load manifest at %s: %v\n", manifestPath, err)
		os.Exit(1)
	}

	var toDelete []*helper.FileRecord
	for _, rec := range manifest.Records {
		if rec.Status == "converted" {
			toDelete = append(toDelete, rec)
		}
	}

	if len(toDelete) == 0 {
		fmt.Println("No converted screenshots marked for deletion in manifest.")
		return
	}

	fmt.Println()
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

	helper.SaveManifest(manifestPath, manifest)

	fmt.Println("\n========================================")
	fmt.Println("        CLEANUP SUMMARY REPORT          ")
	fmt.Println("========================================")
	fmt.Printf("Successfully Deleted Originals : %d\n", deletedCount)
	fmt.Printf("Failed Deletions              : %d\n", failedCount)
	fmt.Println("========================================")
}