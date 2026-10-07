// SPDX-License-Identifier: MIT
// Copyright (c) 2026 pith-hash

//go:build windows

package pithimage

import (
	"fmt"
	"syscall"
	"unsafe"
)

// ffiPhash loads the cdylib with LoadLibrary (absolute path, no PATH
// involvement), resolves pith_image_phash and calls it. The library is
// released before returning; repeated calls reuse the loader's own
// refcount. Pure standard library — no C toolchain is needed to build
// the windows leg of the binding.
func ffiPhash(libPath string, data *byte, n int, width, height, layout uint32, out *uint64) (int32, error) {
	lib, err := syscall.LoadLibrary(libPath)
	if err != nil {
		return 0, fmt.Errorf("pithimage: LoadLibrary(%s): %w", libPath, err)
	}
	defer syscall.FreeLibrary(lib)

	proc, err := syscall.GetProcAddress(lib, "pith_image_phash")
	if err != nil {
		return 0, fmt.Errorf("pithimage: symbol pith_image_phash missing from %s: %w", libPath, err)
	}
	var hash uint64
	rc, _, _ := syscall.SyscallN(
		proc,
		uintptr(unsafe.Pointer(data)),
		uintptr(n),
		uintptr(width),
		uintptr(height),
		uintptr(layout),
		uintptr(unsafe.Pointer(&hash)),
	)
	*out = hash
	return int32(rc), nil
}
