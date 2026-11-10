import { defineConfig } from "tsup";

export default defineConfig({
  // Two entries so a host that never touches React does not pull it in just
  // by importing the plugin.
  entry: ["src/index.ts", "src/react.ts"],
  format: ["esm"],
  dts: true,
  clean: true,
  sourcemap: true,
  target: "es2022",
  external: [
    "react",
    "react-dom",
    "@authsome/ui-react",
    "@authsome/ui-components",
    "lucide-react",
  ],
});
