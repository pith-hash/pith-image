// SPDX-License-Identifier: MIT
// Copyright (c) 2026 pith-hash
"use strict";

// Hex-exact conformance: the committed reference vectors through koffi.
// Every vector in the repository-root reference.json is replayed through
// the cdylib and compared hex-exact, the same vectors the Rust
// gen-reference verify gate and the Python/Go SDKs check.

const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");

const { LAYOUT, FfiError, findCdylib, imagePhash } = require("../index.js");

const REPO_ROOT = path.resolve(__dirname, "..", "..", "..");

// name -> [layout, width, height], per the committed VECTORS table in
// src/reference.rs (RawLayout declaration order is the wire code).
const VECTORS = {
  base_444: [LAYOUT.RGB8, 32, 24],
  phash_adam7_37x29: [LAYOUT.RGB8, 37, 29],
  phash_gray16_36x28: [LAYOUT.GRAY16, 36, 28],
  phash_gray8_40x32: [LAYOUT.GRAY8, 40, 32],
  phash_pal8_trns_52x44: [LAYOUT.RGBA8, 52, 44],
  phash_rgb16_40x24: [LAYOUT.RGB16, 40, 24],
  phash_rgb8_48x40: [LAYOUT.RGB8, 48, 40],
  phash_rgba8_48x40: [LAYOUT.RGBA8, 48, 40],
  phash_small_13x9: [LAYOUT.RGB8, 13, 9],
};

function channels(layout) {
  return { [LAYOUT.GRAY8]: 1, [LAYOUT.GRAY16]: 1, [LAYOUT.RGB8]: 3, [LAYOUT.RGB16]: 3, [LAYOUT.RGBA8]: 4 }[layout];
}

function sampleBytes(layout) {
  return { [LAYOUT.GRAY8]: 1, [LAYOUT.GRAY16]: 2, [LAYOUT.RGB8]: 1, [LAYOUT.RGB16]: 2, [LAYOUT.RGBA8]: 1 }[layout];
}

test("cdylib is discoverable", () => {
  assert.ok(fs.statSync(findCdylib()).isFile());
});

for (const [name, [layout, width, height]] of Object.entries(VECTORS)) {
  test(`reference vector ${name} is reproduced hex-exact`, () => {
    const expected = JSON.parse(fs.readFileSync(path.join(REPO_ROOT, "reference.json"), "utf8")).vectors[name];
    const dump = fs.readFileSync(path.join(REPO_ROOT, "tests", "fixtures", "phash", `${name}.raw`));
    assert.equal(dump.length, width * height * channels(layout) * sampleBytes(layout));

    const digest = imagePhash(dump, width, height, layout).toString(16).padStart(16, "0");
    assert.equal(digest, expected, name);
  });
}

test("invalid arguments are refused, not crashing", () => {
  const dump = fs.readFileSync(path.join(REPO_ROOT, "tests", "fixtures", "phash", "base_444.raw"));
  assert.throws(() => imagePhash(dump, 32, 24, 99), TypeError); // unknown layout, refused client-side
  assert.throws(() => imagePhash(dump.subarray(0, dump.length - 1), 32, 24, LAYOUT.RGB8), (err) => {
    assert.ok(err instanceof FfiError);
    assert.equal(err.status, -1); // truncated dump
    return true;
  });
});

test("full dump matches a rust-pinned value", () => {
  // The oracle value pinned twice upstream (src/reference.rs), so this
  // test fails loudly even if reference.json were regenerated wrongly.
  const dump = fs.readFileSync(path.join(REPO_ROOT, "tests", "fixtures", "phash", "base_444.raw"));
  assert.equal(imagePhash(dump, 32, 24, LAYOUT.RGB8).toString(16).padStart(16, "0"), "8d0a3f411ee50f7a");
});
