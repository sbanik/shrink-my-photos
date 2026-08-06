package detector

import (
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"

	"github.com/rwcarlsen/goexif/exif"
)

var standardAspectRatios = []float64{
	16.0 / 9.0,  // 1.777 (1920x1080, 2560x1440, 3840x2160) - iPhone 6/7/8, SE series
	16.0 / 10.0, // 1.600 (MacBook Display aspect ratios)
	3.0 / 2.0,   // 1.500 (Surface, iPad, photography display)
	4.0 / 3.0,   // 1.333 (Standard display)
	19.5 / 9.0,  // 2.166 (Modern iPhone X, 11, 12, 13, 14, 15, 16 series / Android screen ratio)
}

var knownCameraMakes = []string{
	"nikon", "fujifilm", "canon", "sony", "panasonic",
	"olympus", "leica", "hasselblad", "pentax", "ricoh",
}

func IsScreenshot(path string) bool {
	if IsCameraPhoto(path) {
		return false
	}

	// Aspect Ratio Fallback
	fImage, err := os.Open(path)
	if err != nil {
		return false
	}
	defer fImage.Close()

	config, _, err := image.DecodeConfig(fImage)
	if err != nil {
		return false
	}

	width := float64(config.Width)
	height := float64(config.Height)

	if width == 0 || height == 0 {
		return false
	}

	ratio := width / height
	invRatio := height / width
	tolerance := 0.02

	for _, stdRatio := range standardAspectRatios {
		if (ratio >= stdRatio-tolerance && ratio <= stdRatio+tolerance) ||
			(invRatio >= stdRatio-tolerance && invRatio <= stdRatio+tolerance) {
			return true
		}
	}

	return false
}

// IsCameraPhoto identifies image files with camera EXIF metadata. It is
// intentionally conservative: files without readable EXIF are not classified
// as camera photos, so screenshots and downloaded images are not discarded.
func IsCameraPhoto(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	x, err := exif.Decode(f)
	if err != nil {
		return false
	}

	makeTag, err := x.Get(exif.Make)
	if err != nil {
		return false
	}
	makeStr := strings.ToLower(makeTag.String())
	for _, camera := range knownCameraMakes {
		if strings.Contains(makeStr, camera) {
			return true
		}
	}
	if !strings.Contains(makeStr, "apple") {
		return false
	}

	if lensTag, err := x.Get(exif.LensModel); err == nil && strings.TrimSpace(lensTag.String()) != "" {
		return true
	}
	if modelTag, err := x.Get(exif.Model); err == nil {
		model := strings.ToLower(modelTag.String())
		return strings.Contains(model, "iphone") || strings.Contains(model, "ipad")
	}
	return false
}
