package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// Helper: Create a dummy PNG with specified dimensions
func createDummyPNG(t *testing.T, path string, width, height int) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			img.Set(x, y, color.RGBA{100, 150, 200, 255})
		}
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create test image at %s: %v", path, err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		t.Fatalf("Failed to encode test PNG: %v", err)
	}
}

// 1. Test Screenshot Detection Logic
func TestIsIPhoneScreenshot(t *testing.T) {
	tempDir := t.TempDir()

	// Case A: Valid iPhone 13/14 portrait screenshot resolution (1170x2532 ~ 19.5:9 ratio)
	validScreenshot := filepath.Join(tempDir, "iphone_screenshot.png")
	createDummyPNG(t, validScreenshot, 1170, 2532)

	if !isScreenshot(validScreenshot) {
		t.Errorf("Expected %s to be recognized as an iPhone screenshot", validScreenshot)
	}

	// Case B: Non-iPhone image / arbitrary square image (500x500 ~ 1:1 ratio)
	regularImage := filepath.Join(tempDir, "photo.png")
	createDummyPNG(t, regularImage, 500, 500)

	if isScreenshot(regularImage) {
		t.Errorf("Expected %s to NOT be recognized as an iPhone screenshot", regularImage)
	}
}

// 2. Test Pipeline Operations (Staging, Conversion, and Safe Deletion)
func TestPipelineFlow(t *testing.T) {
	sourceDir := t.TempDir()
	outputDir := t.TempDir()
	stagedFolder := filepath.Join(outputDir, "screenshots")
	_ = os.MkdirAll(stagedFolder, 0755)

	// Create a dummy source screenshot
	srcPath := filepath.Join(sourceDir, "test_screenshot.png")
	createDummyPNG(t, srcPath, 1170, 2532)

	// --- STAGE PHASE ---
	stagedPath := filepath.Join(stagedFolder, "screenshot_1_test_screenshot.png")
	if err := copyFile(srcPath, stagedPath); err != nil {
		t.Fatalf("Failed to copy file during staging: %v", err)
	}

	// Verify staged file exists and original is still intact on volume
	if _, err := os.Stat(stagedPath); os.IsNotExist(err) {
		t.Fatalf("Staged file was not created")
	}
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		t.Fatalf("Original source file was deleted prematurely during staging!")
	}

	// --- CONVERT PHASE ---
	ext := filepath.Ext(stagedPath)
	webpPath := stagedPath[:len(stagedPath)-len(ext)] + ".webp"

	// Convert staged copy to WebP
	err := convertToWebP(stagedPath, webpPath, 80.0)
	if err != nil {
		t.Fatalf("WebP conversion failed: %v", err)
	}

	// Clean up staged copy
	_ = os.Remove(stagedPath)

	// Safely delete original file ONLY after conversion is verified
	if _, err := os.Stat(webpPath); err == nil {
		_ = os.Remove(srcPath)
	}

	// --- FINAL ASSERTIONS ---
	// 1. WebP output must exist and non-empty
	webpInfo, err := os.Stat(webpPath)
	if os.IsNotExist(err) {
		t.Errorf("WebP output file was not created")
	} else if webpInfo.Size() == 0 {
		t.Errorf("WebP output file is empty")
	}

	// 2. Staged PNG must be cleaned up
	if _, err := os.Stat(stagedPath); err == nil {
		t.Errorf("Staged copy was not deleted after conversion")
	}

	// 3. Original source file must be deleted from source directory
	if _, err := os.Stat(srcPath); err == nil {
		t.Errorf("Original file was not deleted after successful conversion")
	}
}