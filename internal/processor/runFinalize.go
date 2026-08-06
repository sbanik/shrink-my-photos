package processor

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"

	"github.com/sbanik/shrink-my-photos/internal/config"
	"github.com/sbanik/shrink-my-photos/internal/helper"
)

// RunFinalize moves verified WebP output back beside successfully deleted
// originals. It supports a processed folder on another filesystem.
func RunFinalize(cfg *config.Config) {
	manifest, err := helper.LoadManifest(cfg.ManifestPath)
	if err != nil {
		fmt.Printf("Unable to load manifest for finalization: %v\n", err)
		return
	}
	var finalized, failed int
	for _, record := range manifest.Records {
		if record.Status != "completed" {
			continue
		}
		if _, err := os.Stat(record.OriginalPath); !os.IsNotExist(err) {
			failed++
			log.Printf("Skipping finalization because original still exists: %s", record.OriginalPath)
			continue
		}
		target := replaceExtension(record.OriginalPath, ".webp")
		if err := helper.MoveFile(record.WebPPath, target); err != nil {
			failed++
			log.Printf("Failed to finalize %s to %s: %v", record.WebPPath, target, err)
			continue
		}
		record.WebPPath = target
		record.Status = "finalized"
		finalized++
	}
	if err := helper.SaveManifest(cfg.ManifestPath, manifest); err != nil {
		fmt.Printf("Could not save finalization manifest: %v\n", err)
	}
	cleanupEmptyDirectories(cfg.ProcessedFolder, true)
	fmt.Printf("Finalization complete: %d WebP file(s) moved into original folders, %d failed.\n", finalized, failed)
}

func cleanupEmptyDirectories(root string, removeRoot bool) {
	var directories []string
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err == nil && info != nil && info.IsDir() {
			directories = append(directories, path)
		}
		return nil
	})
	for _, directorie := range slices.Backward(directories) {
		if directorie == root && !removeRoot {
			continue
		}
		_ = os.Remove(directorie)
	}
}
