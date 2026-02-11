package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Config struct {
	ManifestDir  string   `mapstructure:"manifest_dir"`
	BlobDir      string   `mapstructure:"blob_dir"`
	Destinations []string `mapstructure:"destinations"`
}

func LoadConfig() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("unable to determine user home directory: %w", err)
	}

	var configDir, defaultManifest, defaultBlob, defaultDest string

	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		localAppData := os.Getenv("LOCALAPPDATA")
		// Fallback to home directory if these variables are not set.
		if appData == "" {
			appData = homeDir
		}
		if localAppData == "" {
			localAppData = homeDir
		}
		// Use Windows conventions.
		configDir = filepath.Join(appData, "ollama-sync")
		defaultManifest = filepath.Join(appData, "Ollama", "models", "manifests")
		defaultBlob = filepath.Join(appData, "Ollama", "models", "blobs")
		defaultDest = filepath.Join(localAppData, "lm-studio", "ollama")
	} else {
		// POSIX-style defaults.
		configDir = filepath.Join(homeDir, ".config", "ollama-sync")
		defaultManifest = filepath.Join(homeDir, ".ollama", "models", "manifests")
		defaultBlob = filepath.Join(homeDir, ".ollama", "models", "blobs")
		defaultDest = filepath.Join(homeDir, ".cache", "lm-studio", "models", "ollama")
	}

	configFile := filepath.Join(configDir, "config.yaml")

	// If the config file does not exist, create it with default values.
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create config directory: %w", err)
		}

		defaultConfig := []byte(fmt.Sprintf(`manifest_dir: "%s"
blob_dir: "%s"
destinations:
  - "%s"
`, defaultManifest, defaultBlob, defaultDest))

		if err := os.WriteFile(configFile, defaultConfig, 0644); err != nil {
			return nil, fmt.Errorf("failed to write default config file: %w", err)
		}
	}

	viper.SetConfigFile(configFile)
	logrus.WithField("config_file", configFile).Info("Reading configuration from file")
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unable to decode config into struct: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"manifest_dir":  cfg.ManifestDir,
		"blob_dir":      cfg.BlobDir,
		"destinations":  cfg.Destinations,
	}).Info("Loaded configuration defaults")

	return &cfg, nil
}
