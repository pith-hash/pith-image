# SPDX-License-Identifier: MIT
# Copyright (c) 2026 pith-hash
"""pith-image SDK: the 64-bit image perceptual hash through ctypes.

The single Rust core (the ``pith-image`` cdylib built by
``cargo build --release``) is loaded at runtime; this package carries
no third-party dependency — ``ctypes`` is the standard library.

Discovery order (the suite's cdylib convention):

1. ``PITH_CDYLIB`` — an explicit cdylib *file* path;
2. ``PITH_CDYLIB_DIR`` — a *directory* scanned for the cdylib names
   (the CD pipeline points this at ``target/release``);
3. the package directory itself (the built wheel ships the cdylib as
   package data);
4. ``<repo root>/target/release`` — the repository working-tree layout,
   so a source checkout runs against a local cargo build with no
   configuration.

The vectors in ``reference.json`` at the repository root are the
hex-exact cross-language source of truth; the conformance tests under
``tests/`` replay all of them through this binding.
"""

from __future__ import annotations

import ctypes
import os
from pathlib import Path

__all__ = [
    "FfiError",
    "LibraryNotFoundError",
    "find_cdylib",
    "image_phash",
    "LAYOUT_GRAY8",
    "LAYOUT_GRAY16",
    "LAYOUT_RGB8",
    "LAYOUT_RGB16",
    "LAYOUT_RGBA8",
    "STATUS_OK",
    "STATUS_INVALID",
    "STATUS_REJECTED",
]

#: Status: success.
STATUS_OK = 0
#: Status: a caller argument is invalid (null pointer, unknown layout
#: code, length/geometry mismatch).
STATUS_INVALID = -1
#: Status: the core pipeline refused the input.
STATUS_REJECTED = -2

#: Raw dump layouts, in ``RawLayout`` declaration order.
LAYOUT_GRAY8 = 0
LAYOUT_GRAY16 = 1
LAYOUT_RGB8 = 2
LAYOUT_RGB16 = 3
LAYOUT_RGBA8 = 4

#: Every cdylib file name cargo may drop into the build directory, per
#: platform (windows / linux / macOS).
CDYLIB_NAMES = ("pith_image.dll", "libpith_image.so", "libpith_image.dylib")


class LibraryNotFoundError(OSError):
    """No cdylib was found through the discovery chain."""


class FfiError(Exception):
    """A non-zero status code came back from the cdylib."""

    def __init__(self, op: str, status: int) -> None:
        kind = {
            STATUS_INVALID: "invalid argument",
            STATUS_REJECTED: "input rejected",
        }.get(status, "unknown failure")
        super().__init__(f"{op} failed: {kind} (status {status})")
        #: The raw status code the FFI returned.
        self.status = status


def find_cdylib() -> Path:
    """Locates the cdylib through the suite's discovery chain."""
    explicit = os.environ.get("PITH_CDYLIB")
    if explicit:
        p = Path(explicit)
        if p.is_file():
            return p
    env_dir = os.environ.get("PITH_CDYLIB_DIR")
    candidates: list[Path] = []
    if env_dir:
        env_dir_path = Path(env_dir)
        candidates.append(env_dir_path)
        if not env_dir_path.is_absolute():
            # CD and local runs invoke tools from the repository root or
            # from sdk/<lang>; resolve the env value against both.
            candidates.append(Path.cwd() / env_dir_path)
            candidates.append(Path(__file__).resolve().parents[3] / env_dir_path)
    candidates.append(Path(__file__).resolve().parent)  # packaged wheel
    candidates.append(Path(__file__).resolve().parents[3] / "target" / "release")
    for directory in candidates:
        for name in CDYLIB_NAMES:
            p = directory / name
            if p.is_file():
                return p
    raise LibraryNotFoundError(
        "no pith-image cdylib found (searched PITH_CDYLIB, PITH_CDYLIB_DIR, "
        "the package directory and <repo>/target/release); "
        "run `cargo build --release` first"
    )


_lib: ctypes.CDLL | None = None


def _load() -> ctypes.CDLL:
    global _lib
    if _lib is None:
        lib = ctypes.CDLL(str(find_cdylib()))
        lib.pith_image_phash.argtypes = [
            ctypes.c_void_p,  # data
            ctypes.c_size_t,  # len
            ctypes.c_uint32,  # width
            ctypes.c_uint32,  # height
            ctypes.c_uint32,  # layout
            ctypes.POINTER(ctypes.c_uint64),  # out hash
        ]
        lib.pith_image_phash.restype = ctypes.c_int32
        _lib = lib
    return _lib


def image_phash(dump: bytes, width: int, height: int, layout: int) -> int:
    """Computes the 64-bit perceptual hash of a raw pixel dump.

    ``dump`` is a flat, row-major, channel-interleaved buffer of
    ``width * height * channels`` samples (little-endian for the 16-bit
    layouts) — the format the fixtures under ``tests/fixtures/phash/``
    commit. Returns the hash as an ``int``; render it with
    ``format(hash, "016x")`` to compare against ``reference.json``.
    """
    if width <= 0 or height <= 0:
        raise ValueError("width and height must be positive")
    if layout not in (LAYOUT_GRAY8, LAYOUT_GRAY16, LAYOUT_RGB8, LAYOUT_RGB16, LAYOUT_RGBA8):
        raise ValueError(f"unknown layout code {layout}")
    out = ctypes.c_uint64()
    status = _load().pith_image_phash(dump, len(dump), width, height, layout, ctypes.byref(out))
    if status != STATUS_OK:
        raise FfiError("pith_image_phash", status)
    return out.value
