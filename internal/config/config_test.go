package config

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

// Helper to reset flag state between individual test executions
func resetFlags() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
}

func TestLoadConfig_MissingOutDir(t *testing.T) {
	resetFlags()
	os.Unsetenv("OUT_DIR")
	os.Unsetenv("VOLUME_PATH")
	os.Unsetenv("MODE")
	os.Args = []string{"cmd"}

	_, err := LoadConfig()
	if err == nil {
		t.Errorf("Expected error when OUT_DIR and -out flag are missing, got nil")
	}
}

func TestLoadConfig_InvalidMode(t *testing.T) {
	resetFlags()
	tempOut := t.TempDir()
	os.Args = []string{"cmd", "-out", tempOut, "-mode", "invalid_mode"}

	_, err := LoadConfig()
	if err == nil {
		t.Errorf("Expected error for invalid mode 'invalid_mode', got nil")
	}
}

func TestLoadConfig_MissingVolumeInStageMode(t *testing.T) {
	resetFlags()
	tempOut := t.TempDir()
	os.Unsetenv("VOLUME_PATH")
	os.Args = []string{"cmd", "-out", tempOut, "-mode", "stage"}

	_, err := LoadConfig()
	if err == nil {
		t.Errorf("Expected error when volume is missing in 'stage' mode, got nil")
	}
}

func TestLoadConfig_SuccessAllMode(t *testing.T) {
	resetFlags()
	tempVol := t.TempDir()
	tempOut := t.TempDir()

	os.Args = []string{"cmd", "-volume", tempVol, "-out", tempOut, "-quality", "75", "-workers", "4"}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("Unexpected error loading config: %v", err)
	}

	if cfg.Mode != "stage" {
		t.Errorf("Expected default mode 'stage', got '%s'", cfg.Mode)
	}
	if cfg.VolumePath != tempVol {
		t.Errorf("Expected VolumePath %s, got %s", tempVol, cfg.VolumePath)
	}
	if cfg.OutDir != tempOut {
		t.Errorf("Expected OutDir %s, got %s", tempOut, cfg.OutDir)
	}
	if cfg.Quality != 75.0 {
		t.Errorf("Expected Quality 75.0, got %f", cfg.Quality)
	}
	if cfg.Workers != 4 {
		t.Errorf("Expected Workers 4, got %d", cfg.Workers)
	}

	expectedStaged := filepath.Join(tempOut, "screenshots")
	if cfg.StagedFolder != expectedStaged {
		t.Errorf("Expected StagedFolder %s, got %s", expectedStaged, cfg.StagedFolder)
	}
}

func TestLoadConfig_DeleteModeWithoutVolumePath(t *testing.T) {
	resetFlags()
	tempOut := t.TempDir()

	// In 'delete' or 'convert' mode, volume path is optional
	os.Args = []string{"cmd", "-out", tempOut, "-mode", "delete"}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig should succeed in delete mode without volume path: %v", err)
	}

	if cfg.Mode != "delete" {
		t.Errorf("Expected mode 'delete', got '%s'", cfg.Mode)
	}
}

func TestLoadConfig_EnvFallback(t *testing.T) {
	resetFlags()
	tempVol := t.TempDir()
	tempOut := t.TempDir()

	os.Setenv("VOLUME_PATH", tempVol)
	os.Setenv("OUT_DIR", tempOut)
	os.Setenv("MODE", "CONVERT")
	os.Setenv("QUALITY", "90")
	defer func() {
		os.Unsetenv("VOLUME_PATH")
		os.Unsetenv("OUT_DIR")
		os.Unsetenv("MODE")
		os.Unsetenv("QUALITY")
	}()

	os.Args = []string{"cmd"}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config from environment variables: %v", err)
	}

	if cfg.Mode != "convert" {
		t.Errorf("Expected normalized mode 'convert', got '%s'", cfg.Mode)
	}
	if cfg.VolumePath != tempVol {
		t.Errorf("Expected VolumePath from ENV %s, got %s", tempVol, cfg.VolumePath)
	}
	if cfg.Quality != 90.0 {
		t.Errorf("Expected Quality from ENV 90.0, got %f", cfg.Quality)
	}
}