package helper

import (
	"encoding/json"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"

	"github.com/chai2010/webp"
)

func CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// MoveFile renames within a filesystem and falls back to copy-and-remove when
// the processed workspace is on a different volume.
func MoveFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	if _, err := os.Lstat(dst); err == nil {
		return fmt.Errorf("destination already exists: %s", dst)
	}
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	if err := CopyFile(src, dst); err != nil {
		return err
	}
	return os.Remove(src)
}

func ConvertToWebP(src, dst string, quality float32) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return err
	}

	outFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer outFile.Close()

	return webp.Encode(outFile, img, &webp.Options{Lossless: false, Quality: quality})
}

// ConvertToWebPBounded prefers the best visual quality at the ideal target
// size, accepts output up to maxOutputSize, and never lowers quality below 55.
func ConvertToWebPBounded(src, dst string, preferredQuality float32, qualitySpecified bool, targetSize, maxOutputSize, smallFileSize int64) (int64, float32, error) {
	fileInfo, err := os.Stat(src)
	if err != nil {
		return 0, 0, err
	}
	file, err := os.Open(src)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return 0, 0, err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return 0, 0, err
	}

	qualities := qualityCandidates(preferredQuality, qualitySpecified)
	if fileInfo.Size() <= smallFileSize {
		return encodeWebP(img, dst, qualities[0])
	}

	working := img
	for {
		var acceptedQuality float32
		for _, quality := range qualities {
			size, usedQuality, err := encodeWebP(working, dst, quality)
			if err != nil {
				return 0, 0, err
			}
			if size <= targetSize {
				return size, usedQuality, nil
			}
			if size <= maxOutputSize && acceptedQuality == 0 {
				acceptedQuality = usedQuality
			}
		}
		if acceptedQuality != 0 {
			return encodeWebP(working, dst, acceptedQuality)
		}

		bounds := working.Bounds()
		newWidth := int(float64(bounds.Dx()) * 0.85)
		newHeight := int(float64(bounds.Dy()) * 0.85)
		if newWidth < 1 || newHeight < 1 || (newWidth == bounds.Dx() && newHeight == bounds.Dy()) {
			return encodeWebP(working, dst, qualities[len(qualities)-1])
		}
		working = resizeNearest(working, newWidth, newHeight)
	}
}

func qualityCandidates(preferred float32, qualitySpecified bool) []float32 {
	if !qualitySpecified {
		return []float32{80, 75, 70, 65, 60, 55}
	}
	high := minFloat32(90, preferred+5)
	low := maxFloat32(55, preferred-5)
	candidates := []float32{high}
	if preferred != high && preferred != low {
		candidates = append(candidates, preferred)
	}
	if low != high {
		candidates = append(candidates, low)
	}
	return candidates
}

func minFloat32(left, right float32) float32 {
	if left < right {
		return left
	}
	return right
}
func maxFloat32(left, right float32) float32 {
	if left > right {
		return left
	}
	return right
}

func encodeWebP(img image.Image, dst string, quality float32) (int64, float32, error) {
	outFile, err := os.Create(dst)
	if err != nil {
		return 0, 0, err
	}
	err = webp.Encode(outFile, img, &webp.Options{Lossless: false, Quality: quality})
	closeErr := outFile.Close()
	if err != nil {
		return 0, 0, err
	}
	if closeErr != nil {
		return 0, 0, closeErr
	}
	info, err := os.Stat(dst)
	if err != nil {
		return 0, 0, err
	}
	return info.Size(), quality, nil
}

func resizeNearest(src image.Image, width, height int) image.Image {
	bounds := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		sourceY := bounds.Min.Y + y*bounds.Dy()/height
		for x := 0; x < width; x++ {
			sourceX := bounds.Min.X + x*bounds.Dx()/width
			dst.Set(x, y, src.At(sourceX, sourceY))
		}
	}
	return dst
}

func SaveManifest(path string, m *Manifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create manifest directory: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest to %s: %w", path, err)
	}
	return nil
}

func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	err = json.Unmarshal(data, &m)
	return &m, err
}

func FmtPrintfStagedInfo(stagedFolder string) {
	fmt.Printf("Staged screenshots location:\n%s\n\n", stagedFolder)
	fmt.Println("--> Review or remove any unwanted images in that directory now.")
}

func FmtPrintfDeleteOriginalsCmd(outDir string) {
	fmt.Printf("To delete original files later, run:\n./shrinker -mode=delete -out %s\n", outDir)
}

// FormatBytes presents storage amounts using binary units without unwieldy values.
func FormatBytes(bytes int64) string {
	sign := ""
	value := float64(bytes)
	if value < 0 {
		sign = "-"
		value = -value
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	unit := 0
	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%s%.0f %s", sign, value, units[unit])
	}
	return fmt.Sprintf("%s%.2f %s", sign, value, units[unit])
}
