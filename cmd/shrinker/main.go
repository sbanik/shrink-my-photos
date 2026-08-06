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

	case "stage":
		result := processor.RunStage(cfg)
		promptHiddenFileDeletion(bufio.NewReader(os.Stdin), cfg.ManifestPath, result.HiddenFiles)

	case "sync":
		processor.RunSync(cfg)

	case "convert":
		processor.RunConvert(cfg)

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
		processor.RunConvert(cfg)
		if promptYesNo(reader, "Delete converted original files now? (yes/no): ") {
			processor.RunDeleteOriginals(cfg.ManifestPath)
		} else {
			fmt.Println("Original files were kept. Run with -mode=delete after reviewing output if needed.")
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
		processor.RunConvert(cfg)
		fmt.Println("Deleting converted original files...")
		processor.RunDeleteOriginals(cfg.ManifestPath)
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
