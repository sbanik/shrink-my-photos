package config

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

const (
	defaultQuality       = 80.0
	minQuality           = 55.0
	maxQuality           = 90.0
	defaultTargetSize    = 200 * 1024
	defaultMaxOutputSize = 400 * 1024
	defaultSmallFileSize = 150 * 1024
)

type Config struct {
	Mode                  string
	VolumePath            string
	ProcessedFolder       string
	ProcessedPathProvided bool
	ManifestPath          string
	LogPath               string
	Quality               float64 // Preferred WebP quality.
	QualitySpecified      bool
	TargetSize            int64
	MaxOutputSize         int64
	SmallFileSize         int64
	Workers               int
	Clean                 bool
	DeleteHiddenFiles     bool
	DeleteOriginals       bool
	HiddenFileList        bool
	AllowedTypes          []string
	DuplicateFolders      []string
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	fs := flag.NewFlagSet("config", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var modeFlag, volFlag, processedFlag, stateFlag, typesFlag, duplicateFoldersFlag string
	var qualityFlag float64
	var targetFlag, smallFlag int64
	var workersFlag int
	var cleanFlag, deleteHiddenFlag, deleteOriginalsFlag, hiddenFileListFlag bool

	fs.StringVar(&modeFlag, "mode", getEnvString("MODE", "all"), "Execution mode: auto, all, stage, sync, convert, delete, duplicates")
	fs.StringVar(&volFlag, "volume", os.Getenv("VOLUME_PATH"), "Source directory path")
	fs.StringVar(&processedFlag, "processed", os.Getenv("PROCESSED_PATH"), "Temporary directory for converted WebP files")
	fs.StringVar(&stateFlag, "state", os.Getenv("STATE_DIR"), "State directory for manifests and logs")
	fs.Float64Var(&qualityFlag, "quality", getEnvFloat("QUALITY", defaultQuality), "Preferred WebP quality (55-90)")
	fs.Int64Var(&targetFlag, "target-size", getEnvInt64("TARGET_SIZE_KB", defaultTargetSize/1024)*1024, "Target WebP size in KiB")
	fs.Int64Var(&smallFlag, "small-file-size", getEnvInt64("SMALL_FILE_SIZE_KB", defaultSmallFileSize/1024)*1024, "Files at or below this size are not resized")
	fs.IntVar(&workersFlag, "workers", getEnvInt("WORKERS", runtime.NumCPU()), "Number of concurrent workers")
	fs.BoolVar(&cleanFlag, "clean", getEnvBool("CLEAN_MANIFEST", false), "Start a new manifest before scanning")
	fs.BoolVar(&deleteHiddenFlag, "delete-hidden-files", getEnvBool("DELETE_HIDDEN_FILES", false), "Delete discovered hidden files without a prompt")
	fs.BoolVar(&deleteOriginalsFlag, "delete-originals", getEnvBool("DELETE_ORIGINALS", false), "Delete converted original files")
	fs.BoolVar(&hiddenFileListFlag, "hidden-file-list", false, "Print hidden files recorded in the manifest")
	fs.StringVar(&typesFlag, "types", getEnvString("ALLOWED_TYPES", "png,jpg,jpeg"), "Comma-separated extensions to scan")
	fs.StringVar(&duplicateFoldersFlag, "folders", os.Getenv("MULTI_FOLDER"), "Comma-separated folders to compare in duplicates mode")

	// Tests and callers may have their own flags. Ignore those after the first unknown flag.
	_ = fs.Parse(os.Args[1:])
	qualitySpecified := os.Getenv("QUALITY") != ""
	fs.Visit(func(flag *flag.Flag) {
		if flag.Name == "quality" {
			qualitySpecified = true
		}
	})

	mode := strings.ToLower(modeFlag)
	validModes := map[string]bool{"auto": true, "all": true, "stage": true, "sync": true, "convert": true, "delete": true, "duplicates": true}
	if !validModes[mode] {
		return nil, fmt.Errorf("invalid mode %q: must be one of auto, all, stage, sync, convert, delete, duplicates", modeFlag)
	}
	var duplicateFolders []string
	if mode == "duplicates" {
		resolvedFolders, folderErr := resolveFolders(duplicateFoldersFlag)
		if folderErr != nil {
			return nil, folderErr
		}
		duplicateFolders = resolvedFolders
		if len(duplicateFolders) == 0 {
			return nil, fmt.Errorf("at least one folder must be provided via -folders or MULTI_FOLDER in duplicates mode")
		}
	}
	if mode != "duplicates" && volFlag == "" {
		return nil, fmt.Errorf("volume path must be provided via -volume flag or VOLUME_PATH env variable")
	}
	if volFlag == "" {
		volFlag = duplicateFolders[0]
	}
	volumePath, err := filepath.Abs(volFlag)
	if err != nil {
		return nil, fmt.Errorf("resolve volume path: %w", err)
	}
	if qualityFlag < minQuality || qualityFlag > maxQuality {
		return nil, fmt.Errorf("quality must be between %.0f and %.0f", minQuality, maxQuality)
	}
	if workersFlag < 1 {
		return nil, fmt.Errorf("workers must be at least 1")
	}
	if targetFlag <= 0 || smallFlag <= 0 {
		return nil, fmt.Errorf("target-size and small-file-size must be positive")
	}

	allowedTypes := normalizeTypes(typesFlag)
	if len(allowedTypes) == 0 {
		return nil, fmt.Errorf("at least one allowed file type is required")
	}

	stateDir, err := resolveStateDir(stateFlag)
	if err != nil {
		return nil, err
	}
	key := volumeKey(volumePath)
	processedPath := filepath.Join(volumePath, "processed")
	if processedFlag != "" {
		processedPath, err = filepath.Abs(processedFlag)
		if err != nil {
			return nil, fmt.Errorf("resolve processed path: %w", err)
		}
	}

	return &Config{
		Mode:                  mode,
		VolumePath:            volumePath,
		ProcessedFolder:       processedPath,
		ProcessedPathProvided: processedFlag != "",
		ManifestPath:          filepath.Join(stateDir, "manifests", key+".json"),
		LogPath:               filepath.Join(stateDir, "logs", key+".log"),
		Quality:               qualityFlag,
		QualitySpecified:      qualitySpecified,
		TargetSize:            targetFlag,
		MaxOutputSize:         defaultMaxOutputSize,
		SmallFileSize:         smallFlag,
		Workers:               workersFlag,
		Clean:                 cleanFlag,
		DeleteHiddenFiles:     deleteHiddenFlag,
		DeleteOriginals:       deleteOriginalsFlag,
		HiddenFileList:        hiddenFileListFlag,
		AllowedTypes:          allowedTypes,
		DuplicateFolders:      duplicateFolders,
	}, nil
}

func resolveFolders(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	seen := make(map[string]bool)
	var folders []string
	for _, value := range strings.Split(raw, ",") {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		path, err := filepath.Abs(value)
		if err != nil {
			return nil, fmt.Errorf("resolve duplicate folder %q: %w", value, err)
		}
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("access duplicate folder %s: %w", path, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("duplicate folder is not a directory: %s", path)
		}
		if !seen[path] {
			seen[path] = true
			folders = append(folders, path)
		}
	}
	return folders, nil
}

func resolveStateDir(override string) (string, error) {
	if override != "" {
		return filepath.Abs(override)
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate user configuration directory: %w", err)
	}
	return filepath.Join(dir, "shrink-my-photos"), nil
}

func volumeKey(volumePath string) string {
	sum := sha256.Sum256([]byte(volumePath))
	return hex.EncodeToString(sum[:8])
}

func normalizeTypes(raw string) []string {
	var types []string
	for _, value := range strings.Split(raw, ",") {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if !strings.HasPrefix(value, ".") {
			value = "." + value
		}
		types = append(types, value)
	}
	return types
}

func getEnvString(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
func getEnvFloat(key string, fallback float64) float64 {
	if value, err := strconv.ParseFloat(os.Getenv(key), 64); err == nil {
		return value
	}
	return fallback
}
func getEnvInt(key string, fallback int) int {
	if value, err := strconv.Atoi(os.Getenv(key)); err == nil {
		return value
	}
	return fallback
}
func getEnvInt64(key string, fallback int64) int64 {
	if value, err := strconv.ParseInt(os.Getenv(key), 10, 64); err == nil {
		return value
	}
	return fallback
}
func getEnvBool(key string, fallback bool) bool {
	if value, err := strconv.ParseBool(os.Getenv(key)); err == nil {
		return value
	}
	return fallback
}
