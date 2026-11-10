import { defineConfig } from "vitest/config";

// Same shape as ui-react's config. The hook test renders, so it needs a DOM,
// and globals is on so @testing-library/react can register its cleanup.
export default defineConfig({
  test: {
    environment: "jsdom",
    globals: true,
    include: ["src/**/*.test.{ts,tsx}"],
  },
});
