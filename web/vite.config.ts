import { defineConfig } from "vite";

/** Build browser assets directly into the Go embedding directory. */
export default defineConfig({
  build: {
    license: { fileName: ".vite/license.json" },
    outDir: "../internal/webassets/dist",
    emptyOutDir: true,
    assetsInlineLimit: (filePath: string) => (filePath.endsWith(".woff2") ? false : undefined),
  },
});
