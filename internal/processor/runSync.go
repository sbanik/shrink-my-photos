package processor

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/sbanik/shrink-my-photos/internal/config"
	"github.com/sbanik/shrink-my-photos/internal/helper"
)

// RunSync cleans missing staged records and permanently deletes files moved into to_process/duplicates/ along with their originals
func RunSync(cfg *config.Config) int {
	manifest, err := helper.LoadManifest(cfg.ManifestPath)
	if err != nil {
		fmt.Printf("Error loading manifest at %s: %v\n", cfg.ManifestPath, err)
		return 0
	}

	duplicatesDir := filepath.Join(cfg.StagedFolder, "duplicates")
	var duplicateDeletedCount int

	// 1. Process files manually moved to "to_process/duplicates/"
	if _, err := os.Stat(duplicatesDir); err == nil {
		entries, err := os.ReadDir(duplicatesDir)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}

				dupFilePath := filepath.Join(duplicatesDir, entry.Name())
				stagedOriginalPath := filepath.Join(cfg.StagedFolder, entry.Name())

				// Locate corresponding record in manifest
				var targetKey string
				var record *helper.FileRecord

				for key, rec := range manifest.Records {
					if key == stagedOriginalPath || rec.StagedPath == stagedOriginalPath || filepath.Base(rec.StagedPath) == entry.Name() {
						targetKey = key
						record = rec
						break
					}
				}

				if record != nil {
					// Delete original file if present
					if record.OriginalPath != "" {
						if err := os.Remove(record.OriginalPath); err == nil || os.IsNotExist(err) {
							fmt.Printf("Deleted original: %s\n", record.OriginalPath)
						} else {
							log.Printf("Failed to delete original file %s: %v\n", record.OriginalPath, err)
						}
					}

					// Remove from manifest
					delete(manifest.Records, targetKey)
				}

				// Delete duplicate file from duplicates folder
				if err := os.Remove(dupFilePath); err == nil {
					fmt.Printf("Deleted duplicate: %s\n", dupFilePath)
					duplicateDeletedCount++
				} else {
					log.Printf("Failed to delete duplicate file %s: %v\n", dupFilePath, err)
				}
			}

			// Clean up empty duplicates directory
			_ = os.Remove(duplicatesDir)
		}
	}

	// 2. Sync manifest records for any remaining missing staged files
	var syncRemovedCount int
	for key, rec := range manifest.Records {
		if rec.Status == "staged" {
			if _, err := os.Stat(rec.StagedPath); os.IsNotExist(err) {
				delete(manifest.Records, key)
				syncRemovedCount++
			}
		}
	}

	_ = helper.SaveManifest(cfg.ManifestPath, manifest)

	fmt.Printf("\n=======================================================\n")
	fmt.Printf("Duplicates Deleted       : %d\n", duplicateDeletedCount)
	fmt.Printf("Manifest Records Removed : %d\n", syncRemovedCount)
	fmt.Printf("=======================================================\n")

	return duplicateDeletedCount
}