package manifest

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

// PruneStaleLinks removes links the bridge made that no current manifest accounts
// for: symlinks in <destDir>/<model>/ that point into blobDir and aren't in keep.
//
// Without it, a link outlives its model whenever the blob does: deleting one of two
// tags that share weights, or a change in how folders are named, leaves a working
// link to a model Ollama no longer has. Model folders the prune empties are removed.
// Anything else (links elsewhere, regular files such as the Windows copy fallback)
// is left alone, since the bridge can't tell it made them.
func PruneStaleLinks(destDirs []string, blobDir string, keep map[string]bool, logger *logrus.Logger) (int, error) {
	removed := 0
	blobDir = filepath.Clean(blobDir)
	for _, destDir := range destDirs {
		modelDirs, err := os.ReadDir(destDir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return removed, fmt.Errorf("failed to read destination %s: %w", destDir, err)
		}
		for _, modelDir := range modelDirs {
			if !modelDir.IsDir() {
				continue
			}
			n, err := pruneModelDir(filepath.Join(destDir, modelDir.Name()), blobDir, keep, logger)
			removed += n
			if err != nil {
				return removed, err
			}
		}
	}
	return removed, nil
}

func pruneModelDir(dirPath, blobDir string, keep map[string]bool, logger *logrus.Logger) (int, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return 0, fmt.Errorf("failed to read model directory %s: %w", dirPath, err)
	}
	removed := 0
	for _, entry := range entries {
		path := filepath.Join(dirPath, entry.Name())
		if keep[path] || !isLinkIntoBlobDir(path, blobDir) {
			continue
		}
		if err := os.Remove(path); err != nil {
			return removed, fmt.Errorf("failed to remove stale link %s: %w", path, err)
		}
		logger.WithField("symlink", path).Info("Removed link with no current manifest")
		removed++
	}
	if removed > 0 && removed == len(entries) {
		if err := os.Remove(dirPath); err != nil {
			return removed, fmt.Errorf("failed to remove empty model directory %s: %w", dirPath, err)
		}
	}
	return removed, nil
}

// isLinkIntoBlobDir reports whether path is a symlink to a file directly in blobDir,
// i.e. a link the bridge could have made (whether or not the blob still exists).
func isLinkIntoBlobDir(path, blobDir string) bool {
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		return false
	}
	target, err := os.Readlink(path)
	if err != nil {
		return false
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(path), target)
	}
	return filepath.Dir(filepath.Clean(target)) == blobDir
}
