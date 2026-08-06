package processor

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sbanik/shrink-my-photos/internal/config"
	"github.com/sbanik/shrink-my-photos/internal/helper"
)

// RunSync permanently removes files that were moved to a discarded folder,
// including files the user moved there manually, and records the decision.
func RunSync(cfg *config.Config) int {
	manifest, err := helper.LoadManifest(cfg.ManifestPath)
	if err != nil {
		fmt.Printf("No manifest available to sync: %v\n", err)
		return 0
	}

	var removed, tracked int
	_ = filepath.Walk(cfg.VolumePath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if info.IsDir() {
			if path != cfg.VolumePath && info.Name() == "processed" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Base(filepath.Dir(path)) != "discarded" {
			return nil
		}

		if record := recordForDiscardedPath(manifest, path); record != nil {
			record.DiscardedPath = path
			record.Status = "discarded"
			tracked++
		}
		if err := os.Remove(path); err != nil {
			fmt.Printf("Could not discard %s: %v\n", path, err)
			return nil
		}
		removed++
		return nil
	})

	if err := helper.SaveManifest(cfg.ManifestPath, manifest); err != nil {
		fmt.Printf("Could not save manifest: %v\n", err)
	}
	fmt.Printf("Sync complete: discarded %d file(s); updated %d manifest record(s).\n", removed, tracked)
	return removed
}

func recordForDiscardedPath(manifest *helper.Manifest, discardedPath string) *helper.FileRecord {
	if record, ok := manifest.Records[discardedPath]; ok {
		return record
	}
	expectedOriginal := filepath.Join(filepath.Dir(filepath.Dir(discardedPath)), filepath.Base(discardedPath))
	if record, ok := manifest.Records[expectedOriginal]; ok {
		return record
	}
	for _, record := range manifest.Records {
		if filepath.Clean(record.DiscardedPath) == filepath.Clean(discardedPath) {
			return record
		}
	}
	return nil
}
