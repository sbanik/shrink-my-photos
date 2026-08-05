package helper

import (
	"encoding/json"
	"fmt"
	"image"
	"io"
	"os"

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

func SaveManifest(path string, m *Manifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
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