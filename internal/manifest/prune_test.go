package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func linkTo(t *testing.T, target, linkPath string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(linkPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, linkPath); err != nil {
		t.Fatal(err)
	}
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func TestProcessManifestReturnsItsLinks(t *testing.T) {
	f := newFixture(t, true)
	for run := 0; run < 2; run++ { // created, then already up to date
		links, err := ProcessManifest(f.manifestPath, f.blobDir, []string{f.destDir}, quietLogger())
		if err != nil {
			t.Fatal(err)
		}
		if len(links) != 1 || links[0] != f.symlinkPath {
			t.Fatalf("run %d: links = %v, want [%s]", run, links, f.symlinkPath)
		}
	}
}

// A tag deleted while another tag keeps its blob leaves a working link behind;
// pruning removes it (and its now-empty folder) but keeps the current link.
func TestPruneRemovesLinkForDeletedTagWhoseBlobSurvives(t *testing.T) {
	f := newFixture(t, true)
	links, err := ProcessManifest(f.manifestPath, f.blobDir, []string{f.destDir}, quietLogger())
	if err != nil {
		t.Fatal(err)
	}
	staleDir := filepath.Join(f.destDir, "oldname")
	stale := filepath.Join(staleDir, "oldname-7B-Q8_0.gguf")
	linkTo(t, f.modelBlob, stale) // same blob, no manifest any more

	removed, err := PruneStaleLinks([]string{f.destDir}, f.blobDir, keepSet(links), quietLogger())
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 || exists(stale) || exists(staleDir) {
		t.Fatalf("removed=%d, stale link exists=%v, folder exists=%v; want 1, false, false",
			removed, exists(stale), exists(staleDir))
	}
	if !exists(f.symlinkPath) {
		t.Fatal("current link was removed")
	}
}

func TestPruneRemovesDanglingLinkIntoBlobDir(t *testing.T) {
	f := newFixture(t, true)
	gone := filepath.Join(f.destDir, "deleted", "deleted-7B-Q8_0.gguf")
	linkTo(t, filepath.Join(f.blobDir, "sha256-gone"), gone)
	if removed, err := PruneStaleLinks([]string{f.destDir}, f.blobDir, map[string]bool{}, quietLogger()); err != nil || removed != 1 {
		t.Fatalf("removed=%d err=%v; want 1, nil", removed, err)
	}
}

// Anything the bridge couldn't have made is left alone.
func TestPruneKeepsFilesTheBridgeDidNotMake(t *testing.T) {
	f := newFixture(t, true)
	elsewhere := filepath.Join(t.TempDir(), "hand-placed.gguf")
	mustWrite(t, elsewhere, "weights")
	foreignLink := filepath.Join(f.destDir, "mine", "hand-placed.gguf")
	linkTo(t, elsewhere, foreignLink)
	regular := filepath.Join(f.destDir, "copies", "copied.gguf")
	mustWrite(t, regular, "weights")

	if removed, err := PruneStaleLinks([]string{f.destDir}, f.blobDir, map[string]bool{}, quietLogger()); err != nil || removed != 0 {
		t.Fatalf("removed=%d err=%v; want 0, nil", removed, err)
	}
	if !exists(foreignLink) || !exists(regular) {
		t.Fatal("pruned a file the bridge didn't make")
	}
}

func TestPruneIgnoresMissingDestination(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope")
	if removed, err := PruneStaleLinks([]string{missing}, "/blobs", map[string]bool{}, quietLogger()); err != nil || removed != 0 {
		t.Fatalf("removed=%d err=%v; want 0, nil", removed, err)
	}
}

func keepSet(paths []string) map[string]bool {
	keep := map[string]bool{}
	for _, path := range paths {
		keep[path] = true
	}
	return keep
}
