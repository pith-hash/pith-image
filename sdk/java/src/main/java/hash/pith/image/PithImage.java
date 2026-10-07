// SPDX-License-Identifier: MIT
// Copyright (c) 2026 pith-hash
package hash.pith.image;

import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;

/**
 * Java JNI bindings for the {@code pith-image} cdylib: the 64-bit
 * perceptual hash of a raw pixel dump — the same C library the Python
 * (ctypes), Node (koffi) and Go (cgo) SDKs bind through.
 *
 * <p>The cdylib is resolved once at class-load time, mirroring the
 * discovery chain of the other SDKs: (1) the {@code PITH_CDYLIB}
 * environment variable — the explicit file; (2) {@code PITH_CDYLIB_DIR}
 * — a directory holding one of the platform library names; (3) a
 * {@code target/release} directory at the working directory or up to
 * six ancestors above it. {@link LinkageError} names every candidate
 * when nothing matches.</p>
 *
 * <p>Hashes are unsigned 64-bit values carried in a Java {@code long}
 * (the two's-complement bit pattern — format with {@code %016x} for
 * the hex forms the {@code reference.json} vectors pin). Refusals
 * raise {@link FfiError}, carrying the C ABI status code.</p>
 */
public final class PithImage {

    /** Status: success. */
    public static final int PITH_OK = 0;

    /** Status: invalid argument — a null array, an unknown layout code, or a dump whose length disagrees with the geometry. */
    public static final int PITH_E_INVALID = -1;

    /** Status: the core pipeline refused the input. */
    public static final int PITH_E_REJECTED = -2;

    /** Layout code: one {@code u8} luma sample per pixel. */
    public static final int LAYOUT_GRAY8 = 0;

    /** Layout code: one {@code u16} luma sample per pixel, little-endian. */
    public static final int LAYOUT_GRAY16 = 1;

    /** Layout code: three {@code u8} samples per pixel, {@code R G B}. */
    public static final int LAYOUT_RGB8 = 2;

    /** Layout code: three {@code u16} samples per pixel, {@code R G B}, little-endian. */
    public static final int LAYOUT_RGB16 = 3;

    /** Layout code: four {@code u8} samples per pixel, {@code R G B A} (alpha ignored by the luma step). */
    public static final int LAYOUT_RGBA8 = 4;

    /** Platform cdylib file names, in probe order. */
    private static final String[] CDYLIB_NAMES = {
        "pith_image.dll", "libpith_image.so", "libpith_image.dylib",
    };

    private static final String CDYLIB_PATH = findCdylib();

    static {
        System.load(CDYLIB_PATH);
    }

    private PithImage() {
    }

    /** The absolute path of the loaded cdylib (tests and diagnostics). */
    public static String cdylibPath() {
        return CDYLIB_PATH;
    }

    private static native long phashNative(
            byte[] data, int width, int height, int layout, int[] status);

    /**
     * Computes the 64-bit perceptual hash of a raw pixel dump.
     *
     * <p>{@code data} is the flat, row-major, channel-interleaved dump
     * of a {@code width}×{@code height} image — little-endian for the
     * 16-bit layouts, exactly the format the fixtures under
     * {@code tests/fixtures/phash/} commit; {@code layout} is one of
     * the {@code LAYOUT_*} codes above.</p>
     *
     * @throws IllegalArgumentException for an unknown layout code
     *     (refused client-side, mirroring the Python SDK)
     * @throws FfiError for a null dump or a length/geometry mismatch
     *     (status {@code PITH_E_INVALID})
     */
    public static long phash(byte[] data, int width, int height, int layout) {
        if (layout < LAYOUT_GRAY8 || layout > LAYOUT_RGBA8) {
            throw new IllegalArgumentException("unknown layout code: " + layout);
        }
        int[] status = new int[1];
        long hash = phashNative(data, width, height, layout, status);
        if (status[0] != PITH_OK) {
            throw new FfiError("pith_image_phash", status[0]);
        }
        return hash;
    }

    /**
     * A native call refused or failed: the C ABI status code plus the
     * operation that reported it — the Java face of the ctypes/koffi/
     * cgo {@code FfiError}.
     */
    public static final class FfiError extends RuntimeException {
        private static final long serialVersionUID = 1L;

        /** The refusing operation (its C ABI name). */
        public final String op;

        /** The C ABI status code ({@code -1} invalid, {@code -2} rejected). */
        public final int status;

        FfiError(String op, int status) {
            super(op + " failed with status " + status);
            this.op = op;
            this.status = status;
        }
    }

    /**
     * Resolves the cdylib path: {@code PITH_CDYLIB} (explicit file),
     * then {@code PITH_CDYLIB_DIR} + platform name, then a
     * {@code target/release} directory at the working directory or up
     * to six ancestors above it.
     */
    private static String findCdylib() {
        Path cwd = Paths.get("").toAbsolutePath();

        String explicitFile = System.getenv("PITH_CDYLIB");
        if (explicitFile != null && !explicitFile.isEmpty()) {
            Path file = Paths.get(explicitFile).toAbsolutePath();
            if (Files.isRegularFile(file)) {
                return file.toString();
            }
        }

        String dir = System.getenv("PITH_CDYLIB_DIR");
        if (dir != null && !dir.isEmpty()) {
            Path base = Paths.get(dir).toAbsolutePath();
            for (String name : CDYLIB_NAMES) {
                Path candidate = base.resolve(name);
                if (Files.isRegularFile(candidate)) {
                    return candidate.toString();
                }
            }
        }

        for (Path base = cwd; base != null; base = base.getParent()) {
            for (String name : CDYLIB_NAMES) {
                Path candidate = base.resolve("target").resolve("release").resolve(name);
                if (Files.isRegularFile(candidate)) {
                    return candidate.toString();
                }
            }
        }

        throw new LinkageError(
                "cannot locate the pith_image cdylib; set PITH_CDYLIB or PITH_CDYLIB_DIR"
                + " (probed PITH_CDYLIB, PITH_CDYLIB_DIR, and target/release at "
                + cwd + " and its ancestors)");
    }
}
