import { defineConfig } from "vitest/config";

// ui-core and ui-nextjs get by on vitest's defaults because they test plain
// modules. This package renders components, so it needs a DOM.
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
