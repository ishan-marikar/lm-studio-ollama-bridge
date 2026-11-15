package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	// Ollama 0.40+ may store a model tag as a list of per-runner variants.
	manifestListMediaType = "application/vnd.ollama.manifest.list.v2+json"
	// The variant laid out for llama.cpp (separate projector), which LM Studio and
	// llama-server load; the "ggml" variant is for Ollama's own engine.
	llamaCppRunner = "llamacpp"
	ggufFormat     = "gguf"
)

// ManifestList is Ollama's index of per-runner variant manifests for one tag.
type ManifestList struct {
	MediaType string        `json:"mediaType"`
	Manifests []VariantInfo `json:"manifests"`
}

// VariantInfo points at one variant's manifest, stored as a blob.
type VariantInfo struct {
	Digest string `json:"digest"`
	Runner string `json:"runner"`
	Format string `json:"format"`
}

// chooseVariant prefers the llama.cpp variant, else the first GGUF one.
func chooseVariant(variants []VariantInfo) (VariantInfo, bool) {
	var first *VariantInfo
	for i, v := range variants {
		if v.Format != ggufFormat {
			continue
		}
		if v.Runner == llamaCppRunner {
			return v, true
		}
		if first == nil {
			first = &variants[i]
		}
	}
	if first == nil {
		return VariantInfo{}, false
	}
	return *first, true
}

// resolveManifestList returns the manifest JSON to process: data itself, or,
// when data is a manifest list, the chosen variant's manifest from blobDir.
// ok is false when a list has no GGUF variant.
func resolveManifestList(data []byte, blobDir string) (resolved []byte, ok bool, err error) {
	var list ManifestList
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, false, fmt.Errorf("failed to unmarshal manifest JSON: %w", err)
	}
	if list.MediaType != manifestListMediaType {
		return data, true, nil
	}
	variant, found := chooseVariant(list.Manifests)
	if !found {
		return nil, false, nil
	}
	resolved, err = os.ReadFile(filepath.Join(blobDir, NormalizeDigest(variant.Digest)))
	if err != nil {
		return nil, false, fmt.Errorf("failed to read %s variant manifest: %w", variant.Runner, err)
	}
	return resolved, true, nil
}
