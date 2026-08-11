// Flat ESLint config shared by every package in the workspace.
//
// One config at the root rather than per-package: the packages are the same
// stack (TypeScript, some React) and drift between per-package configs is how
// a rule ends up silently disabled in the one place it mattered.
//
// Type-aware linting is deliberately not enabled: it needs a per-file program
// and is markedly slower, and `pnpm typecheck` already runs tsc across all 8
// packages. The cost is that type-only rules are unavailable — notably
// no-floating-promises, which would be worth having in a codebase where an
// unawaited refresh means work that silently did not happen. Worth revisiting
// if lint moves off the hot path.

import js from "@eslint/js";
import reactHooks from "eslint-plugin-react-hooks";
import tseslint from "typescript-eslint";

export default tseslint.config(
  {
    // Build output, deps, and generated clients. api-client.ts and
    // api-types.ts are emitted by sdkgen from the OpenAPI spec — linting them
    // would report on a generator's output, which no one can act on here.
    ignores: [
      "**/dist/**",
      "**/node_modules/**",
      "**/.next/**",
      "**/storybook-static/**",
      "**/src/generated/**",
    ],
  },

  js.configs.recommended,
  ...tseslint.configs.recommended,

  {
    files: ["**/*.{ts,tsx}"],
    rules: {
      // Unused values are usually a leftover or a bug, but an intentionally
      // ignored binding is legitimate — allow the conventional _ prefix so it
      // has to be stated rather than assumed.
      "@typescript-eslint/no-unused-vars": [
        "error",
        {
          argsIgnorePattern: "^_",
          varsIgnorePattern: "^_",
          caughtErrorsIgnorePattern: "^_",
        },
      ],

      // `any` erases the checking that makes the rest of this worthwhile, but
      // it is sometimes the honest type at a boundary — warn so it shows up in
      // review without blocking a build.
      "@typescript-eslint/no-explicit-any": "warn",

      // Stray logging in an auth SDK risks printing tokens into a console the
      // page's other scripts can read; warn/error are kept for real faults.
      "no-console": ["warn", { allow: ["warn", "error"] }],
    },
  },

  {
    // React packages. The hooks rules are the point: exhaustive-deps catches
    // stale closures over auth state, and rules-of-hooks catches conditional
    // hook calls that only fail at runtime.
    files: ["packages/{react,components,nextjs}/**/*.{ts,tsx}"],
    plugins: { "react-hooks": reactHooks },
    rules: {
      ...reactHooks.configs.recommended.rules,
    },
  },

  {
    // Tests assert on shapes that are deliberately wrong, and reach into
    // internals to stub them. Relaxing here keeps the signal in src high.
    files: ["**/*.test.{ts,tsx}"],
    rules: {
      "@typescript-eslint/no-explicit-any": "off",
      "@typescript-eslint/no-unsafe-function-type": "off",
    },
  },
);
