package processor

import (
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/schollz/progressbar/v3"

	"github.com/sbanik/shrink-my-photos/internal/config"
	"github.com/sbanik/shrink-my-photos/internal/detector"
	"github.com/sbanik/shrink-my-photos/internal/helper"
)


// runStage scans the target volume, identifies screenshots, copies them to the staging directory, and builds/updates manifest.json
func RunStage(cfg *config.Config) int {
	_ = os.MkdirAll(cfg.StagedFolder, 0755)

	fmt.Printf("Scanning volume path: %s\n", cfg.VolumePath)
	var detected []string

	_ = filepath.WalkDir(cfg.VolumePath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if !d.IsDir() && (ext == ".png" || ext == ".jpg" || ext == ".jpeg") {
			if detector.IsScreenshot(path) {
				detected = append(detected, path)
			}
		}
		return nil
	})

	if len(detected) == 0 {
		fmt.Println("No screenshots detected on target volume.")
		return 0
	}

	manifest, err := helper.LoadManifest(cfg.ManifestPath)
	if err != nil {
		manifest = &helper.Manifest{Records: make(map[string]*helper.FileRecord)}
	}

	fmt.Println()
	barStage := progressbar.Default(int64(len(detected)), "Staging screenshots")

	for i, srcPath := range detected {
		fileName := fmt.Sprintf("screenshot_%d_%s", i+1, filepath.Base(srcPath))
		destPath := filepath.Join(cfg.StagedFolder, fileName)

		fi, err := os.Stat(srcPath)
		var origSize int64
		if err == nil {
			origSize = fi.Size()
		}

		if err := helper.CopyFile(srcPath, destPath); err == nil {
			manifest.Records[destPath] = &helper.FileRecord{
				OriginalPath: srcPath,
				StagedPath:   destPath,
				OriginalSize: origSize,
				Status:       "staged",
			}
		} else {
			log.Printf("Failed to stage %s -> %s: %v", srcPath, destPath, err)
		}
		_ = barStage.Add(1)
	}

	helper.SaveManifest(cfg.ManifestPath, manifest)

	if cfg.Mode == "stage" {
		fmt.Println("\n=======================================================")
		helper.FmtPrintfStagedInfo(cfg.StagedFolder)
		fmt.Printf("Run conversion when ready using:\n./shrinker -mode=convert -out %s\n", cfg.OutDir)
		fmt.Println("=======================================================")
	}

	return len(manifest.Records)
}