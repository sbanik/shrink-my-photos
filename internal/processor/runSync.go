package processor

import (
	"fmt"
	"os"

	"github.com/sbanik/shrink-my-photos/internal/config"
	"github.com/sbanik/shrink-my-photos/internal/helper"
)


func RunSync(cfg *config.Config) {
    manifest, err := helper.LoadManifest(cfg.ManifestPath)
    if err != nil { return }

    var removedCount int
    for key, rec := range manifest.Records {
        // If the staged file no longer exists and hasn't been converted/completed yet
        if rec.Status == "staged" {
            if _, err := os.Stat(rec.StagedPath); os.IsNotExist(err) {
                delete(manifest.Records, key)
                removedCount++
            }
        }
    }
    helper.SaveManifest(cfg.ManifestPath, manifest)
    fmt.Printf("Synced manifest: Removed %d missing records.\n", removedCount)
}