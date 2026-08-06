package processor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sbanik/shrink-my-photos/internal/helper"
)

func TestRunDeleteOriginals_SuccessAndStatusFilter(t *testing.T) {
	tempDir := t.TempDir()
	manifestPath := filepath.Join(tempDir, "manifest.json")

	// 1. Create dummy original files
	original1 := filepath.Join(tempDir, "orig_1.png")
	original2 := filepath.Join(tempDir, "orig_2.png")
	original3 := filepath.Join(tempDir, "orig_3.png")

	_ = os.WriteFile(original1, []byte("data1"), 0644)
	_ = os.WriteFile(original2, []byte("data2"), 0644)
	_ = os.WriteFile(original3, []byte("data3"), 0644)

	// 2. Prepare a manifest with various statuses
	manifest := &helper.Manifest{
		Records: map[string]*helper.FileRecord{
			"key1": {
				OriginalPath: original1,
				Status:       "converted", // Should be deleted
			},
			"key2": {
				OriginalPath: original2,
				Status:       "staged", // Should be skipped
			},
			"key3": {
				OriginalPath: original3,
				Status:       "converted", // Should be deleted
			},
		},
	}

	if err := helper.SaveManifest(manifestPath, manifest); err != nil {
		t.Fatalf("Failed to write initial manifest: %v", err)
	}

	// 3. Execute cleanup
	RunDeleteOriginals(manifestPath)

	// 4. Verify file deletion on disk
	if _, err := os.Stat(original1); !os.IsNotExist(err) {
		t.Errorf("Expected %s to be deleted, but it still exists", original1)
	}

	if _, err := os.Stat(original2); os.IsNotExist(err) {
		t.Errorf("Expected %s to remain untouched (staged), but it was deleted", original2)
	}

	if _, err := os.Stat(original3); !os.IsNotExist(err) {
		t.Errorf("Expected %s to be deleted, but it still exists", original3)
	}

	// 5. Verify updated manifest record statuses
	updatedManifest, err := helper.LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("Failed to reload updated manifest: %v", err)
	}

	if updatedManifest.Records["key1"].Status != "completed" {
		t.Errorf("Expected status 'completed' for key1, got '%s'", updatedManifest.Records["key1"].Status)
	}
	if updatedManifest.Records["key2"].Status != "staged" {
		t.Errorf("Expected status 'staged' for key2, got '%s'", updatedManifest.Records["key2"].Status)
	}
	if updatedManifest.Records["key3"].Status != "completed" {
		t.Errorf("Expected status 'completed' for key3, got '%s'", updatedManifest.Records["key3"].Status)
	}
}

func TestRunDeleteOriginals_AlreadyMissingFile(t *testing.T) {
	tempDir := t.TempDir()
	manifestPath := filepath.Join(tempDir, "manifest.json")

	missingOriginal := filepath.Join(tempDir, "already_deleted.png")

	manifest := &helper.Manifest{
		Records: map[string]*helper.FileRecord{
			"key_missing": {
				OriginalPath: missingOriginal,
				Status:       "converted",
			},
		},
	}

	if err := helper.SaveManifest(manifestPath, manifest); err != nil {
		t.Fatalf("Failed to write manifest: %v", err)
	}

	// Execution should treat non-existent original files gracefully without failing
	RunDeleteOriginals(manifestPath)

	updatedManifest, err := helper.LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("Failed to reload manifest: %v", err)
	}

	if updatedManifest.Records["key_missing"].Status != "completed" {
		t.Errorf("Expected status 'completed' for missing file record, got '%s'", updatedManifest.Records["key_missing"].Status)
	}
}

func TestRunDeleteOriginals_NoConvertedRecords(t *testing.T) {
	tempDir := t.TempDir()
	manifestPath := filepath.Join(tempDir, "manifest.json")

	manifest := &helper.Manifest{
		Records: map[string]*helper.FileRecord{
			"key1": {
				OriginalPath: "/dummy/path.png",
				Status:       "staged",
			},
		},
	}

	if err := helper.SaveManifest(manifestPath, manifest); err != nil {
		t.Fatalf("Failed to write manifest: %v", err)
	}

	// Should exit early gracefully without error
	RunDeleteOriginals(manifestPath)
}
