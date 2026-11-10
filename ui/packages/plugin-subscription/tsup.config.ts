import { defineConfig, type Options } from "tsup";

// Three entries so an app pays only for what it imports: the framework-free
// plugin, the React hooks, and the styled components built on
// @authsome/ui-components. Only the React entries are marked "use client":
// the plugin and its models stay importable from server components. The two
// builds run side by side, so neither cleans dist; the build script does.
const shared: Options = {
  format: ["esm"],
  dts: true,
  clean: false,
  sourcemap: true,
  target: "es2022",
  external: [
    "react",
    "react-dom",
    "@authsome/ui-core",
    "@authsome/ui-react",
    "@authsome/ui-components",
  ],
};

export default defineConfig([
  { ...shared, entry: ["src/index.ts"] },
  {
    ...shared,
    entry: ["src/react.tsx", "src/components.tsx"],
    banner: { js: '"use client";' },
  },
]);
