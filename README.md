<p align="center">
  <img src="https://pith-image.n24q02m.com/logo.svg" alt="pith-image" width="120">
</p>

<h1 align="center">pith-image</h1>

<p align="center">
  <strong>pith image lane: raster buffers, BMP decode, tier-3 features, pHash (zero-dep Rust)</strong>
</p>

<p align="center">
  <a href="https://github.com/pith-hash/pith-image/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/pith-hash/pith-image/actions/workflows/ci.yml/badge.svg"></a>
  <a href="https://github.com/pith-hash/pith-image/actions/workflows/cd.yml"><img alt="CD" src="https://github.com/pith-hash/pith-image/actions/workflows/cd.yml/badge.svg"></a>
  <a href="https://github.com/pith-hash/pith-image/releases/latest"><img alt="Latest release" src="https://img.shields.io/github/v/release/pith-hash/pith-image?display_name=tag&sort=semver"></a>
  <a href="https://github.com/n24q02m/better-semantic-release"><img alt="semantic-release" src="https://img.shields.io/badge/semantic--release-e10079?logo=semantic-release&logoColor=white"></a>
  <a href="LICENSE"><img alt="License: Apache-2.0" src="https://img.shields.io/github/license/pith-hash/pith-image"></a>
</p>

<p align="center">
  <a href="#install">Install</a> ·
  <a href="#quick-start">Quick start</a> ·
  <a href="#the-pith-suite-contract">Suite contract</a>
</p>

<!-- BEGIN: AUTO-GENERATED-CROSS-PROMO -->
<!-- END: AUTO-GENERATED-CROSS-PROMO -->

## The pith suite contract

pith-image is part of the **pith** suite (pith-hash). Every suite repository
follows the same rules; CI enforces them mechanically:

- **Naming**: a library is always `pith-<domain>` (`pith-image`, `pith-audio`,
  `pith-zip`, ...). The curator/repository of repositories is the bare
  `pith-hash`. Never invent a second naming scheme inside the suite.
- **Version pinning**: cross-library dependencies pin `~0.1` (e.g.
  `pith-image = { version = "~0.1", path = "../pith-image" }`). The whole suite
  moves together inside 0.1.x; breaking changes require a suite-wide version
  bump, never a silent minor drift.
- **Zero third-party dependencies**: every crate depends only on other
  `pith-*` crates plus `std`. `scripts/check-zero-deps.py` (run in CI) fails
  the build on any other crate, for normal, build and dev dependencies alike.
- **No unsafe**: every crate root carries `#![forbid(unsafe_code)]`.
- **Hex-exact vectors**: `reference.json` at the repo root is the
  cross-language source of truth. The `gen-reference` binary regenerates it;
  CI verifies the committed copy is current (`gen-reference verify`), and CD
  ships the regenerated file with every SDK artifact. Python, Node and Go SDKs
  MUST test against the same bytes.

## Repository layout

```
crates/            one published crate per suite lib (pith-<domain>)
tools/gen-reference  the vector generator binary (bin name: gen-reference)
sdk/python         ctypes wheel; build backend reads PITH_CDYLIB_DIR
sdk/node           koffi-based package; prebuilds/<os-arch>/ carry the cdylib
sdk/go             cgo binding; go.mod carries the module's cgo flags
fuzz/corpus        fuzz inputs, replayed by tests/fuzz_corpus.rs (parser crates)
reference.json     hex-exact cross-SDK test vectors
```

## Install

Rust (the core library):

```bash
cargo add pith-image
```

Python / Node / Go SDKs are published from the same cdylib on every release;
see the release assets or the package registries for the matching version.

## Quick start

Rust (the core library):

```bash
cargo add pith-image
```

The 64-bit perceptual hash of a raw RGB buffer:

```rust
use pith_image::phash::image_phash;
use pith_image::raster::{Image, Rgb};

// 2x1 red/green test image: tight row-major RGB, no padding.
let img = Image::<Rgb, u8>::from_vec(2, 1, vec![255, 0, 0, 0, 255, 0])
    .expect("2x1 matches its buffer length");

// luma -> 32x32 box average -> DCT-II -> low 8x8 minus DC -> median threshold.
let bits = image_phash(&img).expect("well-formed images cannot fail");
println!("phash: {bits:016x}");
```

Any layout works: `Gray` pixels are already luma, `Rgb`/`Rgba` convert
through BT.601 with alpha ignored, `u16` samples reduce to `u8` first.
Hex-exact cross-SDK vectors live in `reference.json` (regenerate with
`cargo run --bin gen-reference`, verify with
`cargo run --bin gen-reference -- verify`).

Python / Node / Go SDKs are published from the same cdylib on every release;
see the release assets or the package registries for the matching version.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## Security

See [SECURITY.md](SECURITY.md).

## License

[Apache-2.0](LICENSE) © pith-hash
