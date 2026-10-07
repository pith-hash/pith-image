// SPDX-License-Identifier: MIT
// Copyright (c) 2026 pith-hash
"use strict";

/**
 * pith-image SDK: the 64-bit image perceptual hash through koffi.
 *
 * The single Rust core (the `pith-image` cdylib built by
 * `cargo build --release`) is loaded at runtime; koffi is the only
 * runtime dependency.
 *
 * Discovery order (the suite's cdylib convention):
 *
 *  1. `PITH_CDYLIB` — an explicit cdylib *file* path;
 *  2. `PITH_CDYLIB_DIR` — a *directory* scanned for the cdylib names
 *     (the CD pipeline points this at `target/release`);
 *  3. `prebuilds/` — the packaged npm layout the CD publish job
 *     assembles, flat and per `<os-arch>` (e.g. `linux-x64`);
 *  4. `<repo root>/target/release` — the repository working-tree
 *     layout, so a source checkout runs against a local cargo build
 *     with no configuration.
 *
 * The vectors in `reference.json` at the repository root are the
 * hex-exact cross-language source of truth; the conformance tests
 * under `test/` replay all of them through this binding.
 */

const koffi = require("koffi");
const fs = require("node:fs");
const path = require("node:path");

const STATUS_OK = 0;
const STATUS_INVALID = -1;
const STATUS_REJECTED = -2;

/** Raw dump layouts, in RawLayout declaration order. */
const LAYOUT = Object.freeze({
  GRAY8: 0,
  GRAY16: 1,
  RGB8: 2,
  RGB16: 3,
  RGBA8: 4,
});

/** Every cdylib file name cargo may drop into the build directory, per platform. */
const CDYLIB_NAMES = ["pith_image.dll", "libpith_image.so", "libpith_image.dylib"];

const PKG_ROOT = path.join(__dirname);
const REPO_ROOT = path.resolve(__dirname, "..", "..");

/** FfiError: a non-zero status code came back from the cdylib. */
class FfiError extends Error {
  /**
   * @param {string} op the FFI operation name
   * @param {number} status the raw status code
   */
  constructor(op, status) {
    const kind = { [STATUS_INVALID]: "invalid argument", [STATUS_REJECTED]: "input rejected" }[status] ?? "unknown failure";
    super(`${op} failed: ${kind} (status ${status})`);
    this.name = "FfiError";
    /** The raw status code the FFI returned. */
    this.status = status;
  }
}

/**
 * Locates the cdylib through the suite's discovery chain.
 * @returns {string} an absolute path to the cdylib file
 * @throws {Error} when nothing is found
 */
function findCdylib() {
  const explicit = process.env.PITH_CDYLIB;
  if (explicit && fs.statSync(explicit, { throwIfNoEntry: false })?.isFile()) {
    return path.resolve(explicit);
  }
  /** @type {string[]} */
  const dirs = [];
  const envDir = process.env.PITH_CDYLIB_DIR;
  if (envDir) {
    dirs.push(envDir);
    if (!path.isAbsolute(envDir)) {
      dirs.push(path.join(REPO_ROOT, envDir));
    }
  }
  const osArch = `${process.platform}-${process.arch}`;
  dirs.push(path.join(PKG_ROOT, "prebuilds", osArch));
  dirs.push(path.join(PKG_ROOT, "prebuilds"));
  dirs.push(path.join(REPO_ROOT, "target", "release"));
  for (const dir of dirs) {
    for (const name of CDYLIB_NAMES) {
      const p = path.join(dir, name);
      if (fs.statSync(p, { throwIfNoEntry: false })?.isFile()) return p;
    }
  }
  throw new Error(
    "no pith-image cdylib found (searched PITH_CDYLIB, PITH_CDYLIB_DIR, prebuilds/ and <repo>/target/release); " +
      "run `cargo build --release` first",
  );
}

let cached = undefined;

/**
 * Loads the cdylib and binds the exported symbols (lazily, once).
 * @returns {{phash: (dump: Buffer, width: number, height: number, layout: number, out: {value?: bigint|number}) => number}}
 */
function loadLibrary() {
  if (cached) return cached;
  const lib = koffi.load(findCdylib());
  const phash = lib.func("pith_image_phash", "int32_t", [
    "const uint8_t *",
    "size_t",
    "uint32_t",
    "uint32_t",
    "uint32_t",
    koffi.out(koffi.pointer("uint64_t")),
  ]);
  cached = { phash };
  return cached;
}

/**
 * Computes the 64-bit perceptual hash of a raw pixel dump.
 *
 * `dump` is a flat, row-major, channel-interleaved buffer of
 * `width * height * channels` samples (little-endian for the 16-bit
 * layouts) — the format the fixtures under `tests/fixtures/phash/`
 * commit. Returns the hash; format it with
 * `hash.toString(16).padStart(16, "0")` to compare against
 * `reference.json`.
 *
 * @param {Buffer} dump the raw pixel dump
 * @param {number} width image width in pixels
 * @param {number} height image height in pixels
 * @param {number} layout one of the LAYOUT codes
 * @returns {bigint} the 64-bit hash
 */
function imagePhash(dump, width, height, layout) {
  if (!Number.isInteger(width) || width <= 0 || !Number.isInteger(height) || height <= 0) {
    throw new TypeError("width and height must be positive integers");
  }
  if (!Object.values(LAYOUT).includes(layout)) {
    throw new TypeError(`unknown layout code ${layout}`);
  }
  if (!Buffer.isBuffer(dump)) {
    throw new TypeError("dump must be a Buffer");
  }
  const { phash } = loadLibrary();
  const out = [null];
  const status = phash(dump, dump.length, width, height, layout, out);
  if (status !== STATUS_OK) {
    throw new FfiError("pith_image_phash", status);
  }
  return BigInt(out[0]);
}

module.exports = {
  STATUS_OK,
  STATUS_INVALID,
  STATUS_REJECTED,
  LAYOUT,
  CDYLIB_NAMES,
  FfiError,
  findCdylib,
  imagePhash,
};
