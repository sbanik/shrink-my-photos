package processor

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/schollz/progressbar/v3"

	"github.com/sbanik/shrink-my-photos/internal/config"
	"github.com/sbanik/shrink-my-photos/internal/helper"
)

// RunStage coordinates the staging process.
func RunStage(cfg *config.Config) int {
	manifest := prepareManifest(cfg)

	if err := createRequiredDirs(cfg.StagedFolder, cfg.DuplicatesFolder); err != nil {
		fmt.Println(err)
		return 0
	}

	totalFiles := countCandidateFiles(cfg.VolumePath)
	if totalFiles == 0 {
		fmt.Println("No candidate files found in volume path.")
		return 0
	}

	fmt.Println()
	barStage := progressbar.Default(totalFiles, "Staging photos")
	seenHashes := make(map[string]string)

	var stagedCount, duplicateCount, skippedCount, hiddenSkippedCount int

	_ = filepath.Walk(cfg.VolumePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		filename := filepath.Base(path)

		// Check for hidden files/folders
		if isHidden(filename) {
			if info.IsDir() && path != cfg.VolumePath {
				return filepath.SkipDir
			}
			hiddenSkippedCount++
			return nil
		}

		if info.IsDir() {
			return nil
		}

		defer func() { _ = barStage.Add(1) }()

		if !isAllowedExtension(path, cfg.AllowedTypes) {
			return nil
		}

		action := processFile(path, filename, info, cfg, manifest, seenHashes)
		switch action {
		case actionStaged:
			stagedCount++
		case actionDuplicate:
			duplicateCount++
		case actionSkipped:
			skippedCount++
		}

		return nil
	})

	_ = helper.SaveManifest(cfg.ManifestPath, manifest)
	printStagingSummary(stagedCount, duplicateCount, hiddenSkippedCount, skippedCount)

	return stagedCount
}

// --- Helper Types & Constants ---

type processAction int

const (
	actionIgnored processAction = iota
	actionStaged
	actionDuplicate
	actionSkipped
)

// --- Helper Methods ---

// prepareManifest loads an existing manifest or initializes a clean one if configured.
func prepareManifest(cfg *config.Config) *helper.Manifest {
	if cfg.Clean {
		_ = os.RemoveAll(cfg.StagedFolder)
		return &helper.Manifest{Records: make(map[string]*helper.FileRecord)}
	}

	manifest, err := helper.LoadManifest(cfg.ManifestPath)
	if err != nil {
		return &helper.Manifest{Records: make(map[string]*helper.FileRecord)}
	}
	return manifest
}

// createRequiredDirs ensures staged and duplicates output directories exist.
func createRequiredDirs(stagedDir, dupDir string) error {
	if err := os.MkdirAll(stagedDir, 0755); err != nil {
		return fmt.Errorf("Error creating staged folder: %w", err)
	}
	if err := os.MkdirAll(dupDir, 0755); err != nil {
		return fmt.Errorf("Error creating duplicates folder: %w", err)
	}
	return nil
}

// countCandidateFiles performs a quick pre-scan to count non-hidden candidate files.
func countCandidateFiles(volPath string) int64 {
	var count int64
	_ = filepath.Walk(volPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		filename := filepath.Base(path)
		if isHidden(filename) {
			if info.IsDir() && path != volPath {
				return filepath.SkipDir
			}
			return nil
		}
		if !info.IsDir() {
			count++
		}
		return nil
	})
	return count
}

// isHidden checks if a file or directory starts with a dot.
func isHidden(filename string) bool {
	return strings.HasPrefix(filename, ".")
}

// isAllowedExtension checks if the file matches one of the configured allowed extensions.
func isAllowedExtension(filePath string, allowedTypes []string) bool {
	ext := filepath.Ext(filePath)
	for _, t := range allowedTypes {
		if strings.EqualFold(ext, t) {
			return true
		}
	}
	return false
}

// processFile hashes the file content, checks for duplicates, and handles staging logic.
func processFile(
	filePath, filename string,
	info os.FileInfo,
	cfg *config.Config,
	manifest *helper.Manifest,
	seenHashes map[string]string,
) processAction {
	fileHash, err := calculateSHA256(filePath)
	if err != nil {
		return actionIgnored
	}

	// Route duplicate content
	if _, exists := seenHashes[fileHash]; exists {
		dupDestPath := filepath.Join(cfg.DuplicatesFolder, filename)
		if err := helper.CopyFile(filePath, dupDestPath); err == nil {
			return actionDuplicate
		}
		return actionIgnored
	}

	seenHashes[fileHash] = filePath
	stagedDestPath := filepath.Join(cfg.StagedFolder, filename)

	// Check manifest for existing record
	if _, exists := manifest.Records[stagedDestPath]; exists {
		return actionSkipped
	}

	// Copy to staged directory
	if err := helper.CopyFile(filePath, stagedDestPath); err != nil {
		return actionIgnored
	}

	manifest.Records[stagedDestPath] = &helper.FileRecord{
		OriginalPath: filePath,
		StagedPath:   stagedDestPath,
		OriginalSize: info.Size(),
		Status:       "staged",
	}

	return actionStaged
}

// calculateSHA256 computes the SHA-256 hash of a file's content.
func calculateSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// printStagingSummary displays execution metrics.
func printStagingSummary(staged, duplicates, hidden, skipped int) {
	fmt.Printf("\n========================================")
	fmt.Printf("\nStaging Summary:")
	fmt.Printf("\nSuccessfully Staged : %d", staged)
	fmt.Printf("\nDuplicates Detected : %d", duplicates)
	fmt.Printf("\nHidden Files Skipped: %d", hidden)
	fmt.Printf("\nSkipped (Existing)  : %d", skipped)
	fmt.Printf("\n========================================\n")
}