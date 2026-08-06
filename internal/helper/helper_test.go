package helper

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func createTestPNG(t *testing.T, path string, width, height int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			img.Set(x, y, color.RGBA{R: 200, G: 100, B: 50, A: 255})
		}
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create temp PNG file: %v", err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		t.Fatalf("Failed to encode temp PNG image: %v", err)
	}
}

func TestCopyFile(t *testing.T) {
	tempDir := t.TempDir()
	srcPath := filepath.Join(tempDir, "source.txt")
	dstPath := filepath.Join(tempDir, "destination.txt")

	content := []byte("Hello, Shrinker!")
	if err := os.WriteFile(srcPath, content, 0644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	if err := CopyFile(srcPath, dstPath); err != nil {
		t.Fatalf("CopyFile failed unexpectedly: %v", err)
	}

	copiedContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("Failed to read copied file: %v", err)
	}

	if string(copiedContent) != string(content) {
		t.Errorf("Copied content mismatch: got %s, want %s", string(copiedContent), string(content))
	}
}

func TestCopyFile_SourceNotFound(t *testing.T) {
	tempDir := t.TempDir()
	srcPath := filepath.Join(tempDir, "non_existent.txt")
	dstPath := filepath.Join(tempDir, "destination.txt")

	if err := CopyFile(srcPath, dstPath); err == nil {
		t.Errorf("Expected error when copying non-existent source file, got nil")
	}
}

func TestConvertToWebP(t *testing.T) {
	tempDir := t.TempDir()
	srcPath := filepath.Join(tempDir, "test.png")
	dstPath := filepath.Join(tempDir, "test.webp")

	createTestPNG(t, srcPath, 100, 100)

	err := ConvertToWebP(srcPath, dstPath, 80.0)
	if err != nil {
		t.Fatalf("ConvertToWebP failed unexpectedly: %v", err)
	}

	info, err := os.Stat(dstPath)
	if err != nil {
		t.Fatalf("WebP file was not created: %v", err)
	}

	if info.Size() == 0 {
		t.Errorf("Converted WebP file is empty")
	}
}

func TestManifestSaveAndLoad(t *testing.T) {
	tempDir := t.TempDir()
	manifestPath := filepath.Join(tempDir, "manifest.json")

	originalManifest := &Manifest{
		Records: map[string]*FileRecord{
			"/out/screenshot_1.png": {
				OriginalPath: "/volume/screenshot_1.png",
				StagedPath:   "/out/screenshot_1.png",
				WebPPath:     "/out/screenshot_1.webp",
				OriginalSize: 102400,
				WebPSize:     35000,
				Status:       "converted",
			},
		},
	}

	if err := SaveManifest(manifestPath, originalManifest); err != nil {
		t.Fatalf("SaveManifest failed: %v", err)
	}

	loadedManifest, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("LoadManifest failed: %v", err)
	}

	record, exists := loadedManifest.Records["/out/screenshot_1.png"]
	if !exists {
		t.Fatalf("Record not found in loaded manifest")
	}

	if record.OriginalPath != "/volume/screenshot_1.png" {
		t.Errorf("OriginalPath mismatch: got %s, want %s", record.OriginalPath, "/volume/screenshot_1.png")
	}
	if record.WebPSize != 35000 {
		t.Errorf("WebPSize mismatch: got %d, want 35000", record.WebPSize)
	}
	if record.Status != "converted" {
		t.Errorf("Status mismatch: got %s, want converted", record.Status)
	}
}

func TestLoadManifest_FileNotFound(t *testing.T) {
	tempDir := t.TempDir()
	missingPath := filepath.Join(tempDir, "missing_manifest.json")

	_, err := LoadManifest(missingPath)
	if err == nil {
		t.Errorf("Expected error loading non-existent manifest, got nil")
	}
}

func TestFormatBytes(t *testing.T) {
	if got := FormatBytes(10 * 1024 * 1024 * 1024); got != "10.00 GB" {
		t.Errorf("FormatBytes = %q", got)
	}
}
