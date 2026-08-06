package processor

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sbanik/shrink-my-photos/internal/config"
	"github.com/sbanik/shrink-my-photos/internal/helper"
	"github.com/sbanik/shrink-my-photos/internal/storage"
)

// RequiredConversionSpace returns a conservative output-space estimate for all
// pending records. The 10% reserve covers filesystem and encoder overhead.
func RequiredConversionSpace(cfg *config.Config) (int64, int, error) {
	manifest, err := helper.LoadManifest(cfg.ManifestPath)
	if err != nil {
		return 0, 0, err
	}
	var required int64
	count := 0
	for _, record := range manifest.Records {
		if record.Status != "pending" {
			continue
		}
		if _, err := os.Stat(record.OriginalPath); err != nil {
			continue
		}
		required += cfg.TargetSize
		count++
	}
	return required + required/10, count, nil
}

func AvailableWorkspaceSpace(path string) (int64, error) {
	if err := os.MkdirAll(path, 0755); err != nil {
		return 0, err
	}
	return storage.AvailableSpace(path)
}

// ValidateFallbackWorkspace accepts only an existing, empty directory with
// enough available space so a user-selected location cannot be overwritten.
func ValidateFallbackWorkspace(path string, required int64) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("read fallback directory: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("fallback path is not a directory")
	}
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return "", fmt.Errorf("read fallback directory: %w", err)
	}
	if len(entries) != 0 {
		return "", fmt.Errorf("fallback directory must be empty")
	}
	available, err := storage.AvailableSpace(absPath)
	if err != nil {
		return "", fmt.Errorf("check fallback free space: %w", err)
	}
	if available < required {
		return "", fmt.Errorf("fallback has %s available; %s is required", helper.FormatBytes(available), helper.FormatBytes(required))
	}
	return absPath, nil
}
