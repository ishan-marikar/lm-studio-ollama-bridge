package manifest

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// v2Link adds a manifests-v2 entry the way Ollama 0.40 does: a symlink to the
// manifest JSON stored as a blob.
func v2Link(t *testing.T, modelsDir string, parts ...string) string {
	t.Helper()
	blob := filepath.Join(modelsDir, "blobs", "sha256-"+filepath.Join(parts...))
	blob = filepath.Join(filepath.Dir(blob), filepath.Base(blob)+"-manifest")
	mustWrite(t, blob, `{}`)
	link := filepath.Join(append([]string{modelsDir, "manifests-v2"}, parts...)...)
	if err := os.MkdirAll(filepath.Dir(link), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(blob, link); err != nil {
		t.Fatal(err)
	}
	return link
}

func legacyFile(t *testing.T, modelsDir string, parts ...string) string {
	t.Helper()
	path := filepath.Join(append([]string{modelsDir, "manifests"}, parts...)...)
	mustWrite(t, path, `{}`)
	return path
}

func findModelManifests(t *testing.T, modelsDir string) []string {
	t.Helper()
	got, err := FindModelManifests(filepath.Join(modelsDir, "manifests"))
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestFindModelManifestsIncludesV2OnlyModels(t *testing.T) {
	dir := t.TempDir()
	legacy := legacyFile(t, dir, "registry.ollama.ai", "library", "llama3", "latest")
	v2 := v2Link(t, dir, "hf.co", "someuser", "mymodel-GGUF", "Q8_0")

	want := []string{legacy, v2}
	sort.Strings(want) // results are sorted by path
	if got := findModelManifests(t, dir); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v; want %v", got, want)
	}
}

func TestFindModelManifestsPrefersV2ForSameModel(t *testing.T) {
	dir := t.TempDir()
	legacyFile(t, dir, "hf.co", "someuser", "mymodel-GGUF", "Q8_0")
	v2 := v2Link(t, dir, "hf.co", "someuser", "mymodel-GGUF", "Q8_0")

	if got := findModelManifests(t, dir); !reflect.DeepEqual(got, []string{v2}) {
		t.Fatalf("got %v; want only the v2 manifest %v", got, v2)
	}
}

func TestFindModelManifestsTreatsOllamaComAsDefaultRegistry(t *testing.T) {
	dir := t.TempDir()
	legacyFile(t, dir, "registry.ollama.ai", "library", "qwen3.6", "35b-a3b-q4_K_M")
	v2 := v2Link(t, dir, "ollama.com", "library", "qwen3.6", "35b-a3b-q4_K_M")

	if got := findModelManifests(t, dir); !reflect.DeepEqual(got, []string{v2}) {
		t.Fatalf("got %v; want only the v2 manifest %v", got, v2)
	}
}

func TestFindModelManifestsFallsBackWhenV2LinkDangles(t *testing.T) {
	dir := t.TempDir()
	legacy := legacyFile(t, dir, "hf.co", "someuser", "mymodel-GGUF", "Q8_0")
	v2 := v2Link(t, dir, "hf.co", "someuser", "mymodel-GGUF", "Q8_0")
	target, _ := os.Readlink(v2)
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}

	if got := findModelManifests(t, dir); !reflect.DeepEqual(got, []string{legacy}) {
		t.Fatalf("got %v; want the legacy manifest %v", got, legacy)
	}
}

func TestFindModelManifestsWithoutV2Dir(t *testing.T) {
	dir := t.TempDir()
	legacy := legacyFile(t, dir, "registry.ollama.ai", "library", "llama3", "latest")
	if got := findModelManifests(t, dir); !reflect.DeepEqual(got, []string{legacy}) {
		t.Fatalf("got %v; want %v", got, []string{legacy})
	}
}

func TestModelNameFromV2ManifestPath(t *testing.T) {
	root := filepath.Join("home", ".ollama", "models", "manifests-v2")
	cases := map[string]string{
		filepath.Join(root, "ollama.com", "library", "qwen3.6", "35b-a3b-q4_K_M"): "qwen3.6",
		filepath.Join(root, "ollama.com", "someuser", "llama3", "latest"):         "someuser-llama3",
		filepath.Join(root, "hf.co", "someuser", "llama3", "latest"):              "hf.co-someuser-llama3",
	}
	for path, want := range cases {
		if got := modelNameFromManifestPath(path); got != want {
			t.Errorf("modelNameFromManifestPath(%q) = %q; want %q", path, got, want)
		}
	}
}
