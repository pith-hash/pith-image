// SPDX-License-Identifier: MIT
// Copyright (c) 2026 pith-hash

// Package pithimage provides Go bindings for the pith-image Rust
// cdylib: the 64-bit image perceptual hash.
//
// The single Rust core (built by `cargo build --release`) is loaded at
// runtime; the package carries zero module dependencies. On unix the
// cdylib is opened with dlopen through cgo, on Windows with
// LoadLibrary through the standard syscall package — both resolve the
// library through the same discovery chain, so `go build ./... &&
// go test ./...` works unchanged on every OS the CD matrix builds.
//
// Discovery order (the suite's cdylib convention):
//
//  1. PITH_CDYLIB — an explicit cdylib file path;
//  2. PITH_CDYLIB_DIR — a directory scanned for the cdylib names (the
//     CD pipeline points this at target/release);
//  3. <repo root>/target/release — the repository working-tree layout,
//     anchored at this package's source directory, so a source
//     checkout runs against a local cargo build unconfigured.
//
// The vectors in reference.json at the repository root are the
// hex-exact cross-language source of truth; the conformance tests
// replay all of them through this binding.
package pithimage

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

// Status codes returned by the cdylib's C ABI.
const (
	// StatusOK: success.
	StatusOK int32 = 0
	// StatusInvalid: a caller argument is invalid (null pointer,
	// unknown layout code, length/geometry mismatch).
	StatusInvalid int32 = -1
	// StatusRejected: the core pipeline refused the input.
	StatusRejected int32 = -2
)

// Layout is the sample layout of a raw pixel dump, in RawLayout
// declaration order (the wire code the C ABI speaks).
type Layout uint32

// The raw dump layouts.
const (
	// Gray8: one u8 luma sample per pixel.
	Gray8 Layout = 0
	// Gray16: one u16 luma sample per pixel, little-endian.
	Gray16 Layout = 1
	// Rgb8: three u8 samples per pixel, R G B.
	Rgb8 Layout = 2
	// Rgb16: three u16 samples per pixel, R G B, little-endian.
	Rgb16 Layout = 3
	// Rgba8: four u8 samples per pixel, R G B A (alpha ignored by the
	// pHash luma step).
	Rgba8 Layout = 4
)

// cdylibNames are the file names cargo may drop into the build
// directory, per platform (windows / linux / macOS).
var cdylibNames = []string{"pith_image.dll", "libpith_image.so", "libpith_image.dylib"}

// FfiError reports a non-zero status code from the cdylib.
type FfiError struct {
	// Op is the FFI operation name.
	Op string
	// Status is the raw status code the FFI returned.
	Status int32
}

func (e *FfiError) Error() string {
	kind := "unknown failure"
	switch e.Status {
	case StatusInvalid:
		kind = "invalid argument"
	case StatusRejected:
		kind = "input rejected"
	}
	return fmt.Sprintf("%s failed: %s (status %d)", e.Op, kind, e.Status)
}

// FindCdylib locates the cdylib through the suite's discovery chain.
func FindCdylib() (string, error) {
	if p := os.Getenv("PITH_CDYLIB"); p != "" {
		if st, err := os.Stat(p); err == nil && st.Mode().IsRegular() {
			return filepath.Abs(p)
		}
	}
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("pithimage: cannot locate the package source directory")
	}
	pkgDir := filepath.Dir(thisFile)
	repoRoot := filepath.Dir(filepath.Dir(pkgDir)) // sdk/go -> sdk -> repo root

	var dirs []string
	if env := os.Getenv("PITH_CDYLIB_DIR"); env != "" {
		dirs = append(dirs, env)
		if !filepath.IsAbs(env) {
			dirs = append(dirs, filepath.Join(repoRoot, env))
		}
	}
	dirs = append(dirs, filepath.Join(repoRoot, "target", "release"))
	for _, dir := range dirs {
		for _, name := range cdylibNames {
			p := filepath.Join(dir, name)
			if st, err := os.Stat(p); err == nil && st.Mode().IsRegular() {
				return p, nil
			}
		}
	}
	return "", fmt.Errorf(
		"pithimage: no cdylib found (searched PITH_CDYLIB, PITH_CDYLIB_DIR and <repo>/target/release); run `cargo build --release` first",
	)
}

// locate resolves the cdylib path once per process.
var locate = sync.OnceValues(FindCdylib)

// Hash computes the 64-bit perceptual hash of a raw pixel dump.
//
// dump is a flat, row-major, channel-interleaved buffer of
// width*height*channels samples (little-endian for the 16-bit
// layouts) — the format the fixtures under tests/fixtures/phash/
// commit. Render the result with fmt.Sprintf("%016x", hash) to
// compare against reference.json.
func Hash(dump []byte, width, height uint32, layout Layout) (uint64, error) {
	if width == 0 || height == 0 {
		return 0, fmt.Errorf("pithimage: width and height must be positive")
	}
	switch layout {
	case Gray8, Gray16, Rgb8, Rgb16, Rgba8:
	default:
		return 0, fmt.Errorf("pithimage: unknown layout code %d", uint32(layout))
	}
	libPath, err := locate()
	if err != nil {
		return 0, err
	}
	var hash uint64
	var data *byte
	if len(dump) > 0 {
		data = &dump[0]
	}
	status, err := ffiPhash(libPath, data, len(dump), width, height, uint32(layout), &hash)
	if err != nil {
		return 0, err
	}
	if status != StatusOK {
		return 0, &FfiError{Op: "pith_image_phash", Status: status}
	}
	return hash, nil
}
