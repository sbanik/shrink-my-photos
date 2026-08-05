package helper

import (
	"encoding/json"
	"testing"
)

func TestFileRecord_JSONSerialization(t *testing.T) {
	record := FileRecord{
		OriginalPath: "/media/photos/img_1001.png",
		StagedPath:   "/out/screenshots/img_1001.png",
		WebPPath:     "/out/screenshots/img_1001.webp",
		OriginalSize: 2048000,
		WebPSize:     512000,
		Status:       "converted",
	}

	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("Failed to marshal FileRecord to JSON: %v", err)
	}

	var unmarshaled FileRecord
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal JSON into FileRecord: %v", err)
	}

	if unmarshaled.OriginalPath != record.OriginalPath {
		t.Errorf("OriginalPath mismatch: got %s, want %s", unmarshaled.OriginalPath, record.OriginalPath)
	}
	if unmarshaled.StagedPath != record.StagedPath {
		t.Errorf("StagedPath mismatch: got %s, want %s", unmarshaled.StagedPath, record.StagedPath)
	}
	if unmarshaled.WebPPath != record.WebPPath {
		t.Errorf("WebPPath mismatch: got %s, want %s", unmarshaled.WebPPath, record.WebPPath)
	}
	if unmarshaled.OriginalSize != record.OriginalSize {
		t.Errorf("OriginalSize mismatch: got %d, want %d", unmarshaled.OriginalSize, record.OriginalSize)
	}
	if unmarshaled.WebPSize != record.WebPSize {
		t.Errorf("WebPSize mismatch: got %d, want %d", unmarshaled.WebPSize, record.WebPSize)
	}
	if unmarshaled.Status != record.Status {
		t.Errorf("Status mismatch: got %s, want %s", unmarshaled.Status, record.Status)
	}
}

func TestManifest_JSONSerialization(t *testing.T) {
	manifest := Manifest{
		Records: map[string]*FileRecord{
			"/out/screenshots/shot1.png": {
				OriginalPath: "/src/shot1.png",
				StagedPath:   "/out/screenshots/shot1.png",
				WebPPath:     "/out/screenshots/shot1.webp",
				OriginalSize: 1000,
				WebPSize:     400,
				Status:       "staged",
			},
		},
	}

	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("Failed to marshal Manifest to JSON: %v", err)
	}

	var unmarshaled Manifest
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal JSON into Manifest: %v", err)
	}

	rec, exists := unmarshaled.Records["/out/screenshots/shot1.png"]
	if !exists {
		t.Fatalf("Expected key '/out/screenshots/shot1.png' missing from unmarshaled manifest map")
	}

	if rec.Status != "staged" {
		t.Errorf("Status mismatch: got %s, want staged", rec.Status)
	}
	if rec.OriginalSize != 1000 {
		t.Errorf("OriginalSize mismatch: got %d, want 1000", rec.OriginalSize)
	}
}

func TestManifest_EmptyRecordsMap(t *testing.T) {
	manifest := Manifest{
		Records: make(map[string]*FileRecord),
	}

	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("Failed to marshal empty Manifest: %v", err)
	}

	var unmarshaled Manifest
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal empty Manifest: %v", err)
	}

	if unmarshaled.Records == nil {
		t.Errorf("Expected Records map to be non-nil after unmarshaling")
	}
	if len(unmarshaled.Records) != 0 {
		t.Errorf("Expected empty Records map, got len = %d", len(unmarshaled.Records))
	}
}