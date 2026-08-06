package processor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sbanik/shrink-my-photos/internal/config"
	"github.com/sbanik/shrink-my-photos/internal/helper"
)

func TestRequiredConversionSpace(t *testing.T) {
	volume := t.TempDir()
	first := filepath.Join(volume, "first.png")
	second := filepath.Join(volume, "second.png")
	if err := os.WriteFile(first, []byte("one"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte("two"), 0644); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	manifest := &helper.Manifest{Records: map[string]*helper.FileRecord{
		first:  {OriginalPath: first, Status: "pending"},
		second: {OriginalPath: second, Status: "pending"},
	}}
	if err := helper.SaveManifest(manifestPath, manifest); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{ManifestPath: manifestPath, TargetSize: 100}
	required, count, err := RequiredConversionSpace(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 || required != 220 {
		t.Fatalf("required/count = %d/%d, want 220/2", required, count)
	}
}

func TestValidateFallbackWorkspaceRequiresEmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	if _, err := ValidateFallbackWorkspace(dir, 0); err != nil {
		t.Fatalf("empty fallback rejected: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "existing.webp"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateFallbackWorkspace(dir, 0); err == nil {
		t.Fatal("non-empty fallback accepted")
	}
}

func TestRunFinalizeMovesWebPBackToOriginalFolder(t *testing.T) {
	volume := t.TempDir()
	processed := filepath.Join(t.TempDir(), "processed")
	if err := os.MkdirAll(filepath.Join(processed, "album"), 0755); err != nil {
		t.Fatal(err)
	}
	original := filepath.Join(volume, "album", "photo.jpg")
	webp := filepath.Join(processed, "album", "photo.webp")
	if err := os.WriteFile(webp, []byte("webp"), 0644); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	manifest := &helper.Manifest{Records: map[string]*helper.FileRecord{original: {OriginalPath: original, WebPPath: webp, Status: "completed"}}}
	if err := helper.SaveManifest(manifestPath, manifest); err != nil {
		t.Fatal(err)
	}

	RunFinalize(&config.Config{ManifestPath: manifestPath, ProcessedFolder: processed})
	target := filepath.Join(volume, "album", "photo.webp")
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("finalized WebP missing: %v", err)
	}
	updated, err := helper.LoadManifest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Records[original].Status != "finalized" {
		t.Fatalf("status = %q", updated.Records[original].Status)
	}
}
