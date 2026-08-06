package config

import (
	"path/filepath"
	"testing"
)

func TestLoadConfig_UsesSourceVolumeAndStateDirectory(t *testing.T) {
	volume := t.TempDir()
	state := t.TempDir()
	t.Setenv("MODE", "all")
	t.Setenv("VOLUME_PATH", volume)
	t.Setenv("STATE_DIR", state)
	t.Setenv("QUALITY", "85")
	t.Setenv("TARGET_SIZE_KB", "275")
	t.Setenv("SMALL_FILE_SIZE_KB", "150")
	t.Setenv("WORKERS", "2")
	t.Setenv("ALLOWED_TYPES", "png, JPG, .jpeg")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.VolumePath != volume {
		t.Fatalf("VolumePath = %q, want %q", cfg.VolumePath, volume)
	}
	if cfg.ProcessedFolder != filepath.Join(volume, "processed") {
		t.Errorf("unexpected processed path: %s", cfg.ProcessedFolder)
	}
	if filepath.Dir(filepath.Dir(cfg.ManifestPath)) != state {
		t.Errorf("manifest not stored below state dir: %s", cfg.ManifestPath)
	}
	if cfg.TargetSize != 275*1024 || cfg.SmallFileSize != 150*1024 {
		t.Errorf("unexpected size configuration: %d, %d", cfg.TargetSize, cfg.SmallFileSize)
	}
	if cfg.Quality != 85 || cfg.Workers != 2 {
		t.Errorf("unexpected quality/workers: %.0f/%d", cfg.Quality, cfg.Workers)
	}
	wantTypes := []string{".png", ".jpg", ".jpeg"}
	for i, want := range wantTypes {
		if cfg.AllowedTypes[i] != want {
			t.Errorf("type %d = %q, want %q", i, cfg.AllowedTypes[i], want)
		}
	}
}

func TestLoadConfig_RejectsInvalidQuality(t *testing.T) {
	t.Setenv("VOLUME_PATH", t.TempDir())
	t.Setenv("STATE_DIR", t.TempDir())
	t.Setenv("QUALITY", "49")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("expected quality validation error")
	}
}

func TestLoadConfig_EnforcesQualityFloor(t *testing.T) {
	t.Setenv("VOLUME_PATH", t.TempDir())
	t.Setenv("STATE_DIR", t.TempDir())
	t.Setenv("QUALITY", "50")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("expected 55 quality floor validation error")
	}
}

func TestLoadConfig_RequiresVolume(t *testing.T) {
	t.Setenv("VOLUME_PATH", "")
	t.Setenv("STATE_DIR", t.TempDir())
	if _, err := LoadConfig(); err == nil {
		t.Fatal("expected missing-volume error")
	}
}

func TestLoadConfig_DuplicatesModeUsesSelectedFoldersWithoutVolume(t *testing.T) {
	first := t.TempDir()
	second := t.TempDir()
	t.Setenv("MODE", "duplicates")
	t.Setenv("VOLUME_PATH", "")
	t.Setenv("MULTI_FOLDER", first+","+second)
	t.Setenv("STATE_DIR", t.TempDir())
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.DuplicateFolders) != 2 || cfg.DuplicateFolders[0] != first || cfg.DuplicateFolders[1] != second {
		t.Fatalf("DuplicateFolders = %v", cfg.DuplicateFolders)
	}
	if cfg.VolumePath != first {
		t.Fatalf("VolumePath = %q, want %q", cfg.VolumePath, first)
	}
}

func TestLoadConfig_RecognizesDeletionAndExternalOutputFlags(t *testing.T) {
	volume := t.TempDir()
	processed := t.TempDir()
	t.Setenv("VOLUME_PATH", volume)
	t.Setenv("STATE_DIR", t.TempDir())
	t.Setenv("PROCESSED_PATH", processed)
	t.Setenv("DELETE_ORIGINALS", "true")
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.DeleteOriginals {
		t.Fatal("DELETE_ORIGINALS was not enabled")
	}
	if !cfg.ProcessedPathProvided || cfg.ProcessedFolder != processed {
		t.Fatalf("external processed path not retained: %s", cfg.ProcessedFolder)
	}
}
