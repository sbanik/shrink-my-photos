package config

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	VolumePath      string
	OutDir          string
	Quality         float64
	Workers         int
	DeleteOriginals bool
	StagedFolder    string
	ManifestPath    string
	LogPath         string
}

func LoadConfig() (*Config, error) {
	// 1. Load .env file silently if present
	_ = godotenv.Load()

	// 2. Fetch raw environment variable values
	envVolume := os.Getenv("VOLUME_PATH")
	envOut := os.Getenv("OUT_DIR")
	
	envQuality := 80.0
	if q, err := strconv.ParseFloat(os.Getenv("QUALITY"), 64); err == nil && q > 0 {
		envQuality = q
	}

	envWorkers := runtime.NumCPU()
	if w, err := strconv.Atoi(os.Getenv("WORKERS")); err == nil && w > 0 {
		envWorkers = w
	}

	deleteOriginalFlag := false
	if opt, err :=strconv.ParseBool(os.Getenv("DELETE_ORIGINALS")); err == nil {
		deleteOriginalFlag = opt
	}

	// 3. Define CLI flags using env variables as fallbacks
	volumePath := flag.String("volume", envVolume, "Source volume path to scan (e.g. /Volumes/MySSD)")
	outDir := flag.String("out", envOut, "Output directory path for screenshots")
	quality := flag.Float64("quality", envQuality, "WebP compression quality (0-100)")
	workers := flag.Int("workers", envWorkers, "Number of parallel workers")
	deleteOriginals := flag.Bool("delete-originals", deleteOriginalFlag, "Delete original files on volume for previously converted screenshots")

	flag.Parse()

	// 4. Validate output path (required for both modes)
	if *outDir == "" {
		return nil, fmt.Errorf("output directory path is required. Set OUT_DIR in .env or pass -out flag")
	}

	// 5. Validate volume path (required unless in delete-originals mode)
	if !*deleteOriginals && *volumePath == "" {
		return nil, fmt.Errorf("source volume path is required. Set VOLUME_PATH in .env or pass -volume flag")
	}

	stagedFolder := fmt.Sprintf("%s/screenshots", *outDir)
	manifestPath := fmt.Sprintf("%s/manifest.json", *outDir)
	logPath := fmt.Sprintf("%s/error.log", *outDir)

	return &Config{
		VolumePath:      *volumePath,
		OutDir:          *outDir,
		Quality:         *quality,
		Workers:         *workers,
		DeleteOriginals: *deleteOriginals,
		StagedFolder:    stagedFolder,
		ManifestPath:    manifestPath,
		LogPath:         logPath,
	}, nil
}