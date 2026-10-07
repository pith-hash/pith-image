// SPDX-License-Identifier: MIT
// Copyright (c) 2026 pith-hash
package hash.pith.image;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.Arrays;
import java.util.LinkedHashMap;
import java.util.Map;
import org.junit.jupiter.api.Test;

/**
 * Hex-exact conformance: the committed {@code reference.json} vectors
 * through JNI — the same vectors the Rust {@code gen-reference verify}
 * gate and the Python/Node/Go SDK suites replay, plus one Rust-derived
 * literal pin and the refusal paths (never a crash).
 */
class PithImageTest {

    /** name → {layout, width, height}, per the committed VECTORS table in src/reference.rs (RawLayout declaration order is the wire code). */
    private static final Map<String, int[]> VECTORS = new LinkedHashMap<>();

    static {
        VECTORS.put("base_444", new int[] {PithImage.LAYOUT_RGB8, 32, 24});
        VECTORS.put("phash_adam7_37x29", new int[] {PithImage.LAYOUT_RGB8, 37, 29});
        VECTORS.put("phash_gray16_36x28", new int[] {PithImage.LAYOUT_GRAY16, 36, 28});
        VECTORS.put("phash_gray8_40x32", new int[] {PithImage.LAYOUT_GRAY8, 40, 32});
        VECTORS.put("phash_pal8_trns_52x44", new int[] {PithImage.LAYOUT_RGBA8, 52, 44});
        VECTORS.put("phash_rgb16_40x24", new int[] {PithImage.LAYOUT_RGB16, 40, 24});
        VECTORS.put("phash_rgb8_48x40", new int[] {PithImage.LAYOUT_RGB8, 48, 40});
        VECTORS.put("phash_rgba8_48x40", new int[] {PithImage.LAYOUT_RGBA8, 48, 40});
        VECTORS.put("phash_small_13x9", new int[] {PithImage.LAYOUT_RGB8, 13, 9});
    }

    private static final Path REPO_ROOT = findRepoRoot();

    /**
     * The repo root: the nearest ancestor (or the working directory
     * itself) holding {@code reference.json} — works both for
     * {@code mvn test} from {@code sdk/java} and from the repo root.
     */
    private static Path findRepoRoot() {
        Path dir = Paths.get("").toAbsolutePath();
        for (int up = 0; up <= 6 && dir != null; up++) {
            if (Files.isRegularFile(dir.resolve("reference.json"))) {
                return dir;
            }
            dir = dir.getParent();
        }
        throw new IllegalStateException(
                "no reference.json found at or above " + Paths.get("").toAbsolutePath());
    }

    private static String hex(long hash) {
        return String.format("%016x", hash);
    }

    private static Path fixture(String name) {
        return REPO_ROOT.resolve("tests").resolve("fixtures").resolve("phash")
                .resolve(name + ".raw");
    }

    @Test
    void cdylib_is_discoverable() {
        assertTrue(Files.isRegularFile(Paths.get(PithImage.cdylibPath())),
                PithImage.cdylibPath());
    }

    @Test
    void every_reference_vector_is_reproduced_hex_exact() throws Exception {
        JsonNode expected = new ObjectMapper()
                .readTree(REPO_ROOT.resolve("reference.json").toFile())
                .get("vectors");
        for (Map.Entry<String, int[]> vector : VECTORS.entrySet()) {
            String name = vector.getKey();
            int layout = vector.getValue()[0];
            int width = vector.getValue()[1];
            int height = vector.getValue()[2];
            byte[] dump = Files.readAllBytes(fixture(name));
            assertEquals(width * height * channels(layout) * sampleBytes(layout),
                    dump.length, name + " dump size");
            long hash = PithImage.phash(dump, width, height, layout);
            assertEquals(expected.get(name).asText(), hex(hash), name);
        }
        assertEquals(9, VECTORS.size(), "every vector consumed");
    }

    @Test
    void full_dump_matches_a_rust_pinned_value() throws Exception {
        // Pinned twice upstream (src/reference.rs); fails loudly even
        // if reference.json were regenerated wrongly.
        byte[] dump = Files.readAllBytes(fixture("base_444"));
        assertEquals("8d0a3f411ee50f7a",
                hex(PithImage.phash(dump, 32, 24, PithImage.LAYOUT_RGB8)));
    }

    @Test
    void truncated_dump_is_refused_not_crashing() throws Exception {
        byte[] dump = Files.readAllBytes(fixture("base_444"));
        PithImage.FfiError err = assertThrows(PithImage.FfiError.class,
                () -> PithImage.phash(Arrays.copyOf(dump, dump.length - 1), 32, 24,
                        PithImage.LAYOUT_RGB8));
        assertEquals(PithImage.PITH_E_INVALID, err.status);
        assertEquals("pith_image_phash", err.op);
    }

    @Test
    void null_dump_is_refused_not_crashing() {
        PithImage.FfiError err = assertThrows(PithImage.FfiError.class,
                () -> PithImage.phash(null, 32, 24, PithImage.LAYOUT_RGB8));
        assertEquals(PithImage.PITH_E_INVALID, err.status);
    }

    @Test
    void unknown_layout_is_refused_client_side() throws Exception {
        byte[] dump = Files.readAllBytes(fixture("base_444"));
        assertThrows(IllegalArgumentException.class,
                () -> PithImage.phash(dump, 32, 24, 99));
    }

    private static int channels(int layout) {
        switch (layout) {
            case PithImage.LAYOUT_GRAY8:
            case PithImage.LAYOUT_GRAY16:
                return 1;
            case PithImage.LAYOUT_RGB8:
            case PithImage.LAYOUT_RGB16:
                return 3;
            case PithImage.LAYOUT_RGBA8:
                return 4;
            default:
                throw new IllegalArgumentException("layout " + layout);
        }
    }

    private static int sampleBytes(int layout) {
        return layout == PithImage.LAYOUT_GRAY16 || layout == PithImage.LAYOUT_RGB16 ? 2 : 1;
    }
}
