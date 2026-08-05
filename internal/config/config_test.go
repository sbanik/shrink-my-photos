package config

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

// Helper to reset command-line flags between tests
func resetFlags() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
}

func TestLoadConfig_Success(t *testing.T) {
	tempOut := t.TempDir()
	tempVol := t.TempDir()

	resetFlags()
	os.Args = []string{
		"cmd",
		"-mode=stage",
		"-volume=" + tempVol,
		"-out=" + tempOut,
		"-quality=85.5",
		"-workers=8",
		"-clean=true",
		"-types=png,jpg",
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("Expected LoadConfig to succeed, got error: %v", err)
	}

	if cfg.Mode != "stage" {
		t.Errorf("Expected Mode 'stage', got '%s'", cfg.Mode)
	}
	if cfg.VolumePath != tempVol {
		t.Errorf("Expected VolumePath '%s', got '%s'", tempVol, cfg.VolumePath)
	}
	if cfg.OutDir != tempOut {
		t.Errorf("Expected OutDir '%s', got '%s'", tempOut, cfg.OutDir)
	}
	if cfg.Quality != 85.5 {
		t.Errorf("Expected Quality 85.5, got %f", cfg.Quality)
	}
	if cfg.Workers != 8 {
		t.Errorf("Expected Workers 8, got %d", cfg.Workers)
	}
	if !cfg.Clean {
		t.Errorf("Expected Clean true, got %v", cfg.Clean)
	}
	if len(cfg.AllowedTypes) != 2 || cfg.AllowedTypes[0] != ".png" || cfg.AllowedTypes[1] != ".jpg" {
		t.Errorf("Expected AllowedTypes [.png .jpg], got %v", cfg.AllowedTypes)
	}

	expectedStaged := filepath.Join(tempOut, "to_process")
	if cfg.StagedFolder != expectedStaged {
		t.Errorf("Expected StagedFolder '%s', got '%s'", expectedStaged, cfg.StagedFolder)
	}
}

func TestLoadConfig_InvalidMode(t *testing.T) {
	tempOut := t.TempDir()

	resetFlags()
	os.Args = []string{
		"cmd",
		"-mode=invalid_mode",
		"-out=" + tempOut,
	}

	_, err := LoadConfig()
	if err == nil {
		t.Error("Expected error for invalid mode, got nil")
	}
}

func TestLoadConfig_MissingOutDir(t *testing.T) {
	resetFlags()
	os.Args = []string{
		"cmd",
		"-mode=stage",
	}

	_, err := LoadConfig()
	if err == nil {
		t.Error("Expected error when output directory is missing, got nil")
	}
}

func TestLoadConfig_MissingVolumeInStageMode(t *testing.T) {
	tempOut := t.TempDir()

	resetFlags()
	os.Args = []string{
		"cmd",
		"-mode=stage",
		"-out=" + tempOut,
	}

	_, err := LoadConfig()
	if err == nil {
		t.Error("Expected error when volume path is missing in stage mode, got nil")
	}
}

func TestEnvFallbackFunctions(t *testing.T) {
	t.Setenv("TEST_ENV_STR", "hello")
	t.Setenv("TEST_ENV_FLOAT", "92.5")
	t.Setenv("TEST_ENV_INT", "12")
	t.Setenv("TEST_ENV_BOOL", "true")

	if got := getEnvString("TEST_ENV_STR", "fallback"); got != "hello" {
		t.Errorf("Expected 'hello', got '%s'", got)
	}
	if got := getEnvFloat("TEST_ENV_FLOAT", 50.0); got != 92.5 {
		t.Errorf("Expected 92.5, got %f", got)
	}
	if got := getEnvInt("TEST_ENV_INT", 1); got != 12 {
		t.Errorf("Expected 12, got %d", got)
	}
	if got := getEnvBool("TEST_ENV_BOOL", false); !got {
		t.Errorf("Expected true, got %v", got)
	}

	// Test fallback defaults when key does not exist
	if got := getEnvString("NON_EXISTENT_KEY", "default"); got != "default" {
		t.Errorf("Expected 'default', got '%s'", got)
	}
}