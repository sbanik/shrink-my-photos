package processor

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sbanik/shrink-my-photos/internal/config"
	"github.com/sbanik/shrink-my-photos/internal/helper"
)

// calculateSHA256 computes the hexadecimal SHA-256 hash of a file's content
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

// RunStage scans the volume path for allowed images, calculates hashes, and stages them.
// Hidden files and duplicates are automatically filtered or routed to duplicates.
func RunStage(cfg *config.Config) int {
	manifest, err := helper.LoadManifest(cfg.ManifestPath)
	if err != nil {
		manifest = &helper.Manifest{
			Records: make(map[string]*helper.FileRecord),
		}
	}

	if cfg.Clean {
		_ = os.RemoveAll(cfg.StagedFolder)
		manifest = &helper.Manifest{
			Records: make(map[string]*helper.FileRecord),
		}
	}

	// Ensure directories exist
	if err := os.MkdirAll(cfg.StagedFolder, 0755); err != nil {
		fmt.Printf("Error creating staged folder: %v\n", err)
		return 0
	}
	if err := os.MkdirAll(cfg.DuplicatesFolder, 0755); err != nil {
		fmt.Printf("Error creating duplicates folder: %v\n", err)
		return 0
	}

	// Track existing hashes to detect duplicates across existing manifest records and current run
	seenHashes := make(map[string]string)

	var stagedCount, duplicateCount, skippedCount, hiddenSkippedCount int

	err = filepath.Walk(cfg.VolumePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		filename := filepath.Base(path)

		// 1. Skip hidden files and macOS metadata files (.DS_Store, ._filename, .Trashes, etc.)
		if strings.HasPrefix(filename, ".") {
			if info.IsDir() && path != cfg.VolumePath {
				return filepath.SkipDir // Don't traverse into hidden directories (e.g. .git, .Trash)
			}
			hiddenSkippedCount++
			return nil
		}

		if info.IsDir() {
			return nil
		}

		// 2. Validate allowed extension
		ext := filepath.Ext(path)
		allowed := false
		for _, t := range cfg.AllowedTypes {
			if strings.EqualFold(ext, t) {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil
		}

		// 3. Calculate file content hash
		fileHash, err := calculateSHA256(path)
		if err != nil {
			fmt.Printf("Error hashing file %s: %v\n", path, err)
			return nil
		}

		// 4. Check for duplicates
		if originalPath, exists := seenHashes[fileHash]; exists {
			fmt.Printf("Duplicate content detected: %s (matches %s)\n", path, originalPath)

			dupDestPath := filepath.Join(cfg.DuplicatesFolder, filename)
			if err := helper.CopyFile(path, dupDestPath); err == nil {
				duplicateCount++
			} else {
				fmt.Printf("Failed to copy duplicate %s: %v\n", path, err)
			}
			return nil
		}

		// Mark hash as seen
		seenHashes[fileHash] = path

		stagedDestPath := filepath.Join(cfg.StagedFolder, filename)

		// Check if already staged in manifest
		if _, exists := manifest.Records[stagedDestPath]; exists {
			skippedCount++
			return nil
		}

		// Copy to staged folder
		if err := helper.CopyFile(path, stagedDestPath); err != nil {
			fmt.Printf("Failed to stage file %s: %v\n", path, err)
			return nil
		}

		manifest.Records[stagedDestPath] = &helper.FileRecord{
			OriginalPath: path,
			StagedPath:   stagedDestPath,
			OriginalSize: info.Size(),
			Status:       "staged",
		}
		stagedCount++

		return nil
	})

	if err != nil {
		fmt.Printf("Error walking volume path: %v\n", err)
	}

	_ = helper.SaveManifest(cfg.ManifestPath, manifest)

	fmt.Printf("\n========================================")
	fmt.Printf("\nStaging Summary:")
	fmt.Printf("\nSuccessfully Staged : %d", stagedCount)
	fmt.Printf("\nDuplicates Detected : %d", duplicateCount)
	fmt.Printf("\nHidden Files Skipped: %d", hiddenSkippedCount)
	fmt.Printf("\nSkipped (Existing)  : %d", skippedCount)
	fmt.Printf("\n========================================\n")

	return stagedCount
}