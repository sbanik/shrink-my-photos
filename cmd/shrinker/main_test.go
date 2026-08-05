package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// Helper to generate a dummy PNG file with custom dimensions
func createTestPNG(t *testing.T, path string, width, height int) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for x := range width {
		for y := 0; y < height; y++ {
			img.Set(x, y, color.RGBA{50, 100, 150, 255})
		}
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create test image at %s: %v", path, err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		t.Fatalf("Failed to encode PNG: %v", err)
	}
}

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.png")
	dst := filepath.Join(dir, "dst.png")

	createTestPNG(t, src, 100, 100)

	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	srcInfo, _ := os.Stat(src)
	dstInfo, _ := os.Stat(dst)

	if srcInfo.Size() != dstInfo.Size() {
		t.Errorf("Copied file size (%d) does not match source size (%d)", dstInfo.Size(), srcInfo.Size())
	}
}

func TestConvertToWebP(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "test.png")
	dst := filepath.Join(dir, "test.webp")

	createTestPNG(t, src, 200, 200)

	err := convertToWebP(src, dst, 80.0)
	if err != nil {
		t.Fatalf("convertToWebP returned error: %v", err)
	}

	info, err := os.Stat(dst)
	if os.IsNotExist(err) {
		t.Fatalf("WebP file was not created")
	}
	if info.Size() == 0 {
		t.Errorf("WebP file was created but is empty")
	}
}

func TestManifestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "manifest.json")

	originalManifest := Manifest{
		Records: map[string]*FileRecord{
			"/staged/screenshot_1.png": {
				OriginalPath: "/volume/screenshot_1.png",
				StagedPath:   "/staged/screenshot_1.png",
				OriginalSize: 1024,
				Status:       "staged",
			},
		},
	}

	saveManifest(manifestPath, &originalManifest)

	loadedManifest, err := loadManifest(manifestPath)
	if err != nil {
		t.Fatalf("loadManifest failed: %v", err)
	}

	rec, exists := loadedManifest.Records["/staged/screenshot_1.png"]
	if !exists {
		t.Fatalf("Record missing from loaded manifest")
	}

	if rec.OriginalPath != "/volume/screenshot_1.png" || rec.Status != "staged" {
		t.Errorf("Loaded record data corrupted: %+v", rec)
	}
}

func TestRunDeleteOriginals(t *testing.T) {
	sourceDir := t.TempDir()
	outDir := t.TempDir()
	manifestPath := filepath.Join(outDir, "manifest.json")

	// Create a dummy original file that should be deleted
	origFile := filepath.Join(sourceDir, "orig.png")
	createTestPNG(t, origFile, 100, 100)

	manifest := Manifest{
		Records: map[string]*FileRecord{
			"/staged/orig.png": {
				OriginalPath: origFile,
				StagedPath:   "/staged/orig.png",
				Status:       "converted", // Marked as converted, ready for original deletion
			},
		},
	}
	saveManifest(manifestPath, &manifest)

	// Execute deletion run
	runDeleteOriginals(manifestPath)

	// Original file must be gone
	if _, err := os.Stat(origFile); !os.IsNotExist(err) {
		t.Errorf("runDeleteOriginals failed to remove original file %s", origFile)
	}

	// Manifest status must be updated to 'completed'
	updatedManifest, _ := loadManifest(manifestPath)
	if updatedManifest.Records["/staged/orig.png"].Status != "completed" {
		t.Errorf("Expected manifest status to be 'completed', got '%s'", updatedManifest.Records["/staged/orig.png"].Status)
	}
}