package processor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sbanik/shrink-my-photos/internal/config"
	"github.com/sbanik/shrink-my-photos/internal/helper"
)

func TestRunSync_RemovesMissingStagedFiles(t *testing.T) {
	tempDir := t.TempDir()
	manifestPath := filepath.Join(tempDir, "manifest.json")
	stagedFolder := filepath.Join(tempDir, "to_process")
	_ = os.MkdirAll(stagedFolder, 0755)

	existingFile := filepath.Join(stagedFolder, "keep.png")
	missingFile := filepath.Join(stagedFolder, "deleted.png")

	_ = os.WriteFile(existingFile, []byte("data"), 0644)

	manifest := &helper.Manifest{
		Records: map[string]*helper.FileRecord{
			existingFile: {
				OriginalPath: "/media/keep.png",
				StagedPath:   existingFile,
				Status:       "staged",
			},
			missingFile: {
				OriginalPath: "/media/deleted.png",
				StagedPath:   missingFile,
				Status:       "staged",
			},
		},
	}
	_ = helper.SaveManifest(manifestPath, manifest)

	cfg := &config.Config{
		Mode:         "sync",
		OutDir:       tempDir,
		ManifestPath: manifestPath,
	}

	RunSync(cfg)

	updated, err := helper.LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("Failed to reload manifest: %v", err)
	}

	if _, exists := updated.Records[missingFile]; exists {
		t.Errorf("Expected missing file record to be removed from manifest")
	}

	if _, exists := updated.Records[existingFile]; !exists {
		t.Errorf("Expected existing file record to remain in manifest")
	}
}