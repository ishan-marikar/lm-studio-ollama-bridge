package manifest

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strings"
)

// libraryNamespace is the namespace of Ollama's own library models.
const libraryNamespace = "library"

// isDowngradeGuard reports whether manifestPath is one of the hidden
// "llamacpp:<digest>" tags Ollama 0.40 writes when it converts a model, so
// that older Ollama versions keep the converted blobs. The tag is the sha256
// of the manifest itself. `ollama list` hides these tags the same way
// (server/model_list.go), so the bridge must not link them as a model.
func isDowngradeGuard(manifestPath string, data []byte) bool {
	modelDir := filepath.Dir(manifestPath)
	namespaceDir := filepath.Dir(modelDir)
	registry := filepath.Base(filepath.Dir(namespaceDir))
	if !isDefaultRegistry(registry) ||
		!strings.EqualFold(filepath.Base(namespaceDir), libraryNamespace) ||
		!strings.EqualFold(filepath.Base(modelDir), llamaCppRunner) {
		return false
	}
	sum := sha256.Sum256(data)
	return strings.EqualFold(filepath.Base(manifestPath), hex.EncodeToString(sum[:]))
}
