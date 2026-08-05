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

// Helper to create a test image matching standard screenshot aspect ratios (e.g., 16:9 -> 1920x1080)
func createMockScreenshot(t *testing.T, path string, width, height int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			img.Set(x, y, color.RGBA{R: 50, G: 50, B: 50, A: 255})
		}
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create mock screenshot image: %v", err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		t.Fatalf("Failed to encode mock PNG: %v", err)
	}
}

func TestRunStage_DetectAndStageScreenshots(t *testing.T) {
	tempVolume := t.TempDir()
	tempOut := t.TempDir()

	stagedFolder := filepath.Join(tempOut, "screenshots")
	manifestPath := filepath.Join(tempOut, "manifest.json")

	// 1. Create a valid screenshot (16:9 ratio)
	validShotPath := filepath.Join(tempVolume, "screen_1.png")
	createMockScreenshot(t, validShotPath, 1920, 1080)

	// 2. Create a non-screenshot image (odd ratio e.g., 100x300)
	nonShotPath := filepath.Join(tempVolume, "photo_1.png")
	createMockScreenshot(t, nonShotPath, 100, 300)

	cfg := &config.Config{
		Mode:         "stage",
		VolumePath:   tempVolume,
		OutDir:       tempOut,
		StagedFolder: stagedFolder,
		ManifestPath: manifestPath,
	}

	// 3. Run stage processing
	count := RunStage(cfg)

	if count != 1 {
		t.Errorf("Expected 1 staged screenshot, got %d", count)
	}

	// 4. Verify staged directory contents
	stagedEntries, err := os.ReadDir(stagedFolder)
	if err != nil {
		t.Fatalf("Failed to read staged folder: %v", err)
	}

	if len(stagedEntries) != 1 {
		t.Fatalf("Expected 1 file in staged directory, found %d", len(stagedEntries))
	}

	// 5. Verify manifest file contents
	manifest, err := helper.LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("Failed to load generated manifest: %v", err)
	}

	if len(manifest.Records) != 1 {
		t.Errorf("Expected 1 record in manifest, got %d", len(manifest.Records))
	}

	// Retrieve the single record
	for _, rec := range manifest.Records {
		if rec.OriginalPath != validShotPath {
			t.Errorf("Expected OriginalPath %s, got %s", validShotPath, rec.OriginalPath)
		}
		if rec.Status != "staged" {
			t.Errorf("Expected status 'staged', got '%s'", rec.Status)
		}
		if rec.OriginalSize == 0 {
			t.Errorf("Expected non-zero OriginalSize")
		}
	}
}

func TestRunStage_NoScreenshotsFound(t *testing.T) {
	tempVolume := t.TempDir()
	tempOut := t.TempDir()

	// Place only non-screenshot images
	nonShotPath := filepath.Join(tempVolume, "random_photo.png")
	createMockScreenshot(t, nonShotPath, 123, 456)

	cfg := &config.Config{
		Mode:         "stage",
		VolumePath:   tempVolume,
		OutDir:       tempOut,
		StagedFolder: filepath.Join(tempOut, "screenshots"),
		ManifestPath: filepath.Join(tempOut, "manifest.json"),
	}

	count := RunStage(cfg)
	if count != 0 {
		t.Errorf("Expected 0 screenshots staged, got %d", count)
	}
}

func TestRunStage_AppendsToExistingManifest(t *testing.T) {
	tempVolume := t.TempDir()
	tempOut := t.TempDir()

	stagedFolder := filepath.Join(tempOut, "screenshots")
	manifestPath := filepath.Join(tempOut, "manifest.json")

	// Pre-create an existing manifest record
	existingManifest := &helper.Manifest{
		Records: map[string]*helper.FileRecord{
			"/prior/path.png": {
				OriginalPath: "/prior/path.png",
				StagedPath:   "/prior/staged.png",
				Status:       "converted",
			},
		},
	}
	_ = helper.SaveManifest(manifestPath, existingManifest)

	// Add new valid screenshot to volume
	newShotPath := filepath.Join(tempVolume, "screen_new.png")
	createMockScreenshot(t, newShotPath, 1600, 900)

	cfg := &config.Config{
		Mode:         "stage",
		VolumePath:   tempVolume,
		OutDir:       tempOut,
		StagedFolder: stagedFolder,
		ManifestPath: manifestPath,
	}

	count := RunStage(cfg)

	// Manifest should now contain 2 records total (1 existing + 1 new)
	if count != 2 {
		t.Errorf("Expected total 2 manifest records after append, got %d", count)
	}

	updatedManifest, err := helper.LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("Failed to reload manifest: %v", err)
	}

	if _, exists := updatedManifest.Records["/prior/path.png"]; !exists {
		t.Errorf("Prior record was overwritten or lost during staging")
	}
}