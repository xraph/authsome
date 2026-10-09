# Dependency qualification, 2026-10-09

The affected dependency graphs are corrected. You can reproduce the local gates from the workspace commands below. Deployment state and the separate UI authentication contract review remain open.

| Artifact | Current graph and boundary | Verification |
| --- | --- | --- |
| Docs server | Next 16.3.8, sharp 0.35.5, source-map-js 1.2.2. Fumadocs core/UI 16.15.1, MDX 15.3.1 and direct PostCSS ^8.5.26 preserved. | Frozen pnpm 10.31.0 install, format, lint, typecheck, build and audit on Node 22.23.3. Production HTTP checks covered home, docs, search, full LLM text, both Markdown routes and the OG image. Desktop and narrow pages inspected. |
| TypeScript SDK developer tools | Vitest and @vitest/mocker 4.1.11, Vite 8.3.4, source-map-js 1.2.2. Tinypool removed. Exact esbuild 0.28.2 is a dev dependency. | Clean npm installs, typecheck, build, eight tests and audit on Node 20.20.2 and 22.23.3. Generator template and generated output agree. |
| Published SDK | No runtime dependencies or added consumer engine restriction. Existing main, module and types entry names preserved. | A fresh real pack loaded through CommonJS require and the declared ES module entry on both Node versions. Five named exports and four local fixture health requests per runtime passed. Bundles contain no external test/build tooling or Node builtin imports. |
| Storybook developer app | Direct Vite 6.4.3 matches the existing override and lock. Direct lucide-react ^1.34.0 matches UI components. | Frozen pnpm 9.15.0 install, scoped lint/typecheck, static build, dev and static HTTP checks. Sign-in, MFA and email verification exercised through current mock AuthState variants. Desktop 1440 px and narrow 390 px layouts inspected without horizontal overflow. |
| Published UI libraries | Existing package contracts and shipped UI source preserved. Vite/Storybook are developer tools in this workspace. | All five actual packs inspected: declared main entries exist, workspace dependencies become published versions, and no external build/test tooling imports or runtime dependencies appear. UI build/typecheck and 198 tests passed. |

The original inventory contained 25 records, representing 17 package/advisory pairs: 17 docs records, five SDK developer graph records and three Storybook records. All affected versions are corrected in the checked source graphs. The inventory covers Next draft/cache handling, development MCP, metadata image routes, image optimization and OG rendering; sharp native decoders; source-map-js parsing; Tinypool worker options; Vitest mock redirects; and Vite filesystem, source-map and editor handling. Developer exposure is assessed separately from published bundles. No alert was dismissed or suppressed, and lack of a reproduced exploit was not used to declare an artifact safe.

The packaging check first reproduced a missing advertised dist/index.mjs and extensionless-import failure from the old dist/index.js. The focused repair emits declarations with tsc, then browser-compatible ES2020 CommonJS and ES module bundles with esbuild. It adds an explicit build dependency and bundled artifacts while keeping the existing public entry names and source API.

Storybook now targets chrome87, edge88, es2020, firefox78 and safari14.1 for build and dependency optimization. Its prior Safari14 target cannot compile with the corrected esbuild behavior because of the [older Safari destructuring bug](https://github.com/evanw/esbuild/issues/4436). This developer artifact floor changes from Safari14 to 14.1; SDK and published UI browser contracts do not change. The browser checks ran in Chromium, so they do not establish Safari execution support.

The docs tokenizer restoration helper uses explicit NUL/SOH delimiters. Equivalence checks passed 10,017 bounded placeholder cases and eight markup/highlighting cases against the prior implementation. Existing docs lint findings were repaired without changing copy or adding dummy controls. Storybook callbacks use existing observable action spies.

Commands:

```sh
make f
make l
cd docs
corepack pnpm@10.31.0 install --frozen-lockfile
corepack pnpm@10.31.0 format
corepack pnpm@10.31.0 types:check
corepack pnpm@10.31.0 lint
corepack pnpm@10.31.0 build
corepack pnpm@10.31.0 audit --json
cd ../sdk/typescript
npm ci
npm run typecheck
npm run build
npm test
npm audit --json
npm pack --json
cd ../../ui
corepack pnpm@9.15.0 install --frozen-lockfile
corepack pnpm@9.15.0 exec eslint apps/storybook
corepack pnpm@9.15.0 --filter @authsome/storybook exec tsc --noEmit
corepack pnpm@9.15.0 --filter @authsome/storybook build-storybook
corepack pnpm@9.15.0 build
corepack pnpm@9.15.0 typecheck
corepack pnpm@9.15.0 lint
corepack pnpm@9.15.0 test
corepack pnpm@9.15.0 audit --json
cd ../sdkgen
GOTOOLCHAIN=go1.26.9 GOWORK=off go test ./typescript
```

The SDK developer graph requires Node 20.19+ or 22.12+; these gates are not new consumer requirements. Use the selected Node runtime for subprocesses too. The UI Turbo checks used a task-local pnpm 9.15.0 shim to preserve the workspace package manager in spawned processes.

Zero audit findings are one gate. No deployed docs or Storybook host was verified, Windows-specific advisory behavior was not exercised, and no new Go/provider functional changes or composed Go security analysis belong to this dependency task. Docs has no active CI coverage, so its local gates are required. Seven existing shipped UI core no-explicit-any warnings expose MFA contract drift and are assigned to a separate client-contract correction. They remain unresolved. Storybook mock success does not establish live authentication or security qualification. Final remote CI status must be read from the exact commit run; a cancelled predecessor is not complete suite evidence.
