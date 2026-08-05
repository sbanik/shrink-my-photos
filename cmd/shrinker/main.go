package main

import (
	"bufio"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"

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

	case "convert":
		processor.RunConvert(cfg)

	case "all":
		stagedCount := processor.RunStage(cfg)
		if stagedCount == 0 {
			return
		}

		fmt.Println("\n=======================================================")
		helper.FmtPrintfStagedInfo(cfg.StagedFolder)
		fmt.Print("--> Press ENTER when ready to convert images to WebP... ")
		_, _ = bufio.NewReader(os.Stdin).ReadString('\n')

		processor.RunConvert(cfg)

		fmt.Println("\n=======================================================")
		fmt.Print("--> Press ENTER when ready to delete originals... ")
		_, _ = bufio.NewReader(os.Stdin).ReadString('\n')

		processor.RunDeleteOriginals(cfg.ManifestPath)

	}
}
