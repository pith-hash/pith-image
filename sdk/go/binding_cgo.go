// SPDX-License-Identifier: MIT
// Copyright (c) 2026 pith-hash

//go:build !windows && cgo

package pithimage

/*
#include <dlfcn.h>
#include <stddef.h>
#include <stdint.h>
#include <stdlib.h>

typedef int32_t (*pith_phash_fn)(const uint8_t *, size_t, uint32_t, uint32_t, uint32_t, uint64_t *);

static int32_t pith_call_phash(void *fn, const uint8_t *data, size_t len,
                               uint32_t width, uint32_t height, uint32_t layout,
                               uint64_t *out) {
    return ((pith_phash_fn)fn)(data, len, width, height, layout, out);
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// ffiPhash opens the cdylib at libPath, resolves pith_image_phash and
// calls it. The handle is released before returning; repeated calls
// reuse the loader's own refcount.
func ffiPhash(libPath string, data *byte, n int, width, height, layout uint32, out *uint64) (int32, error) {
	cPath := C.CString(libPath)
	defer C.free(unsafe.Pointer(cPath))
	handle := C.dlopen(cPath, C.RTLD_NOW|C.RTLD_LOCAL)
	if handle == nil {
		// The failure mode is recorded by dlerror; surface it verbatim.
		msg := "unknown dlopen failure"
		if e := C.dlerror(); e != nil {
			msg = C.GoString(e)
		}
		return 0, fmt.Errorf("pithimage: dlopen(%s): %s", libPath, msg)
	}
	defer C.dlclose(handle)

	cName := C.CString("pith_image_phash")
	defer C.free(unsafe.Pointer(cName))
	sym := C.dlsym(handle, cName)
	if sym == nil {
		return 0, fmt.Errorf("pithimage: symbol pith_image_phash missing from %s", libPath)
	}
	var hash C.uint64_t
	rc := C.pith_call_phash(sym, (*C.uint8_t)(unsafe.Pointer(data)), C.size_t(n),
		C.uint32_t(width), C.uint32_t(height), C.uint32_t(layout), &hash)
	*out = uint64(hash)
	return int32(rc), nil
}
