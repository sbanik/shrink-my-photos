package processor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sbanik/shrink-my-photos/internal/config"
	"github.com/sbanik/shrink-my-photos/internal/helper"
)

func TestRunSync_DiscardsDetectedAndManuallyMovedFiles(t *testing.T) {
	volume := t.TempDir()
	original := filepath.Join(volume, "album", "manual.png")
	discarded := filepath.Join(volume, "album", "discarded", "manual.png")
	if err := os.MkdirAll(filepath.Dir(discarded), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(discarded, []byte("discard me"), 0644); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	manifest := &helper.Manifest{Records: map[string]*helper.FileRecord{original: {OriginalPath: original, Status: "pending"}}}
	if err := helper.SaveManifest(manifestPath, manifest); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{VolumePath: volume, ManifestPath: manifestPath}

	if removed := RunSync(cfg); removed != 1 {
		t.Fatalf("removed = %d, want 1", removed)
	}
	if _, err := os.Stat(discarded); !os.IsNotExist(err) {
		t.Fatal("discarded file remains")
	}
	updated, err := helper.LoadManifest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	record := updated.Records[original]
	if record.Status != "discarded" || record.DiscardedPath != discarded {
		t.Fatalf("record not updated: %+v", record)
	}
}
