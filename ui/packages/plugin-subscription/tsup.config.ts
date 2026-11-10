import { defineConfig } from "tsup";

// Three entries so an app pays only for what it imports: the framework-free
// plugin, the React hooks, and the styled components built on
// @authsome/ui-components.
export default defineConfig({
  entry: ["src/index.ts", "src/react.tsx", "src/components.tsx"],
  format: ["esm"],
  dts: true,
  clean: true,
  sourcemap: true,
  target: "es2022",
  external: [
    "react",
    "react-dom",
    "@authsome/ui-core",
    "@authsome/ui-react",
    "@authsome/ui-components",
  ],
  banner: {
    js: '"use client";',
  },
});
