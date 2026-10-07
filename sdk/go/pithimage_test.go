// SPDX-License-Identifier: MIT
// Copyright (c) 2026 pith-hash

package pithimage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// repoRoot resolves the repository root relative to this package
// (sdk/go -> sdk -> repo root), the anchor for reference.json and the
// committed fixtures.
func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if st, err := os.Stat(filepath.Join(root, "reference.json")); err != nil || st.IsDir() {
		t.Fatalf("reference.json not found at %s", root)
	}
	return root
}

// vectors mirrors the committed VECTORS table in src/reference.rs:
// name -> (layout, width, height). RawLayout declaration order is the
// wire code the C ABI speaks.
var vectors = []struct {
	Name          string
	Layout        Layout
	Width, Height uint32
}{
	{"base_444", Rgb8, 32, 24},
	{"phash_adam7_37x29", Rgb8, 37, 29},
	{"phash_gray16_36x28", Gray16, 36, 28},
	{"phash_gray8_40x32", Gray8, 40, 32},
	{"phash_pal8_trns_52x44", Rgba8, 52, 44},
	{"phash_rgb16_40x24", Rgb16, 40, 24},
	{"phash_rgb8_48x40", Rgb8, 48, 40},
	{"phash_rgba8_48x40", Rgba8, 48, 40},
	{"phash_small_13x9", Rgb8, 13, 9},
}

// TestReferenceVectorsHexExact replays every committed reference.json
// vector through the cdylib and compares hex-exact — the same vectors
// the Rust gen-reference verify gate and the Python/Node SDKs check.
func TestReferenceVectorsHexExact(t *testing.T) {
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "reference.json"))
	if err != nil {
		t.Fatal(err)
	}
	var reference struct {
		Vectors map[string]string `json:"vectors"`
	}
	if err := json.Unmarshal(raw, &reference); err != nil {
		t.Fatal(err)
	}

	for _, v := range vectors {
		t.Run(v.Name, func(t *testing.T) {
			dump, err := os.ReadFile(filepath.Join(root, "tests", "fixtures", "phash", v.Name+".raw"))
			if err != nil {
				t.Fatal(err)
			}
			expected, ok := reference.Vectors[v.Name]
			if !ok {
				t.Fatalf("vector %s missing from reference.json", v.Name)
			}
			hash, err := Hash(dump, v.Width, v.Height, v.Layout)
			if err != nil {
				t.Fatalf("Hash(%s): %v", v.Name, err)
			}
			if got := fmt.Sprintf("%016x", hash); got != expected {
				t.Errorf("%s: got %s, want %s", v.Name, got, expected)
			}
		})
	}
}

// TestBase444RustPinned pins the oracle value carried twice upstream
// (src/reference.rs), so the binding fails loudly even if
// reference.json were regenerated wrongly.
func TestBase444RustPinned(t *testing.T) {
	dump, err := os.ReadFile(filepath.Join(repoRoot(t), "tests", "fixtures", "phash", "base_444.raw"))
	if err != nil {
		t.Fatal(err)
	}
	hash, err := Hash(dump, 32, 24, Rgb8)
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%016x", hash); got != "8d0a3f411ee50f7a" {
		t.Errorf("base_444: got %s, want 8d0a3f411ee50f7a", got)
	}
}

// TestInvalidArgumentsAreRefused checks the client-side refusals; the
// cdylib-side ones (truncated dump) come back as a status code, not a
// crash.
func TestInvalidArgumentsAreRefused(t *testing.T) {
	dump, err := os.ReadFile(filepath.Join(repoRoot(t), "tests", "fixtures", "phash", "base_444.raw"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Hash(dump, 32, 24, Layout(99)); err == nil {
		t.Error("unknown layout code must be refused")
	}
	_, err = Hash(dump[:len(dump)-1], 32, 24, Rgb8)
	var ffi *FfiError
	if !asFfiError(err, &ffi) || ffi.Status != StatusInvalid {
		t.Errorf("truncated dump: want StatusInvalid, got %v", err)
	}
}

func asFfiError(err error, target **FfiError) bool {
	if e, ok := err.(*FfiError); ok {
		*target = e
		return true
	}
	return false
}
