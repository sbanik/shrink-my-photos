package detector

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func createTestImage(t *testing.T, dir, filename string, width, height int) string {
	t.Helper()
	filePath := filepath.Join(dir, filename)

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			img.Set(x, y, color.RGBA{100, 100, 100, 255})
		}
	}

	f, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("Failed to create test image at %s: %v", filePath, err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		t.Fatalf("Failed to encode test image: %v", err)
	}

	return filePath
}

func TestIsScreenshot(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		width    int
		height   int
		expected bool
	}{
		{
			name:     "iPhone 13-14 Portrait Screenshot (1170x2532 ~ 19.5:9)", // Removed '/'
			width:    1170,
			height:   2532,
			expected: true,
		},
		{
			name:     "iPhone 13-14 Landscape Screenshot (2532x1170 ~ 19.5:9)", // Removed '/'
			width:    2532,
			height:   1170,
			expected: true,
		},
		{
			name:     "Full HD Desktop Screenshot (1920x1080 ~ 16:9)",
			width:    1920,
			height:   1080,
			expected: true,
		},
		{
			name:     "MacBook Pro Display Screenshot (2560x1600 ~ 16:10)",
			width:    2560,
			height:   1600,
			expected: true,
		},
		{
			name:     "Standard 4:3 Display Screenshot (1024x768)",
			width:    1024,
			height:   768,
			expected: true,
		},
		{
			name:     "Square Photo (1000x1000 ~ 1:1 - Should Fail)",
			width:    1000,
			height:   1000,
			expected: false,
		},
		{
			name:     "Arbitrary Aspect Ratio Photo (1000x600 ~ 1.666 - Should Fail)",
			width:    1000,
			height:   600,
			expected: false,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Safely format the filename to avoid directory separator issues
			filename := fmt.Sprintf("test_img_%d.png", i)
			filePath := createTestImage(t, tempDir, filename, tt.width, tt.height)

			result := IsScreenshot(filePath)
			if result != tt.expected {
				t.Errorf("isScreenshot() for %dx%d = %v; want %v", tt.width, tt.height, result, tt.expected)
			}
		})
	}
}