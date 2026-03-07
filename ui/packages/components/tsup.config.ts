import { defineConfig } from "tsup";
import { copyFileSync } from "fs";

export default defineConfig({
  entry: ["src/index.ts"],
  format: ["esm"],
  dts: true,
  clean: true,
  sourcemap: true,
  target: "es2022",
  external: ["react", "react-dom"],
  banner: {
    js: '"use client";',
  },
  onSuccess: async () => {
    copyFileSync("src/styles/globals.css", "dist/styles.css");
  },
});
