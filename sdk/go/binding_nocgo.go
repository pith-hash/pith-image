// SPDX-License-Identifier: MIT
// Copyright (c) 2026 pith-hash

//go:build !windows && !cgo

package pithimage

import "fmt"

// ffiPhash is unavailable without cgo on unix: there is no pure-Go
// dlopen in the standard library. Build with CGO_ENABLED=1 (the CD
// pipeline always does).
func ffiPhash(string, *byte, int, uint32, uint32, uint32, *uint64) (int32, error) {
	return 0, fmt.Errorf("pithimage: cgo is required to load the cdylib on this platform (build with CGO_ENABLED=1)")
}
