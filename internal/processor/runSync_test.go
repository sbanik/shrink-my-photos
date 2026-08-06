package processor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sbanik/shrink-my-photos/internal/config"
	"github.com/sbanik/shrink-my-photos/internal/helper"
)

func TestRunSync_DuplicatesDirectory(t *testing.T) {
	outDir := t.TempDir()
	volDir := t.TempDir()

	stagedFolder := filepath.Join(outDir, "to_process")
	duplicatesDir := filepath.Join(stagedFolder, "duplicates")
	manifestPath := filepath.Join(outDir, "manifest.json")

	_ = os.MkdirAll(duplicatesDir, 0755)

	// Create original file
	originalFile := filepath.Join(volDir, "photo_1.png")
	_ = os.WriteFile(originalFile, []byte("original image data"), 0644)

	// Create duplicate file inside to_process/duplicates/
	duplicateFile := filepath.Join(duplicatesDir, "photo_1.png")
	_ = os.WriteFile(duplicateFile, []byte("duplicate image data"), 0644)

	// Build manifest with original staged path mapping
	stagedPath := filepath.Join(stagedFolder, "photo_1.png")
	manifest := &helper.Manifest{
		Records: map[string]*helper.FileRecord{
			stagedPath: {
				OriginalPath: originalFile,
				StagedPath:   stagedPath,
				Status:       "staged",
			},
		},
	}
	_ = helper.SaveManifest(manifestPath, manifest)

	cfg := &config.Config{
		OutDir:           outDir,
		StagedFolder:     stagedFolder,
		DuplicatesFolder: duplicatesDir,
		ManifestPath:     manifestPath,
	}

	deletedCount := RunSync(cfg)

	if deletedCount != 1 {
		t.Fatalf("Expected 1 duplicate deleted, got %d", deletedCount)
	}

	// 1. Original file must be deleted
	if _, err := os.Stat(originalFile); !os.IsNotExist(err) {
		t.Errorf("Expected original file %s to be deleted", originalFile)
	}

	// 2. Duplicate file in duplicates/ must be deleted
	if _, err := os.Stat(duplicateFile); !os.IsNotExist(err) {
		t.Errorf("Expected duplicate file %s to be deleted", duplicateFile)
	}

	// 3. Duplicates directory itself must NOT be deleted
	if _, err := os.Stat(duplicatesDir); os.IsNotExist(err) {
		t.Errorf("Expected duplicates directory %s to still exist", duplicatesDir)
	}

	// 4. Manifest record must be deleted
	reloadedManifest, err := helper.LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("Failed to reload manifest: %v", err)
	}
	if _, exists := reloadedManifest.Records[stagedPath]; exists {
		t.Errorf("Expected record for %s to be deleted from manifest", stagedPath)
	}
}

func TestRunSync_MissingStagedFiles(t *testing.T) {
	outDir := t.TempDir()
	stagedFolder := filepath.Join(outDir, "to_process")
	manifestPath := filepath.Join(outDir, "manifest.json")
	_ = os.MkdirAll(stagedFolder, 0755)

	// Staged record for a missing file
	missingStagedPath := filepath.Join(stagedFolder, "missing.png")
	manifest := &helper.Manifest{
		Records: map[string]*helper.FileRecord{
			missingStagedPath: {
				OriginalPath: "/path/to/orig.png",
				StagedPath:   missingStagedPath,
				Status:       "staged",
			},
		},
	}
	_ = helper.SaveManifest(manifestPath, manifest)

	cfg := &config.Config{
		OutDir:           outDir,
		StagedFolder:     stagedFolder,
		DuplicatesFolder: filepath.Join(stagedFolder, "duplicates"),
		ManifestPath:     manifestPath,
	}

	RunSync(cfg)

	reloadedManifest, err := helper.LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("Failed to reload manifest: %v", err)
	}

	if len(reloadedManifest.Records) != 0 {
		t.Errorf("Expected 0 records in manifest after syncing missing files, got %d", len(reloadedManifest.Records))
	}
}