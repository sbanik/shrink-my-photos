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

// Creates a dummy screenshot with 16:9 aspect ratio so detector.IsScreenshot succeeds
func createDummyScreenshot(t *testing.T, path string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1920, 1080))
	for x := 0; x < 100; x++ {
		for y := 0; y < 100; y++ {
			img.Set(x, y, color.RGBA{R: 50, G: 100, B: 150, A: 255})
		}
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		t.Fatalf("Failed to encode test image: %v", err)
	}
}

func TestRunStage_SuccessAndManifest(t *testing.T) {
	volDir := t.TempDir()
	outDir := t.TempDir()

	srcFile := filepath.Join(volDir, "screenshot.png")
	createDummyScreenshot(t, srcFile)

	stagedFolder := filepath.Join(outDir, "to_process")
	manifestPath := filepath.Join(outDir, "manifest.json")

	cfg := &config.Config{
		Mode:         "stage",
		VolumePath:   volDir,
		OutDir:       outDir,
		StagedFolder: stagedFolder,
		ManifestPath: manifestPath,
		Workers:      2,
		AllowedTypes: []string{".png"},
	}

	stagedCount := RunStage(cfg)
	if stagedCount != 1 {
		t.Fatalf("Expected 1 staged file, got %d", stagedCount)
	}

	manifest, err := helper.LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("Failed to load manifest: %v", err)
	}

	if len(manifest.Records) != 1 {
		t.Fatalf("Expected 1 record in manifest, got %d", len(manifest.Records))
	}

	stagedDest := filepath.Join(stagedFolder, "screenshot.png")
	record, exists := manifest.Records[stagedDest]
	if !exists {
		t.Fatalf("Manifest record for %s not found", stagedDest)
	}

	if record.OriginalPath != srcFile {
		t.Errorf("Expected OriginalPath %s, got %s", srcFile, record.OriginalPath)
	}
	if record.Status != "staged" {
		t.Errorf("Expected Status 'staged', got %s", record.Status)
	}
}

func TestRunStage_NameCollisionHandling(t *testing.T) {
	volDir := t.TempDir()
	outDir := t.TempDir()

	dirA := filepath.Join(volDir, "dirA")
	dirB := filepath.Join(volDir, "dirB")
	_ = os.MkdirAll(dirA, 0755)
	_ = os.MkdirAll(dirB, 0755)

	// Create two files with identical filenames in different directories
	fileA := filepath.Join(dirA, "image.png")
	fileB := filepath.Join(dirB, "image.png")
	createDummyScreenshot(t, fileA)
	createDummyScreenshot(t, fileB)

	stagedFolder := filepath.Join(outDir, "to_process")
	manifestPath := filepath.Join(outDir, "manifest.json")

	cfg := &config.Config{
		Mode:         "stage",
		VolumePath:   volDir,
		OutDir:       outDir,
		StagedFolder: stagedFolder,
		ManifestPath: manifestPath,
		Workers:      1, // Single worker to ensure deterministic collision ordering
		AllowedTypes: []string{".png"},
	}

	stagedCount := RunStage(cfg)
	if stagedCount != 2 {
		t.Fatalf("Expected 2 staged files, got %d", stagedCount)
	}

	originalName := filepath.Join(stagedFolder, "image.png")
	collidedName := filepath.Join(stagedFolder, "image_1.png")

	if _, err := os.Stat(originalName); os.IsNotExist(err) {
		t.Errorf("Expected %s to exist", originalName)
	}
	if _, err := os.Stat(collidedName); os.IsNotExist(err) {
		t.Errorf("Expected collision fallback %s to exist", collidedName)
	}
}

func TestGetUniqueDestination(t *testing.T) {
	tempDir := t.TempDir()

	// Initial file does not exist
	dest1 := getUniqueDestination(tempDir, "file.png")
	expected1 := filepath.Join(tempDir, "file.png")
	if dest1 != expected1 {
		t.Errorf("Expected %s, got %s", expected1, dest1)
	}

	// Create the file and check collision handling
	_ = os.WriteFile(expected1, []byte("data"), 0644)
	dest2 := getUniqueDestination(tempDir, "file.png")
	expected2 := filepath.Join(tempDir, "file_1.png")
	if dest2 != expected2 {
		t.Errorf("Expected %s, got %s", expected2, dest2)
	}

	// Create the collision file and check second collision
	_ = os.WriteFile(expected2, []byte("data"), 0644)
	dest3 := getUniqueDestination(tempDir, "file.png")
	expected3 := filepath.Join(tempDir, "file_2.png")
	if dest3 != expected3 {
		t.Errorf("Expected %s, got %s", expected3, dest3)
	}
}

func TestIsTypeAllowed(t *testing.T) {
	allowed := []string{".png", ".jpg", ".jpeg"}

	if !isTypeAllowed(".PNG", allowed) {
		t.Errorf("Expected uppercase .PNG to be allowed")
	}
	if !isTypeAllowed(".jpg", allowed) {
		t.Errorf("Expected .jpg to be allowed")
	}
	if isTypeAllowed(".gif", allowed) {
		t.Errorf("Expected .gif to be rejected")
	}
}