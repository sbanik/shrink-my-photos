package main

import (
	"bufio"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/sbanik/shrink-my-photos/internal/config"
	"github.com/sbanik/shrink-my-photos/internal/helper"
	"github.com/sbanik/shrink-my-photos/internal/processor"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Configuration Error: %v\n", err)
		os.Exit(1)
	}
	if cfg.HiddenFileList {
		processor.PrintHiddenFileList(cfg.ManifestPath)
		return
	}

	// Setup logger to capture errors on disk
	_ = os.MkdirAll(filepath.Dir(cfg.LogPath), 0755)
	logFile, err := os.OpenFile(cfg.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		defer logFile.Close()
		log.SetOutput(logFile)
		log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	}

	// Mode Execution Dispatcher
	switch cfg.Mode {
	case "delete":
		processor.RunDeleteOriginals(cfg.ManifestPath)
		if promptYesNo(bufio.NewReader(os.Stdin), "Move processed WebP files into the original folders? (yes/no): ") {
			processor.RunFinalize(cfg)
		}

	case "stage":
		result := processor.RunStage(cfg)
		promptHiddenFileDeletion(bufio.NewReader(os.Stdin), cfg.ManifestPath, result.HiddenFiles)

	case "sync":
		processor.RunSync(cfg)

	case "convert":
		if ensureConversionWorkspace(cfg, bufio.NewReader(os.Stdin)) {
			processor.RunConvert(cfg)
		}

	case "all":
		result := processor.RunStage(cfg)
		reader := bufio.NewReader(os.Stdin)
		if !waitForReview(reader) {
			return
		}
		promptHiddenFileDeletion(reader, cfg.ManifestPath, result.HiddenFiles)
		fmt.Println("Synchronizing discarded files...")
		processor.RunSync(cfg)
		fmt.Println("Converting images...")
		if !ensureConversionWorkspace(cfg, reader) {
			return
		}
		processor.RunConvert(cfg)
		if promptYesNo(reader, "Delete converted original files now? (yes/no): ") {
			processor.RunDeleteOriginals(cfg.ManifestPath)
			if !cfg.ProcessedPathProvided {
				processor.RunFinalize(cfg)
			}
		} else {
			fmt.Println("Original files were kept. Run -mode=delete after reviewing output if needed.")
		}

	case "auto":
		fmt.Println("Discovering eligible images...")
		result := processor.RunStage(cfg)
		if cfg.DeleteHiddenFiles {
			processor.DeleteHiddenFiles(cfg.ManifestPath, result.HiddenFiles)
		}
		fmt.Println("Synchronizing discarded files...")
		processor.RunSync(cfg)
		fmt.Println("Converting images...")
		if !ensureConversionWorkspace(cfg, bufio.NewReader(os.Stdin)) {
			return
		}
		processor.RunConvert(cfg)
		if cfg.DeleteOriginals {
			fmt.Println("Deleting converted original files...")
			processor.RunDeleteOriginals(cfg.ManifestPath)
			if !cfg.ProcessedPathProvided {
				processor.RunFinalize(cfg)
			}
		} else {
			fmt.Println("Original files were kept because DELETE_ORIGINALS is false.")
		}
	}
}

func waitForReview(reader *bufio.Reader) bool {
	fmt.Println("Review files now. Move unwanted images into their sibling discarded folders.")
	fmt.Print("Press ENTER to continue, or type anything to stop: ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	if strings.TrimSpace(input) != "" {
		fmt.Println("Stopped before synchronization; no discarded or original files were deleted.")
		return false
	}
	return true
}

func promptHiddenFileDeletion(reader *bufio.Reader, manifestPath string, files []string) {
	if len(files) == 0 {
		return
	}
	if promptYesNo(reader, "Delete the listed hidden files? (yes/no): ") {
		processor.DeleteHiddenFiles(manifestPath, files)
	}
}

func promptYesNo(reader *bufio.Reader, message string) bool {
	fmt.Print(message)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(input))
	return answer == "yes" || answer == "y"
}

func ensureConversionWorkspace(cfg *config.Config, reader *bufio.Reader) bool {
	required, count, err := processor.RequiredConversionSpace(cfg)
	if err != nil {
		fmt.Printf("Unable to estimate conversion workspace: %v\n", err)
		return false
	}
	if count == 0 {
		return true
	}
	available, err := processor.AvailableWorkspaceSpace(cfg.ProcessedFolder)
	if err == nil && available >= required {
		return true
	}
	if err != nil {
		fmt.Printf("Unable to check free space at %s: %v\n", cfg.ProcessedFolder, err)
	} else {
		fmt.Printf("Insufficient free space at %s: %s available; %s required for %d image(s).\n", cfg.ProcessedFolder, helper.FormatBytes(available), helper.FormatBytes(required), count)
	}
	for {
		fmt.Printf("Provide a path to an empty directory with at least %s free, or press ENTER to cancel: ", helper.FormatBytes(required))
		input, readErr := reader.ReadString('\n')
		if readErr != nil || strings.TrimSpace(input) == "" {
			fmt.Println("Conversion cancelled before any original files were deleted.")
			return false
		}
		fallbackPath, validationErr := processor.ValidateFallbackWorkspace(strings.TrimSpace(input), required)
		if validationErr != nil {
			fmt.Printf("Invalid fallback directory: %v\n", validationErr)
			continue
		}
		cfg.ProcessedFolder = fallbackPath
		fmt.Printf("Using fallback processed directory: %s\n", fallbackPath)
		return true
	}
}
