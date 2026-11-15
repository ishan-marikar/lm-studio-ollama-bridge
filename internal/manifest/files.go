package manifest

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var ignoredFiles = []string{
	".DS_Store", // macOS
	"Thumbs.db", // Windows
}

func isIgnoredFile(name string) bool {
	for _, ignored := range ignoredFiles {
		if strings.Contains(name, ignored) {
			return true
		}
	}
	return false
}

const (
	legacyManifestDirName = "manifests"
	v2ManifestDirName     = "manifests-v2"
	defaultRegistry       = "registry.ollama.ai"
	v2DefaultRegistry     = "ollama.com" // Ollama 0.40's name for defaultRegistry in manifests-v2
)

// isDefaultRegistry matches Ollama's own registry under either of its names.
func isDefaultRegistry(registry string) bool {
	return strings.EqualFold(registry, defaultRegistry) || strings.EqualFold(registry, v2DefaultRegistry)
}

// FindModelManifests returns one manifest per model tag from manifestDir and,
// when manifestDir is Ollama's manifests/ dir, its manifests-v2/ sibling (where
// Ollama 0.40+ writes every model it pulls or creates). As in Ollama, a v2
// manifest wins over a legacy one, unless its link is dangling.
func FindModelManifests(manifestDir string) ([]string, error) {
	chosen := map[string]string{}
	if err := addManifests(chosen, manifestDir); err != nil {
		return nil, err
	}
	if filepath.Base(manifestDir) == legacyManifestDirName {
		v2Dir := filepath.Join(filepath.Dir(manifestDir), v2ManifestDirName)
		if _, err := os.Stat(v2Dir); err == nil {
			if err := addManifests(chosen, v2Dir); err != nil {
				return nil, err
			}
		}
	}
	files := make([]string, 0, len(chosen))
	for _, path := range chosen {
		files = append(files, path)
	}
	sort.Strings(files)
	return files, nil
}

// addManifests records each readable manifest under root by model key,
// replacing any entry found earlier (so later roots take precedence).
func addManifests(chosen map[string]string, root string) error {
	paths, err := FindManifestFiles(root)
	if err != nil {
		return err
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			continue // dangling v2 link: keep the legacy manifest, as Ollama does
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		chosen[modelKey(rel)] = path
	}
	return nil
}

// modelKey identifies a model tag across both manifest layouts:
// <registry>/<namespace>/<model>/<tag> with the default registry's two names merged.
func modelKey(relPath string) string {
	parts := strings.Split(filepath.ToSlash(relPath), "/")
	if isDefaultRegistry(parts[0]) {
		parts[0] = defaultRegistry
	}
	return strings.Join(parts, "/")
}

func FindManifestFiles(manifestDir string) ([]string, error) {
	var files []string
	err := filepath.Walk(manifestDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && !isIgnoredFile(info.Name()) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}
