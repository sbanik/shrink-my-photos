package main

import (
	"image"
	"os"

	"github.com/rwcarlsen/goexif/exif"
)

// Common display aspect ratios (Width / Height)
var standardAspectRatios = []float64{
	16.0 / 9.0,  // 1.777 (1920x1080, 2560x1440, 3840x2160) - iPhone 6/7/8, SE series
	16.0 / 10.0, // 1.600 (MacBook Display aspect ratios)
	3.0 / 2.0,   // 1.500 (Surface, iPad, photography display)
	4.0 / 3.0,   // 1.333 (Standard display)
	19.5 / 9.0,  // 2.166 (Modern iPhone X, 11, 12, 13, 14, 15, 16 series / Android screen ratio)
}

func isScreenshot(path string) bool {
	// 1. Check metadata (EXIF data)
	f, err := os.Open(path)
	if err == nil {
		defer f.Close()
		
		x, err := exif.Decode(f)
		if err == nil {
			// Check EXIF Software/UserComment tag for "screenshot" or Apple capture markers
			softTag, _ := x.Get(exif.Software)
			if softTag != nil && softTag.String() != "" {
				// macOS screenshot metadata often contains "macOS" or empty camera details
				return true
			}
		}
	}

	// 2. Fallback: Image Dimensions and Aspect Ratio Analysis
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
	// Also check inverse ratio for vertical (portrait) screenshots
	invRatio := height / width 

	// Compare ratio against standard display aspect ratios (with tiny error tolerance)
	tolerance := 0.02
	for _, stdRatio := range standardAspectRatios {
		if (ratio >= stdRatio-tolerance && ratio <= stdRatio+tolerance) ||
		   (invRatio >= stdRatio-tolerance && invRatio <= stdRatio+tolerance) {
			return true
		}
	}

	return false
}