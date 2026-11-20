package manifest

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

const guardManifest = `{"config":{"digest":"sha256:cfg"},"layers":[` +
	`{"mediaType":"application/vnd.ollama.image.model","digest":"sha256:weights"}]}`

// guardFixture writes a legacy library/llamacpp/<tag> manifest with its blobs.
func guardFixture(t *testing.T, tag string) (manifestPath, blobDir, destDir string) {
	t.Helper()
	root := t.TempDir()
	blobDir = filepath.Join(root, "blobs")
	destDir = filepath.Join(root, "dest")
	mustWrite(t, filepath.Join(blobDir, "sha256-cfg"),
		`{"file_type":"Q4_K_M","model_format":"gguf","model_type":"9B"}`)
	mustWrite(t, filepath.Join(blobDir, "sha256-weights"), "weights")
	manifestPath = filepath.Join(root, "manifests", "registry.ollama.ai", "library", "llamacpp", tag)
	mustWrite(t, manifestPath, guardManifest)
	return manifestPath, blobDir, destDir
}

func TestDowngradeGuardTagIsNotLinked(t *testing.T) {
	sum := sha256.Sum256([]byte(guardManifest))
	manifestPath, blobDir, destDir := guardFixture(t, hex.EncodeToString(sum[:]))
	if _, err := ProcessManifest(manifestPath, blobDir, []string{destDir}, quietLogger()); err != nil {
		t.Fatalf("ProcessManifest: %v", err)
	}
	if _, err := os.Stat(filepath.Join(destDir, "llamacpp")); !os.IsNotExist(err) {
		t.Fatalf("downgrade guard tag was linked (stat err: %v)", err)
	}
}

func TestModelNamedLlamaCppIsStillLinked(t *testing.T) {
	manifestPath, blobDir, destDir := guardFixture(t, "latest")
	if _, err := ProcessManifest(manifestPath, blobDir, []string{destDir}, quietLogger()); err != nil {
		t.Fatalf("ProcessManifest: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(destDir, "llamacpp", "llamacpp-9B-Q4_K_M.gguf")); err != nil {
		t.Fatalf("model named llamacpp was not linked: %v", err)
	}
}
