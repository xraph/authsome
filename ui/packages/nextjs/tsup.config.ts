import { defineConfig } from "tsup";

export default defineConfig({
  entry: ["src/index.ts", "src/middleware.ts", "src/pages/index.tsx"],
  format: ["esm"],
  dts: true,
  clean: true,
  sourcemap: true,
  target: "es2022",
  external: ["react", "react-dom", "next", "@authsome/ui-components", "@authsome/ui-react", "@authsome/ui-core"],
});
