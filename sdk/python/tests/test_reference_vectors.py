# SPDX-License-Identifier: MIT
# Copyright (c) 2026 pith-hash
"""Hex-exact conformance: the committed reference vectors through ctypes.

Every vector in the repository-root ``reference.json`` is replayed
through the cdylib and compared hex-exact, the same vectors the Rust
``gen-reference verify`` gate and the Node/Go SDKs check.
"""

from __future__ import annotations

import json
from pathlib import Path

import pytest

from pith_image import (
    LAYOUT_GRAY8,
    LAYOUT_GRAY16,
    LAYOUT_RGB8,
    LAYOUT_RGB16,
    LAYOUT_RGBA8,
    FfiError,
    find_cdylib,
    image_phash,
)

REPO_ROOT = Path(__file__).resolve().parents[3]

# name -> (layout, width, height), per the committed VECTORS table in
# src/reference.rs (RawLayout declaration order is the wire code).
VECTORS: dict[str, tuple[int, int, int]] = {
    "base_444": (LAYOUT_RGB8, 32, 24),
    "phash_adam7_37x29": (LAYOUT_RGB8, 37, 29),
    "phash_gray16_36x28": (LAYOUT_GRAY16, 36, 28),
    "phash_gray8_40x32": (LAYOUT_GRAY8, 40, 32),
    "phash_pal8_trns_52x44": (LAYOUT_RGBA8, 52, 44),
    "phash_rgb16_40x24": (LAYOUT_RGB16, 40, 24),
    "phash_rgb8_48x40": (LAYOUT_RGB8, 48, 40),
    "phash_rgba8_48x40": (LAYOUT_RGBA8, 48, 40),
    "phash_small_13x9": (LAYOUT_RGB8, 13, 9),
}


def test_cdylib_is_discoverable() -> None:
    path = find_cdylib()
    assert path.is_file(), path


@pytest.mark.parametrize("name", sorted(VECTORS))
def test_reference_vector_is_reproduced_hex_exact(name: str) -> None:
    layout, width, height = VECTORS[name]
    expected = json.loads((REPO_ROOT / "reference.json").read_text(encoding="utf-8"))
    expected_hex = expected["vectors"][name]
    dump = (REPO_ROOT / "tests" / "fixtures" / "phash" / f"{name}.raw").read_bytes()
    assert len(dump) == width * height * _channels(layout) * _sample_bytes(layout)

    digest = format(image_phash(dump, width, height, layout), "016x")
    assert digest == expected_hex, name


def _channels(layout: int) -> int:
    return {LAYOUT_GRAY8: 1, LAYOUT_GRAY16: 1, LAYOUT_RGB8: 3, LAYOUT_RGB16: 3, LAYOUT_RGBA8: 4}[layout]


def _sample_bytes(layout: int) -> int:
    return {LAYOUT_GRAY8: 1, LAYOUT_GRAY16: 2, LAYOUT_RGB8: 1, LAYOUT_RGB16: 2, LAYOUT_RGBA8: 1}[layout]


def test_invalid_arguments_are_refused_not_crashing() -> None:
    dump = (REPO_ROOT / "tests" / "fixtures" / "phash" / "base_444.raw").read_bytes()
    with pytest.raises(ValueError):
        image_phash(dump, 32, 24, 99)  # unknown layout code, refused client-side
    with pytest.raises(FfiError) as err:
        image_phash(dump[:-1], 32, 24, LAYOUT_RGB8)  # truncated dump
    assert err.value.status == -1


def test_full_dump_matches_a_rust_pinned_value() -> None:
    # The oracle value pinned twice upstream (src/reference.rs), so this
    # test fails loudly even if reference.json were regenerated wrongly.
    dump = (REPO_ROOT / "tests" / "fixtures" / "phash" / "base_444.raw").read_bytes()
    assert format(image_phash(dump, 32, 24, LAYOUT_RGB8), "016x") == "8d0a3f411ee50f7a"
