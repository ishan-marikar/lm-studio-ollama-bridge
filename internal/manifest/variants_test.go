package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

// listFixture lays out an Ollama 0.40 manifest list: a manifests-v2 entry whose
// JSON names per-runner variant manifests stored as blobs.
func listFixture(t *testing.T, runners ...string) (manifestPath, blobDir, destDir string) {
	t.Helper()
	root := t.TempDir()
	blobDir = filepath.Join(root, "blobs")
	destDir = filepath.Join(root, "dest")
	var refs string
	for i, runner := range runners {
		mustWrite(t, filepath.Join(blobDir, "sha256-cfg-"+runner),
			`{"file_type":"Q4_K_M","model_format":"gguf","model_type":"9B"}`)
		mustWrite(t, filepath.Join(blobDir, "sha256-weights-"+runner), runner)
		mustWrite(t, filepath.Join(blobDir, "sha256-variant-"+runner),
			`{"config":{"digest":"sha256:cfg-`+runner+`"},"layers":[`+
				`{"mediaType":"application/vnd.ollama.image.model","digest":"sha256:weights-`+runner+`"}]}`)
		if i > 0 {
			refs += ","
		}
		refs += `{"mediaType":"application/vnd.docker.distribution.manifest.v2+json",` +
			`"digest":"sha256:variant-` + runner + `","runner":"` + runner + `","format":"gguf"}`
	}
	manifestPath = filepath.Join(root, "manifests-v2", "ollama.com", "library", "qwen3.5", "9b-q4_K_M")
	mustWrite(t, manifestPath,
		`{"schemaVersion":2,"mediaType":"application/vnd.ollama.manifest.list.v2+json","manifests":[`+refs+`]}`)
	return manifestPath, blobDir, destDir
}

func linkedWeights(t *testing.T, manifestPath, blobDir, destDir string) string {
	t.Helper()
	if _, err := ProcessManifest(manifestPath, blobDir, []string{destDir}, quietLogger()); err != nil {
		t.Fatalf("ProcessManifest: %v", err)
	}
	target, err := os.Readlink(filepath.Join(destDir, "qwen3.5", "qwen3.5-9B-Q4_K_M.gguf"))
	if err != nil {
		t.Fatalf("no symlink created: %v", err)
	}
	return filepath.Base(target)
}

func TestManifestListLinksLlamaCppVariant(t *testing.T) {
	manifestPath, blobDir, destDir := listFixture(t, "ggml", "llamacpp")
	if got := linkedWeights(t, manifestPath, blobDir, destDir); got != "sha256-weights-llamacpp" {
		t.Fatalf("linked %s; want the llamacpp variant's weights", got)
	}
}

func TestManifestListWithoutLlamaCppVariantLinksFirstGGUF(t *testing.T) {
	manifestPath, blobDir, destDir := listFixture(t, "ggml")
	if got := linkedWeights(t, manifestPath, blobDir, destDir); got != "sha256-weights-ggml" {
		t.Fatalf("linked %s; want the only variant's weights", got)
	}
}
