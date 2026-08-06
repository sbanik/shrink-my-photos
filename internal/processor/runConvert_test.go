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

func createTestPNG(t *testing.T, path string, width, height int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for x := range width {
		for y := range height {
			img.Set(x, y, color.RGBA{100, 150, 200, 255})
		}
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, img); err != nil {
		t.Fatal(err)
	}
}

func TestRunConvert_PreservesDirectoryStructureAndOriginal(t *testing.T) {
	volume := t.TempDir()
	original := filepath.Join(volume, "nested", "sample.png")
	if err := os.MkdirAll(filepath.Dir(original), 0755); err != nil {
		t.Fatal(err)
	}
	createTestPNG(t, original, 300, 300)
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	manifest := &helper.Manifest{Records: map[string]*helper.FileRecord{original: {OriginalPath: original, Status: "pending"}}}
	if err := helper.SaveManifest(manifestPath, manifest); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{VolumePath: volume, ProcessedFolder: filepath.Join(volume, "processed"), ManifestPath: manifestPath, Quality: 90, TargetSize: 250 * 1024, SmallFileSize: 150 * 1024, Workers: 1}

	RunConvert(cfg)

	webp := filepath.Join(volume, "processed", "nested", "sample.webp")
	if info, err := os.Stat(webp); err != nil || info.Size() == 0 {
		t.Fatalf("expected converted WebP: %v", err)
	}
	if _, err := os.Stat(original); err != nil {
		t.Fatalf("original was altered: %v", err)
	}
	updated, err := helper.LoadManifest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Records[original].Status != "converted" {
		t.Errorf("status = %q", updated.Records[original].Status)
	}
}

func TestRunConvert_RemovesEmptyDiscardedFolder(t *testing.T) {
	volume := t.TempDir()
	discarded := filepath.Join(volume, "album", "discarded")
	if err := os.MkdirAll(discarded, 0755); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	if err := helper.SaveManifest(manifestPath, &helper.Manifest{Records: map[string]*helper.FileRecord{}}); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{VolumePath: volume, ProcessedFolder: filepath.Join(volume, "processed"), ManifestPath: manifestPath, Workers: 1}
	RunConvert(cfg)
	if _, err := os.Stat(discarded); !os.IsNotExist(err) {
		t.Error("empty discarded folder should be removed")
	}
}
