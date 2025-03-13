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

// Application constants
const (
	AppName     = "Ollama Sync"
	AppVersion  = "1.0.0"
	Description = "A tool to bridge Ollama models with other tools (like LM Studio) seamlessly."
	AsciiArt    = `
	'||  '||                                                               
  ...    ||   ||   ....   .. .. ..    ....      ....  .... ... .. ...     ....  
.|  '|.  ||   ||  '' .||   || || ||  '' .||    ||. '   '|.  |   ||  ||  .|   '' 
||   ||  ||   ||  .|' ||   || || ||  .|' ||    . '|..   '|.|    ||  ||  ||      
 '|..|' .||. .||. '|..'|' .|| || ||. '|..'|'   |'..|'    '|    .||. ||.  '|...' 
													  .. |                      
													   ''                       
`
)

func displayWelcomeMessage() {
	fmt.Println(AsciiArt)
	fmt.Printf("%s - Version %s\n", AppName, AppVersion)
	fmt.Println(Description)
	fmt.Println()
}

func applyCommandLineOverrides(cfg *config.Config) {
	if manifestDir != "" {
		cfg.ManifestDir = manifestDir
	}
	if blobDir != "" {
		cfg.BlobDir = blobDir
	}
	if destinations != "" {
		cfg.Destinations = strings.Split(destinations, ",")
	}
}

func processManifests(cfg *config.Config, log *logrus.Logger) error {
	manifestFiles, err := manifest.FindManifestFiles(cfg.ManifestDir)
	if err != nil {
		return fmt.Errorf("error while searching for manifest files: %w", err)
	}
	if len(manifestFiles) == 0 {
		log.Warn("No manifest files found")
		return nil
	}

	for _, filePath := range manifestFiles {
		log.WithField("manifest_path", filePath).Info("Found manifest file")
		if err := manifest.ProcessManifest(filePath, cfg.BlobDir, cfg.Destinations, log); err != nil {
			log.WithError(err).WithField("manifest_path", filePath).Error("Failed to process manifest")
			// Continue processing other manifests
		}
	}
	
	log.Info("ollama-sync complete.")
	return nil
}

func runSync(cfg *config.Config, log *logrus.Logger) error {
	log.Infof("Running on %s", runtime.GOOS)
	log.WithFields(logrus.Fields{
		"manifest_dir": cfg.ManifestDir,
		"blob_dir":     cfg.BlobDir,
		"destinations": cfg.Destinations,
	}).Info("Application directories determined from config")

	return processManifests(cfg, log)
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
				log.WithError(err).Fatal("Failed to load configuration")
			}

			applyCommandLineOverrides(cfg)

			if err := runSync(cfg, log); err != nil {
				log.WithError(err).Fatal("Sync operation failed")
			}
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
