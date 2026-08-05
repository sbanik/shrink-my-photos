package config

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

// Helper to reset flag state between tests
func resetFlags() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
}

func TestLoadConfig_MissingOutDir(t *testing.T) {
	resetFlags()
	os.Unsetenv("OUT_DIR")
	os.Unsetenv("VOLUME_PATH")
	os.Args = []string{"cmd"}

	_, err := LoadConfig()
	if err == nil {
		t.Errorf("Expected error when OUT_DIR and -out flag are missing, got nil")
	}
}

func TestLoadConfig_MissingVolumePathInNormalMode(t *testing.T) {
	resetFlags()
	tempOut := t.TempDir()
	os.Unsetenv("VOLUME_PATH")
	os.Args = []string{"cmd", "-out", tempOut}

	_, err := LoadConfig()
	if err == nil {
		t.Errorf("Expected error when VOLUME_PATH and -volume are missing, got nil")
	}
}

func TestLoadConfig_SuccessNormalMode(t *testing.T) {
	resetFlags()
	tempVol := t.TempDir()
	tempOut := t.TempDir()

	os.Args = []string{"cmd", "-volume", tempVol, "-out", tempOut, "-quality", "75", "-workers", "4"}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("Unexpected config loading error: %v", err)
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
	if cfg.DeleteOriginals {
		t.Errorf("Expected DeleteOriginals false, got true")
	}

	expectedStaged := filepath.Join(tempOut, "screenshots")
	if cfg.StagedFolder != expectedStaged {
		t.Errorf("Expected StagedFolder %s, got %s", expectedStaged, cfg.StagedFolder)
	}
}

func TestLoadConfig_DeleteOriginalsModeWithoutVolume(t *testing.T) {
	resetFlags()
	tempOut := t.TempDir()

	// In -delete-originals mode, -volume path is not mandatory
	os.Args = []string{"cmd", "-out", tempOut, "-delete-originals"}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig should succeed in delete-originals mode without volume path: %v", err)
	}

	if !cfg.DeleteOriginals {
		t.Errorf("Expected DeleteOriginals true, got false")
	}
}

func TestLoadConfig_EnvFallback(t *testing.T) {
	resetFlags()
	tempVol := t.TempDir()
	tempOut := t.TempDir()

	os.Setenv("VOLUME_PATH", tempVol)
	os.Setenv("OUT_DIR", tempOut)
	os.Setenv("QUALITY", "90")
	defer func() {
		os.Unsetenv("VOLUME_PATH")
		os.Unsetenv("OUT_DIR")
		os.Unsetenv("QUALITY")
	}()

	os.Args = []string{"cmd"} // No flags passed

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config from env variables: %v", err)
	}

	if cfg.VolumePath != tempVol {
		t.Errorf("Expected VolumePath from ENV %s, got %s", tempVol, cfg.VolumePath)
	}
	if cfg.Quality != 90.0 {
		t.Errorf("Expected Quality from ENV 90.0, got %f", cfg.Quality)
	}
}