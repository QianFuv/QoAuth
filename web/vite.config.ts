import { defineConfig } from "vite";

/** Build browser assets directly into the Go embedding directory. */
export default defineConfig({
  build: {
    outDir: "../internal/webassets/dist",
    emptyOutDir: true,
  },
});
