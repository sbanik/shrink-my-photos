package processor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sbanik/shrink-my-photos/internal/config"
	"github.com/sbanik/shrink-my-photos/internal/helper"
)

// --- Helper Functions Unit Tests ---

func TestIsHidden(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		{".DS_Store", true},
		{".git", true},
		{"._photo.png", true},
		{"photo.png", false},
		{"subfolder", false},
	}

	for _, tt := range tests {
		if got := isHidden(tt.filename); got != tt.expected {
			t.Errorf("isHidden(%q) = %v; want %v", tt.filename, got, tt.expected)
		}
	}
}

func TestIsAllowedExtension(t *testing.T) {
	allowed := []string{".png", ".jpg", ".jpeg"}

	tests := []struct {
		path     string
		expected bool
	}{
		{"/path/to/image.png", true},
		{"/path/to/image.PNG", true},
		{"/path/to/image.jpeg", true},
		{"/path/to/doc.pdf", false},
		{"/path/to/file_no_ext", false},
	}

	for _, tt := range tests {
		if got := isAllowedExtension(tt.path, allowed); got != tt.expected {
			t.Errorf("isAllowedExtension(%q) = %v; want %v", tt.path, got, tt.expected)
		}
	}
}

func TestCalculateSHA256(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.txt")
	content := []byte("hello world")

	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	hash, err := calculateSHA256(filePath)
	if err != nil {
		t.Fatalf("calculateSHA256 returned unexpected error: %v", err)
	}

	// SHA-256 for "hello world"
	expectedHash := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if hash != expectedHash {
		t.Errorf("calculateSHA256 = %s; want %s", hash, expectedHash)
	}
}

func TestCreateRequiredDirs(t *testing.T) {
	tempDir := t.TempDir()
	stagedDir := filepath.Join(tempDir, "to_process")
	dupDir := filepath.Join(stagedDir, "duplicates")

	if err := createRequiredDirs(stagedDir, dupDir); err != nil {
		t.Fatalf("createRequiredDirs returned unexpected error: %v", err)
	}

	if _, err := os.Stat(stagedDir); os.IsNotExist(err) {
		t.Errorf("Expected staged dir %s to exist", stagedDir)
	}
	if _, err := os.Stat(dupDir); os.IsNotExist(err) {
		t.Errorf("Expected duplicates dir %s to exist", dupDir)
	}
}

func TestCountCandidateFiles(t *testing.T) {
	volDir := t.TempDir()

	_ = os.WriteFile(filepath.Join(volDir, "file1.png"), []byte("data1"), 0644)
	_ = os.WriteFile(filepath.Join(volDir, "file2.jpg"), []byte("data2"), 0644)
	_ = os.WriteFile(filepath.Join(volDir, ".DS_Store"), []byte("hidden"), 0644)

	hiddenSubdir := filepath.Join(volDir, ".hiddenDir")
	_ = os.MkdirAll(hiddenSubdir, 0755)
	_ = os.WriteFile(filepath.Join(hiddenSubdir, "ignored.png"), []byte("data3"), 0644)

	count := countCandidateFiles(volDir)
	if count != 2 {
		t.Errorf("countCandidateFiles = %d; want 2", count)
	}
}

func TestPrepareManifest(t *testing.T) {
	outDir := t.TempDir()
	stagedDir := filepath.Join(outDir, "to_process")
	manifestPath := filepath.Join(outDir, "manifest.json")

	_ = os.MkdirAll(stagedDir, 0755)
	_ = os.WriteFile(filepath.Join(stagedDir, "old.png"), []byte("old"), 0644)

	cfg := &config.Config{
		OutDir:       outDir,
		StagedFolder: stagedDir,
		ManifestPath: manifestPath,
		Clean:        true,
	}

	manifest := prepareManifest(cfg)
	if len(manifest.Records) != 0 {
		t.Errorf("Expected empty manifest records when clean=true")
	}

	if _, err := os.Stat(stagedDir); !os.IsNotExist(err) {
		t.Errorf("Expected stagedFolder %s to be wiped when clean=true", stagedDir)
	}
}

func TestProcessFile_StagedAndDuplicate(t *testing.T) {
	tempDir := t.TempDir()
	stagedFolder := filepath.Join(tempDir, "to_process")
	dupFolder := filepath.Join(stagedFolder, "duplicates")
	_ = os.MkdirAll(dupFolder, 0755)

	srcFile1 := filepath.Join(tempDir, "img1.png")
	srcFile2 := filepath.Join(tempDir, "img2.png")
	content := []byte("identical image bytes")

	_ = os.WriteFile(srcFile1, content, 0644)
	_ = os.WriteFile(srcFile2, content, 0644)

	info1, _ := os.Stat(srcFile1)
	info2, _ := os.Stat(srcFile2)

	cfg := &config.Config{
		StagedFolder:     stagedFolder,
		DuplicatesFolder: dupFolder,
	}
	manifest := &helper.Manifest{Records: make(map[string]*helper.FileRecord)}
	seenHashes := make(map[string]string)

	// Process first file -> should stage
	action1 := processFile(srcFile1, "img1.png", info1, cfg, manifest, seenHashes)
	if action1 != actionStaged {
		t.Errorf("Expected processFile for img1.png to return actionStaged, got %v", action1)
	}

	// Process second file with identical content -> should route to duplicates
	action2 := processFile(srcFile2, "img2.png", info2, cfg, manifest, seenHashes)
	if action2 != actionDuplicate {
		t.Errorf("Expected processFile for img2.png to return actionDuplicate, got %v", action2)
	}

	if _, err := os.Stat(filepath.Join(dupFolder, "img2.png")); os.IsNotExist(err) {
		t.Errorf("Expected duplicate file to exist in duplicates folder")
	}
}

// --- Integration / Full Stage Pipeline Test ---

func TestRunStage_FullPipeline(t *testing.T) {
	volDir := t.TempDir()
	outDir := t.TempDir()

	stagedFolder := filepath.Join(outDir, "to_process")
	duplicatesFolder := filepath.Join(stagedFolder, "duplicates")
	manifestPath := filepath.Join(outDir, "manifest.json")

	// 1. Create original test images
	file1Path := filepath.Join(volDir, "photo1.png")
	file1Content := []byte("unique image content 1")
	_ = os.WriteFile(file1Path, file1Content, 0644)

	// 2. Duplicate image
	file2Path := filepath.Join(volDir, "photo1_copy.png")
	_ = os.WriteFile(file2Path, file1Content, 0644)

	// 3. Hidden macOS file (should be ignored)
	_ = os.WriteFile(filepath.Join(volDir, ".DS_Store"), []byte("mac OS metadata"), 0644)

	cfg := &config.Config{
		VolumePath:       volDir,
		OutDir:           outDir,
		StagedFolder:     stagedFolder,
		DuplicatesFolder: duplicatesFolder,
		ManifestPath:     manifestPath,
		AllowedTypes:     []string{".png"},
		Clean:            false,
	}

	stagedCount := RunStage(cfg)

	if stagedCount != 1 {
		t.Errorf("Expected 1 staged file, got %d", stagedCount)
	}

	if _, err := os.Stat(filepath.Join(duplicatesFolder, "photo1_copy.png")); os.IsNotExist(err) {
		t.Errorf("Expected duplicate to be routed to duplicates folder")
	}
}