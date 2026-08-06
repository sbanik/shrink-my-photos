package processor

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sbanik/shrink-my-photos/internal/helper"
)

// RunDuplicateReview finds byte-identical images across the selected folders.
// The first path in the supplied folder order is kept; deletion is always an
// explicit, per-group decision and never uses the conversion manifest.
func RunDuplicateReview(folders, allowedTypes []string, reader *bufio.Reader) int {
	files := collectDuplicateCandidates(folders, allowedTypes)
	groups := duplicateGroups(files)
	if len(groups) == 0 {
		fmt.Println("No duplicate images found in the selected folders.")
		return 0
	}

	var reclaimable int64
	for _, group := range groups {
		for _, path := range group[1:] {
			if info, err := os.Stat(path); err == nil {
				reclaimable += info.Size()
			}
		}
	}
	fmt.Printf("Found %d duplicate group(s), with %d removable file(s) reclaiming up to %s.\n", len(groups), removableCount(groups), helper.FormatBytes(reclaimable))

	deleted := 0
	for index, group := range groups {
		fmt.Printf("\nDuplicate group %d:\n  Keep: %s\n", index+1, group[0])
		for _, path := range group[1:] {
			fmt.Printf("  Delete candidate: %s\n", path)
		}
		if promptDeleteDuplicateGroup(reader) {
			for _, path := range group[1:] {
				if err := os.Remove(path); err != nil {
					fmt.Printf("Could not delete %s: %v\n", path, err)
					continue
				}
				deleted++
			}
		}
	}
	fmt.Printf("Duplicate review complete: deleted %d file(s).\n", deleted)
	return deleted
}

func collectDuplicateCandidates(folders, allowedTypes []string) []string {
	var files []string
	seen := make(map[string]bool)
	for _, folder := range folders {
		_ = filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil {
				return nil
			}
			if info.IsDir() {
				if path != folder && (info.Name() == "processed" || info.Name() == "discarded" || isHidden(info.Name())) {
					return filepath.SkipDir
				}
				return nil
			}
			if !isHidden(info.Name()) && isAllowedExtension(path, allowedTypes) && !seen[path] {
				seen[path] = true
				files = append(files, path)
			}
			return nil
		})
	}
	return files
}

func duplicateGroups(files []string) [][]string {
	byHash := make(map[string][]string)
	var hashes []string
	for _, path := range files {
		hash, err := calculateSHA256(path)
		if err != nil {
			continue
		}
		if _, found := byHash[hash]; !found {
			hashes = append(hashes, hash)
		}
		byHash[hash] = append(byHash[hash], path)
	}
	sort.Strings(hashes)
	groups := make([][]string, 0)
	for _, hash := range hashes {
		if len(byHash[hash]) > 1 {
			groups = append(groups, byHash[hash])
		}
	}
	return groups
}

func removableCount(groups [][]string) int {
	count := 0
	for _, group := range groups {
		count += len(group) - 1
	}
	return count
}

func promptDeleteDuplicateGroup(reader *bufio.Reader) bool {
	fmt.Print("Delete these candidates permanently? (yes/no): ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(input))
	if answer != "yes" && answer != "y" && answer != "no" && answer != "n" && answer != "" {
		fmt.Println("Skipped; enter yes to delete or no to keep candidates.")
		return false
	}
	return answer == "yes" || answer == "y"
}
