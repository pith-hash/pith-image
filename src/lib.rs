//! Raw raster buffers, BMP decoding, tier-3 local features and the
//! 64-bit pHash — the image lane of the pith hashing suite.
//!
//! pith-image merges three crates of the upstream `modhash` kit into one
//! dependency-free package, ported byte-compatibly (algorithms, public
//! API shape, tests and fixtures unchanged):
//!
//! * [`raster`] — the raw image buffer every decoder writes and every
//!   hash reads (tight row-major layout, integer box-average
//!   resampling, BT.601 luma, EXIF orientation), from `modhash-raster`.
//! * [`bmp`] — BMP decoding for 24/32-bit images and RLE8 compression,
//!   from `modhash-bmp`.
//! * [`tier3`] — FAST-9 corners, rBRIEF descriptors, RANSAC geometric
//!   verification, the D₄ symmetries and banded DTW, from
//!   `modhash-tier3`.
//! * [`phash`] — the 64-bit image perceptual hash
//!   (`luma → 32×32 box average → DCT-II → low 8×8 minus DC → median
//!   threshold`), bit-for-bit identical to the `modhash` facade and
//!   pinned by the committed oracle vectors in [`reference`].
//!
//! The only dependencies are the suite's own [`pith-digest`] and
//! [`pith-math`] crates; CI enforces the zero-third-party gate.
//!
//! [`pith-digest`]: https://github.com/pith-hash/pith-digest
//! [`pith-math`]: https://github.com/pith-hash/pith-math

// `unsafe` is denied everywhere except `ffi`, the C ABI surface the
// language SDKs bind through: raw pointers exist only at that boundary,
// and every exported function is a documented `unsafe extern "C"` fn.
#![deny(unsafe_code)]
#![deny(missing_docs)]

// The `raster` and `bmp` modules were `no_std` crates upstream and keep
// their `alloc`-only buffer code unchanged.
extern crate alloc;

pub mod bmp;
pub mod ffi;
pub mod phash;
pub mod raster;
pub mod reference;
pub mod tier3;
