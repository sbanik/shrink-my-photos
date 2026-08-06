package processor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sbanik/shrink-my-photos/internal/config"
	"github.com/sbanik/shrink-my-photos/internal/helper"
)

func TestIsHidden(t *testing.T) {
	if !isHidden(".DS_Store") || isHidden("image.png") {
		t.Fatal("hidden-file detection is incorrect")
	}
}

func TestIsAllowedExtension(t *testing.T) {
	if !isAllowedExtension("photo.PNG", []string{".png"}) || isAllowedExtension("photo.heic", []string{".png"}) {
		t.Fatal("extension matching is incorrect")
	}
}

func TestRunStage_RecursivelyTracksAndMovesDuplicates(t *testing.T) {
	volume := t.TempDir()
	rootImage := filepath.Join(volume, "root.png")
	nestedImage := filepath.Join(volume, "album", "duplicate.png")
	uniqueImage := filepath.Join(volume, "album", "unique.jpg")
	if err := os.MkdirAll(filepath.Dir(nestedImage), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rootImage, []byte("same bytes"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(nestedImage, []byte("same bytes"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(uniqueImage, []byte("unique bytes"), 0644); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(t.TempDir(), "state", "manifest.json")
	cfg := &config.Config{VolumePath: volume, ManifestPath: manifestPath, AllowedTypes: []string{".png", ".jpg"}}

	if result := RunStage(cfg); result.Pending != 2 {
		t.Fatalf("pending = %d, want 2", result.Pending)
	}
	manifest, err := helper.LoadManifest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var discardedCount int
	for original, record := range manifest.Records {
		if record.Status != "discarded" {
			continue
		}
		discardedCount++
		if _, err := os.Stat(original); !os.IsNotExist(err) {
			t.Fatalf("duplicate %s was not moved", original)
		}
		if _, err := os.Stat(record.DiscardedPath); err != nil {
			t.Fatalf("discarded duplicate missing: %v", err)
		}
	}
	if discardedCount != 1 {
		t.Fatalf("discarded records = %d, want 1", discardedCount)
	}
}

func TestMoveToDiscarded_AvoidsNameCollision(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "image.png")
	if err := os.WriteFile(source, []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "discarded"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "discarded", "image.png"), []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	destination, err := moveToDiscarded(source)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(destination) != "image-1.png" {
		t.Errorf("destination = %s", destination)
	}
}

func TestRunStage_ListsAndDeletesHiddenFilesOnlyOnRequest(t *testing.T) {
	volume := t.TempDir()
	hidden := filepath.Join(volume, ".IMG_1234.JPG")
	if err := os.WriteFile(hidden, []byte("metadata"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{VolumePath: volume, ManifestPath: filepath.Join(t.TempDir(), "manifest.json"), AllowedTypes: []string{".jpg"}}
	result := RunStage(cfg)
	if len(result.HiddenFiles) != 1 || result.HiddenFiles[0] != hidden {
		t.Fatalf("hidden files = %v", result.HiddenFiles)
	}
	if _, err := os.Stat(hidden); err != nil {
		t.Fatal("stage should not delete hidden files")
	}
	if deleted := DeleteHiddenFiles(cfg.ManifestPath, result.HiddenFiles); deleted != 1 {
		t.Fatalf("deleted = %d", deleted)
	}
	if _, err := os.Stat(hidden); !os.IsNotExist(err) {
		t.Fatal("hidden file was not deleted after approval")
	}
	manifest, err := helper.LoadManifest(cfg.ManifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.HiddenFiles[hidden].Status != "deleted" {
		t.Fatal("hidden-file manifest record was not updated")
	}
}
