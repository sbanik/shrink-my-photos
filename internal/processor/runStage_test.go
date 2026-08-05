package processor

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/sbanik/shrink-my-photos/internal/config"
	"github.com/sbanik/shrink-my-photos/internal/helper"
)

// Helper to generate a dummy 16:9 PNG screenshot that passes detector.IsScreenshot
func createDummyScreenshot(t *testing.T, path string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1920, 1080))
	for x := 0; x < 50; x++ {
		for y := 0; y < 50; y++ {
			img.Set(x, y, color.RGBA{R: 50, G: 100, B: 150, A: 255})
		}
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		t.Fatalf("Failed to encode test image: %v", err)
	}
}

func TestScanForScreenshots(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Valid 16:9 screenshot
	shotPath := filepath.Join(tempDir, "screenshot.png")
	createDummyScreenshot(t, shotPath)

	// 2. Square non-screenshot image (500x500)
	nonShotPath := filepath.Join(tempDir, "regular.png")
	img := image.NewRGBA(image.Rect(0, 0, 500, 500))
	f, _ := os.Create(nonShotPath)
	_ = png.Encode(f, img)
	f.Close()

	// 3. Disallowed file type
	txtPath := filepath.Join(tempDir, "note.txt")
	_ = os.WriteFile(txtPath, []byte("text file"), 0644)

	allowedTypes := []string{".png"}
	matchedFiles := scanForScreenshots(tempDir, allowedTypes)

	if len(matchedFiles) != 1 {
		t.Fatalf("Expected 1 screenshot found, got %d", len(matchedFiles))
	}
	if matchedFiles[0] != shotPath {
		t.Errorf("Expected matched file %s, got %s", shotPath, matchedFiles[0])
	}
}

func TestStageFilesAndManifest(t *testing.T) {
	volDir := t.TempDir()
	outDir := t.TempDir()

	srcFile := filepath.Join(volDir, "screenshot.png")
	createDummyScreenshot(t, srcFile)

	stagedFolder := filepath.Join(outDir, "to_process")
	manifestPath := filepath.Join(outDir, "manifest.json")
	_ = os.MkdirAll(stagedFolder, 0755)

	cfg := &config.Config{
		Mode:         "stage",
		VolumePath:   volDir,
		OutDir:       outDir,
		StagedFolder: stagedFolder,
		ManifestPath: manifestPath,
		Workers:      2,
		AllowedTypes: []string{".png"},
	}

	manifest := &helper.Manifest{Records: make(map[string]*helper.FileRecord)}
	matchedFiles := []string{srcFile}

	stagedCount, totalBytes, skippedCount := stageFiles(cfg, matchedFiles, manifest)

	if stagedCount != 1 {
		t.Fatalf("Expected 1 staged file, got %d", stagedCount)
	}
	if skippedCount != 0 {
		t.Fatalf("Expected 0 skipped files, got %d", skippedCount)
	}
	if totalBytes <= 0 {
		t.Fatalf("Expected non-zero total staged bytes, got %d", totalBytes)
	}

	stagedDest := filepath.Join(stagedFolder, "screenshot.png")
	record, exists := manifest.Records[stagedDest]
	if !exists {
		t.Fatalf("Manifest record for %s not found", stagedDest)
	}
	if record.OriginalPath != srcFile {
		t.Errorf("Expected OriginalPath %s, got %s", srcFile, record.OriginalPath)
	}
}

func TestStageFiles_SkipExisting(t *testing.T) {
	volDir := t.TempDir()
	outDir := t.TempDir()

	srcFile := filepath.Join(volDir, "screenshot.png")
	createDummyScreenshot(t, srcFile)

	stagedFolder := filepath.Join(outDir, "to_process")
	manifestPath := filepath.Join(outDir, "manifest.json")
	_ = os.MkdirAll(stagedFolder, 0755)

	// Pre-create file in staged destination so it gets skipped
	existingStagedFile := filepath.Join(stagedFolder, "screenshot.png")
	_ = os.WriteFile(existingStagedFile, []byte("pre-existing contents"), 0644)

	cfg := &config.Config{
		Mode:         "stage",
		VolumePath:   volDir,
		OutDir:       outDir,
		StagedFolder: stagedFolder,
		ManifestPath: manifestPath,
		Workers:      1,
		AllowedTypes: []string{".png"},
	}

	manifest := &helper.Manifest{Records: make(map[string]*helper.FileRecord)}
	stagedCount, _, skippedCount := stageFiles(cfg, []string{srcFile}, manifest)

	if stagedCount != 0 {
		t.Errorf("Expected 0 newly staged files, got %d", stagedCount)
	}
	if skippedCount != 1 {
		t.Errorf("Expected 1 skipped file, got %d", skippedCount)
	}
}

func TestRunStage_CleanFlag(t *testing.T) {
	volDir := t.TempDir()
	outDir := t.TempDir()

	srcFile := filepath.Join(volDir, "screenshot.png")
	createDummyScreenshot(t, srcFile)

	stagedFolder := filepath.Join(outDir, "to_process")
	manifestPath := filepath.Join(outDir, "manifest.json")
	_ = os.MkdirAll(stagedFolder, 0755)

	// Pre-create a stale artifact that should be cleaned
	staleFile := filepath.Join(stagedFolder, "stale.png")
	_ = os.WriteFile(staleFile, []byte("stale data"), 0644)

	cfg := &config.Config{
		Mode:         "stage",
		VolumePath:   volDir,
		OutDir:       outDir,
		StagedFolder: stagedFolder,
		ManifestPath: manifestPath,
		Workers:      1,
		Clean:        true,
		AllowedTypes: []string{".png"},
	}

	stagedCount := RunStage(cfg)

	if stagedCount != 1 {
		t.Fatalf("Expected 1 staged file, got %d", stagedCount)
	}

	if _, err := os.Stat(staleFile); !os.IsNotExist(err) {
		t.Errorf("Expected stale file %s to be cleaned up, but it still exists", staleFile)
	}
}

func TestGetUniqueDestination(t *testing.T) {
	tempDir := t.TempDir()

	dest1 := getUniqueDestination(tempDir, "file.png")
	expected1 := filepath.Join(tempDir, "file.png")
	if dest1 != expected1 {
		t.Errorf("Expected %s, got %s", expected1, dest1)
	}

	_ = os.WriteFile(expected1, []byte("data"), 0644)
	dest2 := getUniqueDestination(tempDir, "file.png")
	expected2 := filepath.Join(tempDir, "file_1.png")
	if dest2 != expected2 {
		t.Errorf("Expected %s, got %s", expected2, dest2)
	}
}

func TestIsTypeAllowed(t *testing.T) {
	allowed := []string{".png", ".jpg"}

	if !isTypeAllowed(".PNG", allowed) {
		t.Errorf("Expected uppercase .PNG to be allowed")
	}
	if isTypeAllowed(".gif", allowed) {
		t.Errorf("Expected .gif to be rejected")
	}
}