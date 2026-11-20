package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/ishan-marikar/lm-studio-ollama-bridge/internal/config"
	"github.com/ishan-marikar/lm-studio-ollama-bridge/internal/logger"
	"github.com/ishan-marikar/lm-studio-ollama-bridge/internal/manifest"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	manifestDir  string
	blobDir      string
	destinations string
)

func displayWelcomeMessage() {
	asciiArt := `
	'||  '||                                                               
  ...    ||   ||   ....   .. .. ..    ....      ....  .... ... .. ...     ....  
.|  '|.  ||   ||  '' .||   || || ||  '' .||    ||. '   '|.  |   ||  ||  .|   '' 
||   ||  ||   ||  .|' ||   || || ||  .|' ||    . '|..   '|.|    ||  ||  ||      
 '|..|' .||. .||. '|..'|' .|| || ||. '|..'|'   |'..|'    '|    .||. ||.  '|...' 
													  .. |                      
													   ''                       
`
	appName := "Ollama Sync"
	version := "1.0.0"
	description := "A tool to bridge Ollama models with other tools (like LM Studio) seamlessly."

	fmt.Println(asciiArt)
	fmt.Printf("%s - Version %s\n", appName, version)
	fmt.Println(description)
	fmt.Println()
}

func main() {
	// Define the root command using Cobra.
	var rootCmd = &cobra.Command{
		Use:   "ollama-sync",
		Short: "A tool to bridge Ollama models with other tools seamlessly",
		Run: func(cmd *cobra.Command, args []string) {
			displayWelcomeMessage()

			log := logger.InitLogger()

			cfg, err := config.LoadConfig()
			if err != nil {
				log.WithError(err).Error("Failed to load configuration")
				os.Exit(1)
			}

			if manifestDir != "" {
				cfg.ManifestDir = manifestDir
			}
			if blobDir != "" {
				cfg.BlobDir = blobDir
			}
			if destinations != "" {
    			cfg.Destinations = strings.Split(destinations, ",")
			}

			log.Infof("Running on %s", runtime.GOOS)
			log.WithFields(logrus.Fields{
				"manifest_dir": cfg.ManifestDir,
				"blob_dir":     cfg.BlobDir,
				"destinations": cfg.Destinations,
			}).Info("Application directories determined from config")

			manifestFiles, err := manifest.FindModelManifests(cfg.ManifestDir)
			if err != nil {
				log.WithError(err).Error("Error while searching for manifest files")
				os.Exit(1)
			}
			if len(manifestFiles) == 0 {
				log.Warn("No manifest files found")
				os.Exit(0)
			}

			keep := map[string]bool{}
			allProcessed := true
			for _, filePath := range manifestFiles {
				log.WithField("manifest_path", filePath).Debug("Found manifest file")
				links, err := manifest.ProcessManifest(filePath, cfg.BlobDir, cfg.Destinations, log)
				if err != nil {
					log.WithError(err).WithField("manifest_path", filePath).Error("Failed to process manifest")
					allProcessed = false
					continue
				}
				for _, link := range links {
					keep[link] = true
				}
			}

			// Prune only after a clean run: a manifest that failed to process would
			// otherwise lose its (still valid) link.
			if allProcessed {
				removed, err := manifest.PruneStaleLinks(cfg.Destinations, cfg.BlobDir, keep, log)
				if err != nil {
					log.WithError(err).Error("Failed to prune stale links")
				} else if removed > 0 {
					log.Infof("Removed %d stale link(s) with no current manifest", removed)
				}
			} else {
				log.Warn("Some manifests failed to process; not pruning stale links this run")
			}

			log.Info("ollama-sync complete.")
		},
	}

	rootCmd.PersistentFlags().StringVar(&manifestDir, "manifest_dir", "", "Directory containing manifest files")
	rootCmd.PersistentFlags().StringVar(&blobDir, "blob_dir", "", "Directory containing blob files")
	rootCmd.PersistentFlags().StringVar(&destinations, "destinations", "", "Comma-separated list of destinations")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
