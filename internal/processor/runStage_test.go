package processor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sbanik/shrink-my-photos/internal/config"
	"github.com/sbanik/shrink-my-photos/internal/helper"
)

func TestRunStage_NormalAndDuplicateStaging(t *testing.T) {
	volDir := t.TempDir()
	outDir := t.TempDir()

	stagedFolder := filepath.Join(outDir, "to_process")
	duplicatesFolder := filepath.Join(stagedFolder, "duplicates")
	manifestPath := filepath.Join(outDir, "manifest.json")

	// 1. Create unique original file 1
	file1Path := filepath.Join(volDir, "photo1.png")
	file1Content := []byte("unique image content 1")
	if err := os.WriteFile(file1Path, file1Content, 0644); err != nil {
		t.Fatalf("Failed to write test file1: %v", err)
	}

	// 2. Create duplicate file (different name, identical content as file1)
	file2Path := filepath.Join(volDir, "photo1_copy.png")
	if err := os.WriteFile(file2Path, file1Content, 0644); err != nil {
		t.Fatalf("Failed to write duplicate test file2: %v", err)
	}

	// 3. Create unique original file 2
	file3Path := filepath.Join(volDir, "photo2.png")
	file3Content := []byte("unique image content 2")
	if err := os.WriteFile(file3Path, file3Content, 0644); err != nil {
		t.Fatalf("Failed to write test file3: %v", err)
	}

	cfg := &config.Config{
		VolumePath:       volDir,
		OutDir:           outDir,
		StagedFolder:     stagedFolder,
		DuplicatesFolder: duplicatesFolder,
		ManifestPath:     manifestPath,
		AllowedTypes:     []string{".png"},
		Clean:            false,
	}

	stagedCount := RunStage(cfg)

	// Should stage 2 unique files (photo1.png and photo2.png)
	if stagedCount != 2 {
		t.Errorf("Expected 2 files staged, got %d", stagedCount)
	}

	// Verify photo1.png was staged
	if _, err := os.Stat(filepath.Join(stagedFolder, "photo1.png")); os.IsNotExist(err) {
		t.Errorf("Expected photo1.png to exist in staged folder")
	}

	// Verify photo2.png was staged
	if _, err := os.Stat(filepath.Join(stagedFolder, "photo2.png")); os.IsNotExist(err) {
		t.Errorf("Expected photo2.png to exist in staged folder")
	}

	// Verify photo1_copy.png was identified as a duplicate and routed to duplicates/
	dupFilePath := filepath.Join(duplicatesFolder, "photo1_copy.png")
	if _, err := os.Stat(dupFilePath); os.IsNotExist(err) {
		t.Errorf("Expected duplicate photo1_copy.png to be routed to %s", dupFilePath)
	}

	// Verify manifest records
	manifest, err := helper.LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("Failed to reload manifest: %v", err)
	}

	if len(manifest.Records) != 2 {
		t.Errorf("Expected 2 manifest records, got %d", len(manifest.Records))
	}
}

func TestRunStage_CleanFlag(t *testing.T) {
	volDir := t.TempDir()
	outDir := t.TempDir()

	stagedFolder := filepath.Join(outDir, "to_process")
	duplicatesFolder := filepath.Join(stagedFolder, "duplicates")
	manifestPath := filepath.Join(outDir, "manifest.json")

	// Pre-create old leftover file in staged folder
	_ = os.MkdirAll(stagedFolder, 0755)
	oldStagedFile := filepath.Join(stagedFolder, "old.png")
	_ = os.WriteFile(oldStagedFile, []byte("old staged data"), 0644)

	// Create new source file
	newFile := filepath.Join(volDir, "new.png")
	_ = os.WriteFile(newFile, []byte("new image data"), 0644)

	cfg := &config.Config{
		VolumePath:       volDir,
		OutDir:           outDir,
		StagedFolder:     stagedFolder,
		DuplicatesFolder: duplicatesFolder,
		ManifestPath:     manifestPath,
		AllowedTypes:     []string{".png"},
		Clean:            true, // Enable clean flag
	}

	RunStage(cfg)

	// Verify old file was deleted during clean
	if _, err := os.Stat(oldStagedFile); !os.IsNotExist(err) {
		t.Errorf("Expected old staged file %s to be cleaned up", oldStagedFile)
	}

	// Verify new file was staged
	if _, err := os.Stat(filepath.Join(stagedFolder, "new.png")); os.IsNotExist(err) {
		t.Errorf("Expected new.png to be staged after cleaning")
	}
}