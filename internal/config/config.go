package config

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Mode         string // "all", "stage", "convert", "delete"
	VolumePath   string
	OutDir       string
	Quality      float64
	Workers      int
	StagedFolder string
	ManifestPath string
	LogPath      string
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	envVolume := os.Getenv("VOLUME_PATH")
	envOut := os.Getenv("OUT_DIR")
	envMode := os.Getenv("MODE")
	if envMode == "" {
		envMode = "stage" // Default mode: stage
	}

	envQuality := 80.0
	if q, err := strconv.ParseFloat(os.Getenv("QUALITY"), 64); err == nil && q > 0 {
		envQuality = q
	}

	envWorkers := runtime.NumCPU()
	if w, err := strconv.Atoi(os.Getenv("WORKERS")); err == nil && w > 0 {
		envWorkers = w
	}

	mode := flag.String("mode", envMode, "Execution mode: 'all' (stage+convert), 'stage' (detect+copy only), 'convert' (convert staged images), 'delete' (remove originals)")
	volumePath := flag.String("volume", envVolume, "Source volume path to scan")
	outDir := flag.String("out", envOut, "Output directory path for screenshots")
	quality := flag.Float64("quality", envQuality, "WebP compression quality (0-100)")
	workers := flag.Int("workers", envWorkers, "Number of parallel workers")

	flag.Parse()

	normalizedMode := strings.ToLower(*mode)
	validModes := map[string]bool{"all": true, "stage": true, "convert": true, "delete": true}
	if !validModes[normalizedMode] {
		return nil, fmt.Errorf("invalid mode '%s'. Valid modes are: 'all', 'stage', 'convert', 'delete'", *mode)
	}

	if *outDir == "" {
		return nil, fmt.Errorf("output directory path is required (-out or OUT_DIR)")
	}

	// Volume path is required only if scanning is needed
	if (normalizedMode == "all" || normalizedMode == "stage") && *volumePath == "" {
		return nil, fmt.Errorf("source volume path is required for mode '%s' (-volume or VOLUME_PATH)", normalizedMode)
	}

	stagedFolder := fmt.Sprintf("%s/screenshots", *outDir)
	manifestPath := fmt.Sprintf("%s/manifest.json", *outDir)
	logPath := fmt.Sprintf("%s/error.log", *outDir)

	return &Config{
		Mode:         normalizedMode,
		VolumePath:   *volumePath,
		OutDir:       *outDir,
		Quality:      *quality,
		Workers:      *workers,
		StagedFolder: stagedFolder,
		ManifestPath: manifestPath,
		LogPath:      logPath,
	}, nil
}