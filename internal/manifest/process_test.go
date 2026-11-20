package manifest

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
)

// fixture lays out an Ollama-style blob store and manifest in a temp dir.
type fixture struct {
	blobDir      string
	manifestPath string
	destDir      string
	modelBlob    string
	symlinkPath  string
}

func newFixture(t *testing.T, withModelLayer bool) fixture {
	t.Helper()
	root := t.TempDir()
	f := fixture{
		blobDir:      filepath.Join(root, "blobs"),
		manifestPath: filepath.Join(root, "manifests", "registry.ollama.ai", "library", "mymodel", "q8_0"),
		destDir:      filepath.Join(root, "dest"),
	}
	mustWrite(t, filepath.Join(f.blobDir, "sha256-cfg"),
		`{"file_type":"Q8_0","model_format":"gguf","model_type":"7B"}`)
	f.modelBlob = filepath.Join(f.blobDir, "sha256-weights")
	mustWrite(t, f.modelBlob, "weights")

	layers := `[{"mediaType":"application/vnd.ollama.image.tensor","digest":"sha256:t1"}]`
	if withModelLayer {
		layers = `[{"mediaType":"application/vnd.ollama.image.model","digest":"sha256:weights"}]`
	}
	mustWrite(t, f.manifestPath, `{"config":{"digest":"sha256:cfg"},"layers":`+layers+`}`)
	f.symlinkPath = filepath.Join(f.destDir, "mymodel", "mymodel-7B-Q8_0.gguf")
	return f
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func quietLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	return logger
}

func (f fixture) process(t *testing.T) {
	t.Helper()
	if _, err := ProcessManifest(f.manifestPath, f.blobDir, []string{f.destDir}, quietLogger()); err != nil {
		t.Fatalf("ProcessManifest: %v", err)
	}
}

func TestCreatesSymlinkToModelBlob(t *testing.T) {
	f := newFixture(t, true)
	f.process(t)
	target, err := os.Readlink(f.symlinkPath)
	if err != nil || target != f.modelBlob {
		t.Fatalf("symlink target = %q, %v; want %q", target, err, f.modelBlob)
	}
}

func TestUpToDateSymlinkIsLeftAlone(t *testing.T) {
	f := newFixture(t, true)
	f.process(t)
	before, _ := os.Lstat(f.symlinkPath)
	f.process(t)
	after, _ := os.Lstat(f.symlinkPath)
	if !os.SameFile(before, after) {
		t.Fatal("up-to-date symlink was deleted and recreated")
	}
}

func TestStaleSymlinkIsReplaced(t *testing.T) {
	f := newFixture(t, true)
	if err := os.MkdirAll(filepath.Dir(f.symlinkPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(f.blobDir, "old-blob"), f.symlinkPath); err != nil {
		t.Fatal(err)
	}
	f.process(t)
	if target, _ := os.Readlink(f.symlinkPath); target != f.modelBlob {
		t.Fatalf("stale symlink not replaced: points to %q", target)
	}
}

func TestManifestWithoutGGUFLayerIsSkipped(t *testing.T) {
	f := newFixture(t, false) // e.g. an image model stored as tensors
	f.process(t)              // must not return an error
	if _, err := os.Lstat(filepath.Join(f.destDir, "mymodel")); !os.IsNotExist(err) {
		t.Fatalf("expected nothing created for non-GGUF model, got err=%v", err)
	}
}

func TestModelNameFromManifestPath(t *testing.T) {
	root := filepath.Join("home", ".ollama", "models", "manifests")
	cases := map[string]string{
		filepath.Join(root, "registry.ollama.ai", "library", "llama3", "latest"):  "llama3",
		filepath.Join(root, "registry.ollama.ai", "someuser", "llama3", "latest"): "someuser-llama3",
		filepath.Join(root, "hf.co", "someuser", "llama3", "latest"):              "hf.co-someuser-llama3",
	}
	for path, want := range cases {
		if got := modelNameFromManifestPath(path); got != want {
			t.Errorf("modelNameFromManifestPath(%q) = %q; want %q", path, got, want)
		}
	}
}

func TestSameModelNameFromDifferentRegistriesDoesNotCollide(t *testing.T) {
	f := newFixture(t, true)
	hfManifest := filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(f.manifestPath)))),
		"hf.co", "someuser", "mymodel", "q8_0")
	hfBlob := filepath.Join(f.blobDir, "sha256-hfweights")
	mustWrite(t, hfBlob, "hf weights")
	mustWrite(t, hfManifest,
		`{"config":{"digest":"sha256:cfg"},"layers":[{"mediaType":"application/vnd.ollama.image.model","digest":"sha256:hfweights"}]}`)

	f.process(t)
	if _, err := ProcessManifest(hfManifest, f.blobDir, []string{f.destDir}, quietLogger()); err != nil {
		t.Fatal(err)
	}

	if target, _ := os.Readlink(f.symlinkPath); target != f.modelBlob {
		t.Fatalf("library symlink points to %q; want %q", target, f.modelBlob)
	}
	hfLink := filepath.Join(f.destDir, "hf.co-someuser-mymodel", "hf.co-someuser-mymodel-7B-Q8_0.gguf")
	if target, _ := os.Readlink(hfLink); target != hfBlob {
		t.Fatalf("hf.co symlink points to %q; want %q", target, hfBlob)
	}
}

func TestFallbackCopyIsLeftAlone(t *testing.T) {
	f := newFixture(t, true)
	if err := copyFile(f.modelBlob, mustMkdirFor(t, f.symlinkPath)); err != nil {
		t.Fatal(err)
	}
	f.process(t)
	info, err := os.Lstat(f.symlinkPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("up-to-date fallback copy was replaced")
	}
}

func TestStaleFallbackCopyIsReplaced(t *testing.T) {
	f := newFixture(t, true)
	mustWrite(t, f.symlinkPath, "an older, different-sized blob")
	f.process(t)
	if target, _ := os.Readlink(f.symlinkPath); target != f.modelBlob {
		t.Fatalf("stale copy not replaced: points to %q", target)
	}
}

func mustMkdirFor(t *testing.T, path string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	return path
}
