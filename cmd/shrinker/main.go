package main

import (
	"bufio"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
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

	// Setup logger to capture errors on disk
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
		processor.RunStage(cfg)

	case "sync":
		processor.RunSync(cfg)

	case "convert":
		processor.RunConvert(cfg)

	case "all":
		stagedCount := processor.RunStage(cfg)
		if stagedCount == 0 {
			return
		}

		reader := bufio.NewReader(os.Stdin)

		fmt.Println("\n=======================================================")
		helper.FmtPrintfStagedInfo(cfg.StagedFolder)
		fmt.Print("--> Press ENTER when ready to convert images to WebP... ")
		_, _ = reader.ReadString('\n')

		processor.RunConvert(cfg)

		fmt.Println("\n=======================================================")
		fmt.Print("--> Do you want to delete original files from the target volume? (yes/no): ")

		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input. Skipping deletion of original files.")
			return
		}

		// Clean up input for comparison
		answer := strings.ToLower(strings.TrimSpace(input))

		if answer == "yes" || answer == "y" {
			processor.RunDeleteOriginals(cfg.ManifestPath)
		} else {
			fmt.Println("\nSkipping deletion of original files. Your original images remain intact.")
			helper.FmtPrintfDeleteOriginalsCmd(cfg.OutDir)
		}

	}
}
