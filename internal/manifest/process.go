package manifest

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

// ProcessManifest reads and processes a single manifest file and creates symlinks in all destination directories.
// It returns the link paths it created or found up to date (none for manifests it skips), so that
// a full sync can prune links no manifest accounts for (see PruneStaleLinks).
func ProcessManifest(manifestPath, blobDir string, destDirs []string, logger *logrus.Logger) ([]string, error) {
	logger.WithField("manifest_path", manifestPath).Debug("Starting processing of manifest")

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest file: %w", err)
	}
	if isDowngradeGuard(manifestPath, data) {
		logger.WithField("manifest_path", manifestPath).Debug("Skipping Ollama downgrade guard tag")
		return nil, nil
	}

	data, ok, err := resolveManifestList(data, blobDir)
	if err != nil {
		return nil, err
	}
	if !ok {
		logger.WithField("manifest_path", manifestPath).Debug("Skipping manifest list with no GGUF variant")
		return nil, nil
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("failed to unmarshal manifest JSON: %w", err)
	}

	configDigest := NormalizeDigest(m.Config.Digest)
	modelConfigPath := filepath.Join(blobDir, configDigest)

	var modelFile string
	for _, layer := range m.Layers {
		if strings.HasSuffix(layer.MediaType, "model") {
			digest := NormalizeDigest(layer.Digest)
			modelFile = filepath.Join(blobDir, digest)
			break
		}
	}
	if modelFile == "" {
		// Not a GGUF model (e.g. an image model stored as raw tensors): nothing to link.
		logger.WithField("manifest_path", manifestPath).Debug("Skipping manifest with no GGUF model layer")
		return nil, nil
	}

	modelConfigData, err := os.ReadFile(modelConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read model config file: %w", err)
	}

	var modelConfig ModelConfig
	if err := json.Unmarshal(modelConfigData, &modelConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal model config JSON: %w", err)
	}

	modelName := modelNameFromManifestPath(manifestPath)
	tag := filepath.Base(manifestPath)

	var links []string
	for _, destDir := range destDirs {
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create destination directory %s: %w", destDir, err)
		}

		modelDir := filepath.Join(destDir, modelName)
		if err := os.MkdirAll(modelDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create model directory %s: %w", modelDir, err)
		}

		fileType := modelConfig.FileType
		if fileType == "unknown" || fileType == "" {
			fileType = tag
		}

		symlinkName := filepath.Join(modelDir,
			fmt.Sprintf("%s-%s-%s.%s", modelName, modelConfig.ModelType, fileType, modelConfig.ModelFormat))

		if isUpToDate(symlinkName, modelFile) {
			logger.WithField("symlink", symlinkName).Debug("Symlink already up to date")
			links = append(links, symlinkName)
			continue
		}
		if _, err := os.Lstat(symlinkName); err == nil {
			if err := os.Remove(symlinkName); err != nil {
				logger.WithField("symlink", symlinkName).Warn("Failed to remove existing symlink")
			} else {
				logger.WithField("symlink", symlinkName).Info("Removed stale symlink")
			}
		}

		// Create symlink with a Windows fallback.
		if runtime.GOOS == "windows" {
			if err := os.Symlink(modelFile, symlinkName); err != nil {
				logger.WithError(err).Warn("Symlink creation failed on Windows; attempting file copy as fallback")
				if copyErr := copyFile(modelFile, symlinkName); copyErr != nil {
					logger.WithError(copyErr).Error("Failed to create file copy as fallback")
					return nil, fmt.Errorf("failed to create file copy as fallback: %w", copyErr)
				}
			} else {
				logger.WithField("symlink", symlinkName).Info("Successfully created symlink on Windows")
			}
		} else {
			if err := os.Symlink(modelFile, symlinkName); err != nil {
				logger.WithError(err).Error("Failed to create symbolic link")
				return nil, fmt.Errorf("failed to create symbolic link from %s to %s: %w", modelFile, symlinkName, err)
			} else {
				logger.WithField("symlink", symlinkName).Info("Successfully created symbolic link")
			}
		}

		// Verify that the symlink exists
		if _, err := os.Lstat(symlinkName); err != nil {
			logger.WithError(err).Error("Symlink does not exist after creation")
			return nil, fmt.Errorf("symlink does not exist after creation: %w", err)
		}

		logger.WithFields(logrus.Fields{
			"model_file": modelFile,
			"symlink":    symlinkName,
		}).Debug("Completed symlink creation process")
		links = append(links, symlinkName)
	}

	return links, nil
}

// modelNameFromManifestPath derives the destination folder name from a
// manifests/<registry>/<namespace>/<model>/<tag> path. Models from the default
// registry and library namespace keep their bare name; others are prefixed so
// that e.g. hf.co/user/llama3 and library/llama3 don't share a folder.
func modelNameFromManifestPath(manifestPath string) string {
	modelDir := filepath.Dir(manifestPath)
	namespaceDir := filepath.Dir(modelDir)
	registry := filepath.Base(filepath.Dir(namespaceDir))
	namespace := filepath.Base(namespaceDir)

	var parts []string
	if !isDefaultRegistry(registry) {
		parts = append(parts, registry)
	}
	if !isDefaultRegistry(registry) || namespace != "library" {
		parts = append(parts, namespace)
	}
	return strings.Join(append(parts, filepath.Base(modelDir)), "-")
}

// isUpToDate reports whether linkPath already provides modelFile, either as a
// symlink to it or as the full-size copy made by the Windows fallback.
func isUpToDate(linkPath, modelFile string) bool {
	info, err := os.Lstat(linkPath)
	if err != nil {
		return false
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(linkPath)
		return err == nil && target == modelFile
	}
	if !info.Mode().IsRegular() {
		return false
	}
	blob, err := os.Stat(modelFile)
	if err != nil {
		return false
	}
	return info.Size() == blob.Size() && !info.ModTime().Before(blob.ModTime())
}

func copyFile(src, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}
