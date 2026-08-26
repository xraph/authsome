import { defineConfig } from "vitest/config";

// Same shape as ui-react's config. These components render, so they need a
// DOM, and several of them read window.location or listen for popstate.
//
// globals is on so @testing-library/react can register its own afterEach
// cleanup. Without it every render stays in the document and the next test
// finds several copies of the same element. Tests still import describe/it/
// expect explicitly, the same way the other packages do.
export default defineConfig({
  test: {
    environment: "jsdom",
    globals: true,
    include: ["src/**/*.test.{ts,tsx}"],
  },
});
