package processor

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/schollz/progressbar/v3"

	"github.com/sbanik/shrink-my-photos/internal/config"
	"github.com/sbanik/shrink-my-photos/internal/detector"
	"github.com/sbanik/shrink-my-photos/internal/helper"
)

type StageResult struct {
	Pending          int
	HiddenFiles      []string
	HiddenFilesBytes int64
}

// RunStage recursively discovers convertible images without copying them out of
// the source tree. Duplicates are moved beside their source folder into discarded.
func RunStage(cfg *config.Config) StageResult {
	manifest := prepareManifest(cfg)
	candidates, cameraFolders, hiddenFiles := collectCandidateFiles(cfg)
	hiddenFilesBytes := recordHiddenFiles(manifest, hiddenFiles)
	if len(candidates) == 0 {
		fmt.Println("No eligible image files found.")
		if err := helper.SaveManifest(cfg.ManifestPath, manifest); err != nil {
			fmt.Printf("Could not save manifest: %v\n", err)
		}
		printHiddenFiles(hiddenFiles, hiddenFilesBytes)
		return StageResult{HiddenFiles: hiddenFiles, HiddenFilesBytes: hiddenFilesBytes}
	}

	bar := progressbar.NewOptions(
		len(candidates),
		progressbar.OptionSetDescription("Scanning"),
		progressbar.OptionShowCount(),
		progressbar.OptionSetPredictTime(true),
	)
	seenHashes := make(map[string]string)
	var pending, duplicates, ignoredCamera, skipped int

	for _, path := range candidates {
		func(path string) {
			defer func() { _ = bar.Add(1) }()
			if cameraFolders[filepath.Dir(path)] {
				ignoredCamera++
				return
			}
			info, err := os.Stat(path)
			if err != nil {
				return
			}
			hash, err := calculateSHA256(path)
			if err != nil {
				return
			}
			if _, exists := seenHashes[hash]; exists {
				discardedPath, err := moveToDiscarded(path)
				if err != nil {
					fmt.Printf("Could not move duplicate %s: %v\n", path, err)
					return
				}
				manifest.Records[path] = &helper.FileRecord{
					OriginalPath: path, DiscardedPath: discardedPath, OriginalSize: info.Size(), Status: "discarded",
				}
				duplicates++
				return
			}
			seenHashes[hash] = path

			if existing, exists := manifest.Records[path]; exists && existing.Status != "discarded" {
				skipped++
				return
			}
			manifest.Records[path] = &helper.FileRecord{OriginalPath: path, OriginalSize: info.Size(), Status: "pending"}
			pending++
		}(path)
	}

	if err := helper.SaveManifest(cfg.ManifestPath, manifest); err != nil {
		fmt.Printf("Could not save manifest: %v\n", err)
	}
	fmt.Printf("\nScan complete: %d pending, %d duplicates moved to discarded, %d camera-folder files ignored, %d already tracked.\n", pending, duplicates, ignoredCamera, skipped)
	printHiddenFiles(hiddenFiles, hiddenFilesBytes)
	return StageResult{Pending: pending, HiddenFiles: hiddenFiles, HiddenFilesBytes: hiddenFilesBytes}
}

func prepareManifest(cfg *config.Config) *helper.Manifest {
	if cfg.Clean {
		return &helper.Manifest{Records: make(map[string]*helper.FileRecord), HiddenFiles: make(map[string]*helper.HiddenRecord)}
	}
	manifest, err := helper.LoadManifest(cfg.ManifestPath)
	if err != nil {
		return &helper.Manifest{Records: make(map[string]*helper.FileRecord), HiddenFiles: make(map[string]*helper.HiddenRecord)}
	}
	if manifest.Records == nil {
		manifest.Records = make(map[string]*helper.FileRecord)
	}
	if manifest.HiddenFiles == nil {
		manifest.HiddenFiles = make(map[string]*helper.HiddenRecord)
	}
	return manifest
}

func collectCandidateFiles(cfg *config.Config) ([]string, map[string]bool, []string) {
	var files []string
	var hiddenFiles []string
	cameraFolders := make(map[string]bool)
	_ = filepath.Walk(cfg.VolumePath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		name := info.Name()
		if info.IsDir() {
			if path != cfg.VolumePath && (name == "processed" || name == "discarded" || isHidden(name)) {
				return filepath.SkipDir
			}
			return nil
		}
		if isHidden(name) {
			hiddenFiles = append(hiddenFiles, path)
			return nil
		}
		if !isAllowedExtension(path, cfg.AllowedTypes) {
			return nil
		}
		files = append(files, path)
		if detector.IsCameraPhoto(path) {
			cameraFolders[filepath.Dir(path)] = true
		}
		return nil
	})
	sort.Slice(files, func(i, j int) bool {
		return pathDepth(cfg.VolumePath, files[i]) < pathDepth(cfg.VolumePath, files[j]) ||
			(pathDepth(cfg.VolumePath, files[i]) == pathDepth(cfg.VolumePath, files[j]) && files[i] < files[j])
	})
	sort.Strings(hiddenFiles)
	return files, cameraFolders, hiddenFiles
}

func recordHiddenFiles(manifest *helper.Manifest, files []string) int64 {
	var total int64
	for _, path := range files {
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		manifest.HiddenFiles[path] = &helper.HiddenRecord{Path: path, Size: info.Size(), Status: "present"}
		total += info.Size()
	}
	return total
}

func printHiddenFiles(files []string, totalBytes int64) {
	if len(files) == 0 {
		return
	}
	fmt.Printf("\nFound %d hidden file(s), including possible Apple metadata (%s total):\n", len(files), helper.FormatBytes(totalBytes))
	for _, path := range files {
		fmt.Printf("  %s\n", path)
	}
}

// DeleteHiddenFiles deletes only regular hidden files explicitly selected by the user.
func DeleteHiddenFiles(manifestPath string, files []string) int {
	manifest, err := helper.LoadManifest(manifestPath)
	if err != nil {
		fmt.Printf("Could not update hidden-file manifest: %v\n", err)
		return 0
	}
	if manifest.HiddenFiles == nil {
		manifest.HiddenFiles = make(map[string]*helper.HiddenRecord)
	}
	deleted := 0
	for _, path := range files {
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		if err := os.Remove(path); err != nil {
			fmt.Printf("Could not delete hidden file %s: %v\n", path, err)
			continue
		}
		if record, exists := manifest.HiddenFiles[path]; exists {
			record.Status = "deleted"
		}
		deleted++
	}
	if err := helper.SaveManifest(manifestPath, manifest); err != nil {
		fmt.Printf("Could not save hidden-file manifest updates: %v\n", err)
	}
	fmt.Printf("Deleted %d hidden file(s).\n", deleted)
	return deleted
}

// PrintHiddenFileList reports every hidden file recorded for the current volume.
func PrintHiddenFileList(manifestPath string) {
	manifest, err := helper.LoadManifest(manifestPath)
	if err != nil {
		fmt.Printf("No hidden-file manifest available: %v\n", err)
		return
	}
	if len(manifest.HiddenFiles) == 0 {
		fmt.Println("No hidden files have been recorded.")
		return
	}
	paths := make([]string, 0, len(manifest.HiddenFiles))
	for path := range manifest.HiddenFiles {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	var presentBytes int64
	for _, path := range paths {
		record := manifest.HiddenFiles[path]
		fmt.Printf("[%s] %s (%s)\n", record.Status, record.Path, helper.FormatBytes(record.Size))
		if record.Status == "present" {
			presentBytes += record.Size
		}
	}
	fmt.Printf("Present hidden-file space: %s\n", helper.FormatBytes(presentBytes))
}

func pathDepth(root, path string) int {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return 0
	}
	return strings.Count(relative, string(os.PathSeparator))
}

func moveToDiscarded(sourcePath string) (string, error) {
	discardedDir := filepath.Join(filepath.Dir(sourcePath), "discarded")
	if err := os.MkdirAll(discardedDir, 0755); err != nil {
		return "", err
	}
	destination := filepath.Join(discardedDir, filepath.Base(sourcePath))
	for suffix := 1; ; suffix++ {
		if _, err := os.Lstat(destination); os.IsNotExist(err) {
			break
		}
		ext := filepath.Ext(filepath.Base(sourcePath))
		base := strings.TrimSuffix(filepath.Base(sourcePath), ext)
		destination = filepath.Join(discardedDir, fmt.Sprintf("%s-%d%s", base, suffix, ext))
	}
	if err := os.Rename(sourcePath, destination); err != nil {
		return "", err
	}
	return destination, nil
}

func isHidden(filename string) bool { return strings.HasPrefix(filename, ".") }

func isAllowedExtension(filePath string, allowedTypes []string) bool {
	ext := filepath.Ext(filePath)
	for _, allowed := range allowedTypes {
		if strings.EqualFold(ext, allowed) {
			return true
		}
	}
	return false
}

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
