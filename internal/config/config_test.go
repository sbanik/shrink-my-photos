package config

import (
	"path/filepath"
	"testing"
)

func TestLoadConfig_Success(t *testing.T) {
	outDir := t.TempDir()
	volDir := t.TempDir()

	t.Setenv("MODE", "stage")
	t.Setenv("OUT_DIR", outDir)
	t.Setenv("VOLUME_PATH", volDir)
	t.Setenv("QUALITY", "85.5")
	t.Setenv("MIN_SAVINGS", "10.0") // 10% threshold
	t.Setenv("WORKERS", "8")
	t.Setenv("CLEAN_STAGED", "true")
	t.Setenv("ALLOWED_TYPES", "png, jpg, .jpeg, WEBP")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("Unexpected error loading config: %v", err)
	}

	if cfg.Mode != "stage" {
		t.Errorf("Expected Mode 'stage', got '%s'", cfg.Mode)
	}
	if cfg.VolumePath != volDir {
		t.Errorf("Expected VolumePath '%s', got '%s'", volDir, cfg.VolumePath)
	}
	if cfg.OutDir != outDir {
		t.Errorf("Expected OutDir '%s', got '%s'", outDir, cfg.OutDir)
	}
	if cfg.Quality != 85.5 {
		t.Errorf("Expected Quality 85.5, got %f", cfg.Quality)
	}
	if cfg.MinSavings != 0.10 {
		t.Errorf("Expected MinSavings ratio 0.10 (10%%), got %f", cfg.MinSavings)
	}
	if cfg.Workers != 8 {
		t.Errorf("Expected Workers 8, got %d", cfg.Workers)
	}
	if !cfg.Clean {
		t.Errorf("Expected Clean to be true")
	}

	expectedStaged := filepath.Join(outDir, "to_process")
	expectedDuplicates := filepath.Join(outDir, "to_process", "duplicates")
	expectedProcessed := filepath.Join(outDir, "processed")
	expectedManifest := filepath.Join(outDir, "manifest.json")
	expectedLog := filepath.Join(outDir, "shrinker.log")

	if cfg.StagedFolder != expectedStaged {
		t.Errorf("Expected StagedFolder '%s', got '%s'", expectedStaged, cfg.StagedFolder)
	}
	if cfg.DuplicatesFolder != expectedDuplicates {
		t.Errorf("Expected DuplicatesFolder '%s', got '%s'", expectedDuplicates, cfg.DuplicatesFolder)
	}
	if cfg.ProcessedFolder != expectedProcessed {
		t.Errorf("Expected ProcessedFolder '%s', got '%s'", expectedProcessed, cfg.ProcessedFolder)
	}
	if cfg.ManifestPath != expectedManifest {
		t.Errorf("Expected ManifestPath '%s', got '%s'", expectedManifest, cfg.ManifestPath)
	}
	if cfg.LogPath != expectedLog {
		t.Errorf("Expected LogPath '%s', got '%s'", expectedLog, cfg.LogPath)
	}

	expectedTypes := []string{".png", ".jpg", ".jpeg", ".webp"}
	if len(cfg.AllowedTypes) != len(expectedTypes) {
		t.Fatalf("Expected %d allowed types, got %d", len(expectedTypes), len(cfg.AllowedTypes))
	}
	for i, ext := range expectedTypes {
		if cfg.AllowedTypes[i] != ext {
			t.Errorf("Expected extension '%s' at index %d, got '%s'", ext, i, cfg.AllowedTypes[i])
		}
	}
}

func TestLoadConfig_DefaultMinSavings(t *testing.T) {
	outDir := t.TempDir()

	t.Setenv("MODE", "sync")
	t.Setenv("OUT_DIR", outDir)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("Unexpected error loading config: %v", err)
	}

	// Default MIN_SAVINGS is 10.0%, which converts to 0.10
	if cfg.MinSavings != 0.10 {
		t.Errorf("Expected default MinSavings 0.05 (5%%), got %f", cfg.MinSavings)
	}
}

func TestLoadConfig_InvalidMode(t *testing.T) {
	t.Setenv("MODE", "invalid_mode")
	t.Setenv("OUT_DIR", t.TempDir())

	_, err := LoadConfig()
	if err == nil {
		t.Error("Expected error for invalid mode, got nil")
	}
}

func TestLoadConfig_MissingOutDir(t *testing.T) {
	t.Setenv("MODE", "sync")
	t.Setenv("OUT_DIR", "")

	_, err := LoadConfig()
	if err == nil {
		t.Error("Expected error when OUT_DIR is empty, got nil")
	}
}

func TestLoadConfig_MissingVolumePathInStageMode(t *testing.T) {
	t.Setenv("MODE", "stage")
	t.Setenv("OUT_DIR", t.TempDir())
	t.Setenv("VOLUME_PATH", "")

	_, err := LoadConfig()
	if err == nil {
		t.Error("Expected error when VOLUME_PATH is missing in 'stage' mode, got nil")
	}
}