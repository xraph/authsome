# Working in authsome

Authsome is a plugin-driven authentication engine for Go that runs standalone or as a Forge extension, and its module path is `github.com/xraph/authsome`. Next to the engine you'll find the generated client SDKs in `sdk/` (Go, TypeScript published as `@authsome/client`, and Dart), five npm UI packages in `ui/packages/` (`@authsome/ui-core`, `ui-react`, `ui-vue`, `ui-components`, `ui-nextjs`) and three Flutter packages in `flutter/packages/` (`authsome_core`, `authsome_flutter`, `authsome_flutter_ui`).

## Build, test and lint

### Go

You run everything from the repo root. Go reads the version and toolchain from `go.mod`.

| You want | Run |
|---|---|
| Unit tests, the same command `ci / Test` runs | `make test` (`go test -v ./...`) |
| The race detector | `make test-race` |
| Postgres and Mongo store conformance (needs Docker) | `make test-integration` |
| Formatting | `make fmt` (`gofmt -s`, then `goimports -local github.com/xraph/authsome`) |
| Lint | `make lint`, or `make lint-fix` to apply fixes |
| Vet | `make vet` |
| Reachable vulnerabilities | `make vuln` (govulncheck) |
| fmt, vet, lint and vuln together | `make check` |
| Coverage | `make coverage`, `make coverage-html` |
| goimports and golangci-lint | `make deps`, then `make check-deps` |

golangci-lint caps its output by default. When you're cleaning up a package, run `golangci-lint run --max-issues-per-linter=0 --max-same-issues=0 ./...` so you see every finding.

CI also runs the shared `xraph/workflows` go-ci (Test, Lint, Verify, Security). Verify runs `go mod tidy` and fails if `go.mod` or `go.sum` changes. Tidy before you push.

### What the store tests need

When you run `go test ./...` you get the unit tests plus the store conformance suite against the memory and SQLite backends. SQLite is embedded, so none of that needs Docker.

Postgres and Mongo tests carry the `integration` build tag. Without `-tags integration` they are not even compiled, so a green `go test ./...` tells you nothing about those two backends. With the tag:

- Postgres tests start `postgres:16-alpine` through testcontainers. You need a running Docker daemon, and without one they fail.
- Mongo tests read `AUTHSOME_MONGO_URI` and call `t.Skip` when it is unset. Without `-v` a package of skipped tests still prints `ok`, so if you forget the variable the run passes silently and you have learned nothing about Mongo. Point it at a replica set, because the app and org cascades run in transactions.

`make test-integration` does all of this for you. It starts a single-node `mongo:7` replica set on port 27019 (change it with `MONGO_PORT`), points `AUTHSOME_MONGO_URI` at it, runs the command CI's "Store Conformance (pg + mongo)" job runs, then removes the container:

```sh
go test -tags integration -p 1 -count=1 -timeout 15m \
  -run '^TestConformance$|^TestStoreConformance_|^TestMigration_' ./store/... ./plugins/...
```

Keep `-p 1`. The packages share one Mongo database and at least one of them drops it in teardown. The `-run` filter matters just as much, because CI only runs integration tests named `TestConformance`, `TestStoreConformance_*` or `TestMigration_*`, and anything else you put behind the tag is compiled by nothing and run by nothing, so name a new one to fit. Several of these tests write fixed values and never clean up, which is why each run wants a fresh database. CI sets `TESTCONTAINERS_RYUK_DISABLED=true`; a local run keeps the reaper.

The `api` test harness (`newTestAPI` in `api/api_test.go`) does not run the identity middleware, since the Forge extension is what adds it. You put the caller on the request context yourself with `middleware.WithUserID` and `middleware.WithUser`. If you're testing a permission-gated route, use `newBootstrappedAPI` in `api/unauth_admin_endpoints_test.go`, which builds the engine with `authsome.WithBootstrap()`.

### UI packages and the TypeScript SDK

```sh
cd ui
pnpm install --frozen-lockfile
pnpm run build
pnpm run typecheck
pnpm run lint
pnpm run test
pnpm audit --audit-level=high

cd ../sdk/typescript
npm install
npm run build
npm test
```

`ui/` is a pnpm workspace driven by turbo. Use the pnpm version pinned in the `packageManager` field of `ui/package.json`, and Node 20 if you want to match CI. `ui/apps/storybook` lives in the workspace and is not published. `sdk/typescript` sits outside the workspace and uses npm.

### Flutter and Dart

```sh
cd flutter
flutter pub get
cp example/.env.example example/.env
flutter analyze --no-fatal-infos

cd ../sdk/dart
dart pub get
dart test
```

`flutter/pubspec.yaml` is a pub workspace over the three packages and `example/`. The example bundles `.env` as an asset and the file is not in git, so you copy the example first or analyze warns. CI stops at `flutter analyze` and the `sdk/dart` tests. Nothing in CI runs the package tests, so if you changed a package, run `flutter test` inside it yourself, or `./flutter/scripts/coverage.sh` for all three with one merged lcov report.

### Docs

`docs/` is a Fumadocs (Next.js) site with its own lockfile, outside the `ui/` workspace: `pnpm lint` runs biome, and `pnpm types:check` and `pnpm build` do what they say. CI does not build it, so if you change the docs, build them before you push.

## Layout

| Path | What lives there |
|---|---|
| repo root `*.go` | package `authsome`: the `Engine` (`engine*.go`), its service methods (`service*.go`), `Option`s (`option.go`), `Config` (`config.go`) and bootstrap |
| `account/`, `user/`, `session/`, `app/`, `environment/`, `organization/`, `device/`, `apikey/`, `webhook/`, `notification/`, `serviceaccount/` and friends | domain types, each with its own `Store` interface |
| `store/` | the aggregate `store.Store`, the four backends (`postgres`, `mongo`, `sqlite`, `memory`) and the shared contract suite in `store/storetest` |
| `api/` | HTTP handlers on the Forge router; query and body structs in `api/requests.go` |
| `middleware/` | authentication and request-context middleware |
| `plugin/` | the plugin interfaces and registry |
| `plugins/` | the 26 plugins, one directory each, indexed in `plugins/README.md` |
| `extension/` | the Forge extension adapter, plus the dashboard contract in `extension/contract` |
| `bridge/` | slim local interfaces for sibling modules; `bridge/<name>adapter` wraps the real ones |
| `id/` | TypeID identifiers and their prefixes |
| `rbac/`, `authz/`, `bootstrap/warden/` | roles, permission checks and the embedded warden seed |
| `dpop/`, `tokenformat/`, `ratelimit/`, `lockout/`, `settings/`, `hook/`, `strategy/` | DPoP, token formats, rate limits, lockout, dynamic settings, the event bus, auth strategies |
| `internal/` | private helpers (`safeurl`, `secutil`, `mask` and more) and the `doclint` repo self-checks |
| `sdkgen/` | the spec dumper (`cmd/specgen`), the generator (`cmd`), templates under `golang/`, `typescript/` and `dart/`, and `spec.json` |
| `sdk/` | generated SDKs: `sdk/go` (package `authclient`, also used by the extension's client mode), `sdk/typescript`, `sdk/dart` |
| `ui/`, `flutter/`, `docs/` | npm UI workspace, Flutter workspace, docs site |
| `testutil/` | an HTTP test server backed by a real engine |
| `_examples/` | `forge` and `standalone` wiring |
| `scripts/` | release and CI helpers |

## Go conventions

### Errors

When you add a sentinel, make it a package-level `ErrX` with the package name as the message prefix: `store.ErrNotFound` reads "authsome: not found" and `account.ErrEmailTaken` reads "account: email already taken". Wrap with `%w` and the same kind of prefix (`fmt.Errorf("authsome: register engine in container: %w", err)`), and compare with `errors.Is` and `errors.As`, since `errorlint` and `errname` enforce both.

Each backend translates driver errors before they leave the store: `pgError` in `store/postgres`, `sqliteError` in `store/sqlite`, `mapWriteErr` in `store/mongo`. If you add a store method, run its errors through the same translator. A missing row becomes `store.ErrNotFound`, a duplicate key becomes `store.ErrConflict` or a more specific sentinel, and the raw constraint name never reaches an API response.

### IDs

Every entity ID is an `id.ID`, a TypeID over UUIDv7 printed as `prefix_suffix`. The prefixes are constants in `id/id.go` (`ausr` for users, `ases` for sessions, `aapp` for apps), each with a typed alias and a constructor like `id.NewUserID()`. Parse untrusted input with `id.ParseWithPrefix` so that a session ID cannot pass for a user ID. If you add an entity type, give it its own prefix, alias and constructor in that file.

### Stores and migrations

Each domain package defines its own `Store` interface, and `store.Store` in `store/store.go` embeds them all. Every backend implements the whole set. Change an interface and you change four backends in the same PR, plus `store/storetest`, which all four run.

Migrations are grove migration groups in `migrations.go` under `store/postgres`, `store/sqlite` and `store/mongo`, versioned by timestamp. Once a migration has shipped in a tag you never edit it. You add a new one. If you add an app-scoped table, it also has to join `DeleteApp`'s cascade in every backend, and `store/*/app_cascade_test.go` is where you prove that it does.

### Plugins

A plugin implements `plugin.Plugin` (only `Name()`) plus whichever optional interfaces from `plugin/plugin.go` it needs: `OnInit`, `BeforeSignUp`, `RouteProvider`, `MigrationProvider`, `SettingsProvider` and the rest. List them as compile-time assertions at the top of the plugin's main file. That block is how a reader finds out when the plugin runs. You register one with `authsome.WithPlugin(...)`.

A plugin that persists anything ships four backends of its own (`store_memory.go`, `store_postgres.go`, `store_sqlite.go`, `store_mongo.go`), shared cases in a `<name>test` package, a `conformance_test.go` for memory and SQLite and a `conformance_integration_test.go` that adds Postgres and Mongo. `plugins/passkey` has the full set if you want one to copy. Fixed configuration goes in the plugin's `Config` at construction; anything an operator should change at runtime is a setting declared with `settings.Define`.

### The Forge extension and bridges

`extension.New(opts ...extension.ExtOption)` returns an `*Extension` that embeds `forge.BaseExtension`. `Register` loads config, pulls siblings out of the vessel container with `vessel.Inject`, builds the engine and provides `*authsome.Engine` back with `vessel.Provide`. Chronicle and warden are required and registration fails without them, while keysmith, relay, vault, dispatch, ledger and herald are optional and a missing one falls back to a no-op bridge, so in a Forge app you register those two before authsome. With `ClientMode` set there is no local engine at all; the extension talks to a remote authsome through `sdk/go`.

Integrations go through the slim interfaces in `bridge/` (`bridge.Chronicle` is one), and the `bridge/<name>adapter` packages translate those to the sibling module's real API.

### Options, logging and context

Configuration is functional options throughout: `authsome.NewEngine(opts ...authsome.Option)` with `type Option func(*Engine)`, `extension.ExtOption`, `authsome.BootstrapOption`. If you add a knob, it's a `WithX` function next to its neighbours in `option.go`.

Logging uses `github.com/xraph/go-utils/log`, imported as `log`, with structured fields (`log.String("app_id", appID)`, `log.Error(err)`) and messages prefixed by package (`"authsome: ..."`). Where no logger was given, the code falls back to `log.NewNoopLogger()`. Don't reach for the standard library's `log` or `log/slog`.

`ctx context.Context` is always the first parameter (revive's `context-as-argument`), and `noctx` flags an HTTP request built without one, so you use `http.NewRequestWithContext`.

### Lint rules that bite

From `.golangci.yml`:

- `errcheck` has `check-blank` and `check-type-assertions` on, so `_ = f()` discarding an error and an unchecked `x.(T)` both fail outside tests.
- `govet` runs every analyzer except `fieldalignment`, shadow included. An inner `err :=` that shadows an outer `err` fails, which is why you'll see names like `ksErr` and `jsonErr` in the code.
- `nolintlint` reports a `//nolint` that no longer suppresses anything. Name the linter and say why: `//nolint:errcheck // reason`.
- `gosec` runs on everything except tests (G104, G115 and G117 are excluded). `_test.go` files also skip `errcheck` and `gocritic`, and `*_gen.go` files skip every linter.
- goimports runs with `local-prefixes: github.com/xraph/authsome`, so authsome imports sit in their own group. CI's Format job fails on any file `goimports -l -local github.com/xraph/authsome .` lists.

## Repo rules

### No go.work, ever

Never create a `go.work`, tracked or local. `.gitignore` lists `go.work*`, and `go.mod` carries the `toolchain` line. If your change needs an unreleased sibling module, that sibling gets released first and `go.mod` bumps to its tag.

Check a build with `GOWORK=off` (`GOWORK=off go build ./...`, `GOWORK=off go test ./...`), because Go otherwise picks up a `go.work` from any parent directory and you'd be testing something CI never sees. Never commit a local `replace` either; the `forge` one in `go.mod` stays commented out. The shared go-ci never sets `GOWORK=off`, and a committed local replace points CI at a path that does not exist on the runner, which breaks Lint, Test, Verify and Security at once.

### Regenerate the SDKs after route changes

CI's "SDK Generation" job runs these three, then fails on any diff under `sdk/`, `sdkgen/spec.json`, `ui/packages/core/src/generated` and `flutter/packages/authsome_core/lib/src/generated`:

```sh
make -C sdkgen generate SDK_OUT=../sdk
make -C sdkgen generate-ui-core
make -C sdkgen generate-flutter-core
```

Run all three after you change a route or a request or response struct, and after a forge bump (forge renders the OpenAPI document, so a bump can change the spec by itself), and commit the output in the same PR.

`specgen` compiles and boots a real engine from your working tree, so any uncommitted API work you have lying around leaks into the spec, and you have to generate from a tree whose Go code is exactly what you're committing. If you have other work in flight, use a worktree of the commit:

```sh
git worktree add --detach /tmp/authsome-gen <commit>
cd /tmp/authsome-gen
make -C sdkgen generate SDK_OUT=../sdk
make -C sdkgen generate-ui-core
make -C sdkgen generate-flutter-core
git add sdk sdkgen/spec.json ui/packages/core/src/generated flutter/packages/authsome_core/lib/src/generated
git commit -m "chore(sdk): regenerate clients"
```

Cherry-pick that commit onto your branch (worktrees share one object store), then `git worktree remove /tmp/authsome-gen`.

specgen registers a fixed list of plugins in `sdkgen/cmd/specgen/main.go`. If you add a plugin, its routes only reach the SDKs once you add it to that list. Generated files open with `Code generated by authsome sdkgen`. Change the templates, never the output. Hand-written companions such as `sdk/go/client_admin.go` and `sdk/go/client_refresh.go` survive regeneration.

### New query fields go last

Generated clients take query parameters positionally, in Go struct field order. `AdminListUsersRequest` declares `AppID`, `Email`, `Cursor`, `Limit`, and the TypeScript client gets `adminListUsers(app_id, email?, cursor?, limit?)`. If you put a new field above an existing one, every later argument shifts in the TS, Dart and embedded clients, which still compile and quietly send the wrong values. Add new query fields at the end of the struct.

### Repo self-checks

`internal/doclint` reads the tree. `TestSecurityDocsResolve` fails when a Go code fence under `docs/content/docs/security` names a `pkg.Identifier` that does not exist in this module, and `TestNoTrackedEnvFiles` fails when any `.env` or `.env.*` file is tracked (`.env.example` is the only exception). After you touch the security docs, run `go test ./internal/doclint/`.

The "Private Paths" job runs `scripts/check-private-paths.sh`, which fails if `cloud/`, `_project_files/`, `.superpowers/` or `docs/superpowers/` is tracked. They are gitignored. Don't `git add -f` them: the Go module proxy archives the whole tree at every tag and serves it forever, so anything you tag stays public for good, whatever you delete afterwards.

### Secret scan

The "Secret scan" job runs the gitleaks CLI, pinned by version and checksum in `ci.yml`, over the commits a PR adds. You can run the same scan locally:

```sh
gitleaks git --no-banner --redact --verbose --log-opts="--no-merges origin/main..HEAD" .
```

If you add a test fixture that looks like a credential, allowlist it in `.gitleaks.toml` by exact value: an anchored `'''^value$'''` regex in the existing `[[allowlists]]` block, with a comment naming the file it comes from. A real key in the same file still fails the scan.

### Landing changes

`main` has a ruleset that blocks force pushes and deletion and requires four checks: `Analyze (go)`, `Store Conformance (pg + mongo)`, `ci / Test (ubuntu-latest, go1.26)` and `gosec`. A commit has to pass them before it lands, so a direct push to main is rejected. You work on a feature branch, push it and open a PR.

`ci / Test (ubuntu-latest, go1.26)` embeds the Go minor from `go-versions` in `.github/workflows/ci.yml`. If you pin a patch there, the job gets renamed, the required check never reports, and every PR blocks. Keep the bare minor. The shared workflow sets `check-latest`, so it still resolves to the newest patch.

Commit messages are Conventional Commits, usually with a scope: `fix(api): ...`, `feat(store): ...`, `chore(deps): ...`.

## Dependencies

authsome requires these xraph modules, with versions in `go.mod`: chronicle, dispatch, herald, keysmith, ledger, relay, vault, warden, grove (plus `grove/drivers/mongodriver`, `pgdriver` and `sqlitedriver`), forge and `forge/extensions/auth`, vessel and go-utils. It sits near the top of the forgery graph, so it releases after all of them. Downstream, twinos (outside forgery) requires it, and so does dispatch's qualification harness (`dispatch/qualification/go.mod`).

To bump:

```sh
go get github.com/xraph/<module>@vX.Y.Z   # once per xraph module you move
./updatemod.sh                           # go mod tidy in every directory holding a go.mod
GOWORK=off go build ./...
GOWORK=off go test ./...
```

If forge moved, regenerate the SDKs as above. `updatemod.sh` finds modules with `find . -name go.mod`, so it also tidies any checkout sitting under `.claude/worktrees/`.

## Releasing

You cut a release by dispatching the release workflow, never by pushing a tag by hand. Wait for CI on `main` to go green on the commit, then:

```sh
gh workflow run release.yml --ref main -f tag=vX.Y.Z -f modules=all -f dry_run=true
```

`modules` takes `all`, `go`, `npm`, `flutter`, `go+npm`, `go+flutter` or `npm+flutter`. This is what the jobs in `release.yml` do:

- `prepare` always runs. On a dispatch it creates the tag on the dispatched commit, pushes it, writes release notes from `git log` since the previous tag and creates the GitHub release. `dry_run` does not gate this job, so even a dry run pushes the tag, and you can't dispatch the same tag a second time.
- `go` publishes nothing. It runs `go build ./...` and `go test -race -count=1 ./...`, and the tag is the Go release.
- `npm` builds `ui/` and `sdk/typescript`, then `scripts/prepare-npm-publish.sh <version>` stamps the tag's version (no `v`) into each package.json and rewrites `workspace:*` dependencies to `^<version>`. npm versions are never committed, so the `version` fields in the tree lag the tags and you leave them alone. It publishes `@authsome/client`, `@authsome/ui-core`, `@authsome/ui-react`, `@authsome/ui-vue`, `@authsome/ui-components` and `@authsome/ui-nextjs` with provenance, adding `--dry-run` when `dry_run=true`.
- `flutter` calls the shared `pub-publish.yml` for `authsome_core`, `authsome_flutter` and `authsome_flutter_ui`, in that order. It skips any version already on pub.dev, so you can re-run a release that failed halfway.

### Publishing the Flutter packages

pub.dev publishes through OIDC and only accepts a run triggered by a tag push. A `workflow_dispatch` run forces dry-run, so it can only rehearse the Flutter publish, and the tag that dispatch pushes does not start another run (GitHub does not trigger workflows from pushes made with a run's own `GITHUB_TOKEN`). The packages reach pub.dev only from a tag-push run, so don't push a tag yourself unless the maintainer asks for one.

pub.dev compares the tag with `version:` in each `pubspec.yaml` and refuses a mismatch, so you commit the versions before anything is tagged. `scripts/bump-flutter-version.sh 1.8.0` (no leading `v`) sets all three packages and their inter-package constraints.

`dart pub publish --dry-run` exits 65 on any warning, and a missing `CHANGELOG.md` is a warning. Every package has one. Keep it, and give it an entry for the version you're publishing.

`sdk/dart` (the `authsome` Dart package) is generated and tested in CI, but `release.yml` does not publish it.
