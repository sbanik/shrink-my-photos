package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Mode         string
	VolumePath   string
	OutDir       string
	StagedFolder string
	ManifestPath string
	LogPath      string
	Quality      float64
	Workers      int
	Clean        bool
	AllowedTypes []string
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	var (
		modeFlag  string
		volFlag   string
		outFlag   string
		qualFlag  float64
		workFlag  int
		cleanFlag bool
		typesFlag string
	)

	flag.StringVar(&modeFlag, "mode", getEnvString("MODE", "stage"), "Execution mode: all, stage, sync, convert, delete")
	flag.StringVar(&volFlag, "volume", os.Getenv("VOLUME_PATH"), "Source volume directory path")
	flag.StringVar(&outFlag, "out", os.Getenv("OUT_DIR"), "Output directory path")
	flag.Float64Var(&qualFlag, "quality", getEnvFloat("QUALITY", 80.0), "WebP quality target (0.0 - 100.0)")
	flag.IntVar(&workFlag, "workers", getEnvInt("WORKERS", 4), "Number of concurrent workers")
	flag.BoolVar(&cleanFlag, "clean", getEnvBool("CLEAN_STAGED", false), "Clean staged folder and manifest before staging")
	flag.StringVar(&typesFlag, "types", getEnvString("ALLOWED_TYPES", "png,jpg,jpeg"), "Comma-separated extensions to scan (e.g. png,jpg,jpeg)")
	flag.Parse()

	mode := strings.ToLower(modeFlag)

	validModes := map[string]bool{
		"all":     true,
		"stage":   true,
		"sync":    true,
		"convert": true,
		"delete":  true,
	}
	if !validModes[mode] {
		return nil, fmt.Errorf("invalid mode '%s': must be one of all, stage, sync, convert, delete", modeFlag)
	}

	if outFlag == "" {
		return nil, fmt.Errorf("output directory path must be provided via -out flag or OUT_DIR env variable")
	}

	if (mode == "stage" || mode == "all") && volFlag == "" {
		return nil, fmt.Errorf("volume path must be provided via -volume flag or VOLUME_PATH env variable for mode '%s'", mode)
	}

	rawTypes := strings.Split(typesFlag, ",")
	var allowedTypes []string
	for _, t := range rawTypes {
		clean := strings.ToLower(strings.TrimSpace(t))
		if clean != "" {
			if !strings.HasPrefix(clean, ".") {
				clean = "." + clean
			}
			allowedTypes = append(allowedTypes, clean)
		}
	}

	stagedFolder := filepath.Join(outFlag, "to_process")
	manifestPath := filepath.Join(outFlag, "manifest.json")
	logPath := filepath.Join(outFlag, "shrinker.log")

	return &Config{
		Mode:         mode,
		VolumePath:   volFlag,
		OutDir:       outFlag,
		StagedFolder: stagedFolder,
		ManifestPath: manifestPath,
		LogPath:      logPath,
		Quality:      qualFlag,
		Workers:      workFlag,
		Clean:        cleanFlag,
		AllowedTypes: allowedTypes,
	}, nil
}

func getEnvString(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		if parsed, err := strconv.ParseFloat(val, 64); err == nil {
			return parsed
		}
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			return parsed
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		if parsed, err := strconv.ParseBool(val); err == nil {
			return parsed
		}
	}
	return fallback
}