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

// Helper to construct a dummy PNG file for testing conversion
func createDummyPNG(t *testing.T, path string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for x := 0; x < 100; x++ {
		for y := 0; y < 100; y++ {
			img.Set(x, y, color.RGBA{R: 100, G: 150, B: 200, A: 255})
		}
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create dummy PNG: %v", err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		t.Fatalf("Failed to encode dummy PNG: %v", err)
	}
}

func TestRunConvert_SuccessfulConversion(t *testing.T) {
	tempDir := t.TempDir()
	manifestPath := filepath.Join(tempDir, "manifest.json")
	stagedFile := filepath.Join(tempDir, "sample.png")

	// 1. Create a valid image in the staging location
	createDummyPNG(t, stagedFile)
	fi, _ := os.Stat(stagedFile)

	// 2. Build initial manifest with 'staged' status
	manifest := &helper.Manifest{
		Records: map[string]*helper.FileRecord{
			stagedFile: {
				OriginalPath: "/media/sample.png",
				StagedPath:   stagedFile,
				OriginalSize: fi.Size(),
				Status:       "staged",
			},
		},
	}
	if err := helper.SaveManifest(manifestPath, manifest); err != nil {
		t.Fatalf("Failed to setup manifest: %v", err)
	}

	cfg := &config.Config{
		Mode:         "convert",
		OutDir:       tempDir,
		ManifestPath: manifestPath,
		Quality:      80.0,
		Workers:      2,
	}

	// 3. Execute conversion
	RunConvert(cfg)

	// 4. Verify staged file was removed and .webp file was created
	if _, err := os.Stat(stagedFile); !os.IsNotExist(err) {
		t.Errorf("Expected staged PNG %s to be removed post-conversion", stagedFile)
	}

	expectedWebP := filepath.Join(tempDir, "sample.webp")
	if webpInfo, err := os.Stat(expectedWebP); err != nil || webpInfo.Size() == 0 {
		t.Errorf("Expected valid WebP output file at %s", expectedWebP)
	}

	// 5. Verify updated status and recorded WebP size in manifest
	updatedManifest, err := helper.LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("Failed to load updated manifest: %v", err)
	}

	record := updatedManifest.Records[stagedFile]
	if record.Status != "converted" {
		t.Errorf("Expected record status 'converted', got '%s'", record.Status)
	}
	if record.WebPPath != expectedWebP {
		t.Errorf("Expected WebPPath %s, got %s", expectedWebP, record.WebPPath)
	}
	if record.WebPSize == 0 {
		t.Errorf("Expected non-zero WebPSize in manifest")
	}
}

func TestRunConvert_CorruptFileFailure(t *testing.T) {
	tempDir := t.TempDir()
	manifestPath := filepath.Join(tempDir, "manifest.json")
	invalidStagedFile := filepath.Join(tempDir, "corrupt.png")

	// Write garbage data that cannot be decoded as an image
	_ = os.WriteFile(invalidStagedFile, []byte("invalid_image_data"), 0644)

	manifest := &helper.Manifest{
		Records: map[string]*helper.FileRecord{
			invalidStagedFile: {
				OriginalPath: "/media/corrupt.png",
				StagedPath:   invalidStagedFile,
				OriginalSize: 18,
				Status:       "staged",
			},
		},
	}
	_ = helper.SaveManifest(manifestPath, manifest)

	cfg := &config.Config{
		Mode:         "convert",
		OutDir:       tempDir,
		ManifestPath: manifestPath,
		Quality:      80.0,
		Workers:      1,
	}

	RunConvert(cfg)

	// Verify status updated to 'failed' and corrupt staged file remains untouched
	updatedManifest, err := helper.LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("Failed to load manifest: %v", err)
	}

	record := updatedManifest.Records[invalidStagedFile]
	if record.Status != "failed" {
		t.Errorf("Expected status 'failed' for corrupt image, got '%s'", record.Status)
	}

	if _, err := os.Stat(invalidStagedFile); os.IsNotExist(err) {
		t.Errorf("Corrupt staged file should not be removed on failure")
	}
}

func TestRunConvert_NoPendingImages(t *testing.T) {
	tempDir := t.TempDir()
	manifestPath := filepath.Join(tempDir, "manifest.json")

	manifest := &helper.Manifest{
		Records: map[string]*helper.FileRecord{
			"dummy": {
				OriginalPath: "/media/sample.png",
				StagedPath:   "/tmp/non_existent.png",
				Status:       "converted", // Already converted
			},
		},
	}
	_ = helper.SaveManifest(manifestPath, manifest)

	cfg := &config.Config{
		Mode:         "convert",
		OutDir:       tempDir,
		ManifestPath: manifestPath,
		Quality:      80.0,
		Workers:      1,
	}

	// Should return early and safely without panic
	RunConvert(cfg)
}