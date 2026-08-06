package processor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

	duplicatesDir := cfg.DuplicatesFolder
	if duplicatesDir == "" {
		duplicatesDir = filepath.Join(cfg.StagedFolder, "duplicates")
	}

	// Resolve absolute path for reliable checks
	absDuplicatesDir, err := filepath.Abs(duplicatesDir)
	if err != nil {
		absDuplicatesDir = duplicatesDir
	}

	fmt.Printf("Checking duplicates directory: %s\n", absDuplicatesDir)

	var duplicateDeletedCount int

	if info, err := os.Stat(absDuplicatesDir); err == nil && info.IsDir() {
		entries, err := os.ReadDir(absDuplicatesDir)
		if err != nil {
			fmt.Printf("Failed to read duplicates folder: %v\n", err)
		} else {
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}

				dupFileName := entry.Name()
				dupFilePath := filepath.Join(absDuplicatesDir, dupFileName)

				// Normalize staged target path for comparison
				stagedOriginalPath := filepath.Clean(filepath.Join(cfg.StagedFolder, dupFileName))

				// Locate corresponding record in manifest
				var targetKey string
				var record *helper.FileRecord

				for key, rec := range manifest.Records {
					cleanKey := filepath.Clean(key)
					cleanStaged := filepath.Clean(rec.StagedPath)

					if cleanKey == stagedOriginalPath ||
						cleanStaged == stagedOriginalPath ||
						strings.EqualFold(filepath.Base(rec.StagedPath), dupFileName) {
						targetKey = key
						record = rec
						break
					}
				}

				if record != nil && record.OriginalPath != "" {
					cleanOrigPath := filepath.Clean(record.OriginalPath)
					if err := os.Remove(cleanOrigPath); err == nil || os.IsNotExist(err) {
						fmt.Printf("Deleted original file: %s\n", cleanOrigPath)
					} else {
						fmt.Printf("ERROR: Failed to delete original file %s: %v\n", cleanOrigPath, err)
					}
					delete(manifest.Records, targetKey)
				} else {
					fmt.Printf("WARNING: No manifest record found for duplicate %s\n", dupFileName)
				}

				// Delete the file inside duplicates folder
				if err := os.Remove(dupFilePath); err == nil {
					fmt.Printf("Deleted duplicate file: %s\n", dupFilePath)
					duplicateDeletedCount++
				} else {
					fmt.Printf("ERROR: Failed to delete duplicate file %s: %v\n", dupFilePath, err)
				}
			}
		}
	} else {
		fmt.Printf("No duplicates directory found at: %s\n", absDuplicatesDir)
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