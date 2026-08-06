package storage

import (
	"path/filepath"
	"testing"
)

func TestAvailableSpace_ReturnsPositiveValueForExistingDirectory(t *testing.T) {
	available, err := AvailableSpace(t.TempDir())
	if err != nil {
		t.Fatalf("AvailableSpace returned an unexpected error: %v", err)
	}
	if available <= 0 {
		t.Fatalf("AvailableSpace = %d, want a positive value", available)
	}
}

func TestAvailableSpace_ReturnsErrorForMissingDirectory(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	if _, err := AvailableSpace(missing); err == nil {
		t.Fatal("AvailableSpace succeeded for a missing directory")
	}
}
