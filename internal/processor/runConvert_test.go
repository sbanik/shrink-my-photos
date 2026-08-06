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

// Helper function to create a compressible test PNG image
func createTestPNG(t *testing.T, path string, width, height int) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	blue := color.RGBA{R: 100, G: 150, B: 200, A: 255}
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			img.Set(x, y, blue)
		}
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create test image file at %s: %v", path, err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		t.Fatalf("Failed to encode PNG image: %v", err)
	}
}

func TestRunConvert_SuccessfulConversion(t *testing.T) {
	outDir := t.TempDir()
	stagedFolder := filepath.Join(outDir, "to_process")
	processedFolder := filepath.Join(outDir, "processed")
	manifestPath := filepath.Join(outDir, "manifest.json")

	_ = os.MkdirAll(stagedFolder, 0755)

	stagedFile := filepath.Join(stagedFolder, "sample.png")
	createTestPNG(t, stagedFile, 200, 200)

	stagedInfo, err := os.Stat(stagedFile)
	if err != nil {
		t.Fatalf("Failed to stat staged file: %v", err)
	}

	manifest := &helper.Manifest{
		Records: map[string]*helper.FileRecord{
			stagedFile: {
				OriginalPath: "/dummy/original/sample.png",
				StagedPath:   stagedFile,
				OriginalSize: stagedInfo.Size(),
				Status:       "staged",
			},
		},
	}
	_ = helper.SaveManifest(manifestPath, manifest)

	cfg := &config.Config{
		OutDir:          outDir,
		StagedFolder:    stagedFolder,
		ProcessedFolder: processedFolder,
		ManifestPath:    manifestPath,
		Quality:         80.0,
		MinSavings:      0.05, // 5% minimum savings
		Workers:         2,
	}

	RunConvert(cfg)

	// 1. Check that output WebP file exists in ProcessedFolder
	expectedWebP := filepath.Join(processedFolder, "sample.webp")
	if _, err := os.Stat(expectedWebP); os.IsNotExist(err) {
		t.Errorf("Expected WebP file at %s, but it was not created", expectedWebP)
	}

	// 2. Check that the staged file was removed
	if _, err := os.Stat(stagedFile); !os.IsNotExist(err) {
		t.Errorf("Expected staged file %s to be deleted after conversion", stagedFile)
	}

	// 3. Verify updated manifest record
	reloadedManifest, err := helper.LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("Failed to reload manifest: %v", err)
	}

	rec, exists := reloadedManifest.Records[stagedFile]
	if !exists {
		t.Fatalf("Expected manifest record for %s to exist", stagedFile)
	}

	if rec.Status != "converted" {
		t.Errorf("Expected status 'converted', got '%s'", rec.Status)
	}
	if rec.WebPPath != expectedWebP {
		t.Errorf("Expected WebPPath '%s', got '%s'", expectedWebP, rec.WebPPath)
	}
	if rec.WebPSize <= 0 {
		t.Errorf("Expected positive WebPSize, got %d", rec.WebPSize)
	}
}

func TestRunConvert_SkippedLowSavings(t *testing.T) {
	outDir := t.TempDir()
	stagedFolder := filepath.Join(outDir, "to_process")
	processedFolder := filepath.Join(outDir, "processed")
	manifestPath := filepath.Join(outDir, "manifest.json")

	_ = os.MkdirAll(stagedFolder, 0755)

	stagedFile := filepath.Join(stagedFolder, "sample.png")
	createTestPNG(t, stagedFile, 200, 200)

	stagedInfo, err := os.Stat(stagedFile)
	if err != nil {
		t.Fatalf("Failed to stat staged file: %v", err)
	}

	manifest := &helper.Manifest{
		Records: map[string]*helper.FileRecord{
			stagedFile: {
				OriginalPath: "/dummy/original/sample.png",
				StagedPath:   stagedFile,
				OriginalSize: stagedInfo.Size(),
				Status:       "staged",
			},
		},
	}
	_ = helper.SaveManifest(manifestPath, manifest)

	cfg := &config.Config{
		OutDir:          outDir,
		StagedFolder:    stagedFolder,
		ProcessedFolder: processedFolder,
		ManifestPath:    manifestPath,
		Quality:         80.0,
		MinSavings:      0.99, // Require an impossible 99% savings threshold to trigger skip
		Workers:         1,
	}

	RunConvert(cfg)

	// 1. WebP file should have been removed
	expectedWebP := filepath.Join(processedFolder, "sample.webp")
	if _, err := os.Stat(expectedWebP); !os.IsNotExist(err) {
		t.Errorf("Expected WebP file at %s to be deleted due to low savings", expectedWebP)
	}

	// 2. Staged file should remain intact
	if _, err := os.Stat(stagedFile); os.IsNotExist(err) {
		t.Errorf("Expected staged file %s to still exist", stagedFile)
	}

	// 3. Manifest status should be updated to 'skipped_low_savings'
	reloadedManifest, err := helper.LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("Failed to reload manifest: %v", err)
	}

	rec := reloadedManifest.Records[stagedFile]
	if rec.Status != "skipped_low_savings" {
		t.Errorf("Expected status 'skipped_low_savings', got '%s'", rec.Status)
	}
}

func TestRunConvert_NoPendingFiles(t *testing.T) {
	outDir := t.TempDir()
	stagedFolder := filepath.Join(outDir, "to_process")
	processedFolder := filepath.Join(outDir, "processed")
	manifestPath := filepath.Join(outDir, "manifest.json")

	manifest := &helper.Manifest{
		Records: map[string]*helper.FileRecord{},
	}
	_ = helper.SaveManifest(manifestPath, manifest)

	cfg := &config.Config{
		OutDir:          outDir,
		StagedFolder:    stagedFolder,
		ProcessedFolder: processedFolder,
		ManifestPath:    manifestPath,
		Workers:         1,
	}

	RunConvert(cfg)
}