package detector

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// Helper function to create dummy PNG files with specific dimensions
func createTestImage(t *testing.T, dir string, filename string, width, height int) string {
	t.Helper()
	imgPath := filepath.Join(dir, filename)

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			img.Set(x, y, color.RGBA{R: 200, G: 200, B: 200, A: 255})
		}
	}

	f, err := os.Create(imgPath)
	if err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		t.Fatalf("Failed to encode PNG image: %v", err)
	}

	return imgPath
}

func TestIsScreenshot_StandardAspectRatios(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		width    int
		height   int
		expected bool
	}{
		{"16:9 Landscape (1920x1080)", 1920, 1080, true},
		{"16:9 Portrait (1080x1920)", 1080, 1920, true},
		{"16:10 MacBook (2560x1600)", 2560, 1600, true},
		{"3:2 Surface or iPad (3000x2000)", 3000, 2000, true},
		{"4:3 Standard Display (1024x768)", 1024, 768, true},
		{"19.5:9 Modern iPhone (2532x1170)", 2532, 1170, true},
		{"Custom Non-Screenshot Aspect Ratio (1000x400)", 1000, 400, false},
		{"Square Image (500x500)", 500, 500, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := createTestImage(t, tempDir, tt.name+".png", tt.width, tt.height)
			got := IsScreenshot(filePath)
			if got != tt.expected {
				t.Errorf("IsScreenshot() for %dx%d = %v, want %v", tt.width, tt.height, got, tt.expected)
			}
		})
	}
}

func TestIsScreenshot_InvalidFiles(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("Non-existent File", func(t *testing.T) {
		got := IsScreenshot(filepath.Join(tempDir, "does_not_exist.png"))
		if got != false {
			t.Errorf("Expected false for non-existent file, got true")
		}
	})

	t.Run("Non-Image File", func(t *testing.T) {
		dummyFile := filepath.Join(tempDir, "text.txt")
		_ = os.WriteFile(dummyFile, []byte("this is not an image"), 0644)

		got := IsScreenshot(dummyFile)
		if got != false {
			t.Errorf("Expected false for non-image text file, got true")
		}
	})
}