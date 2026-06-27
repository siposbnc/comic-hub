# CLAUDE.md

Agent instructions for working in the ComicHub repo. Read this first; it captures the
conventions and gotchas that aren't obvious from a glance at the tree.

## What this is

ComicHub is "Plex, but for comics" — three artifacts in one monorepo:

- **`server/`** — a single Go binary (the media server): scanning, metadata, thumbnails,
  page streaming, progress, background jobs. One Go module.
- **`apps/client/`** — Tauri 2 + React desktop app for browsing/managing the library.
  Bundles the server as a sidecar in dev/prod.
- **`apps/reader/`** — a separate, lightweight Tauri + React reader (standalone `.cbz`
  double-click _or_ launched from the client with progress sync).

Shared TS lives in `packages/`: **`api-client`** (typed REST/WS client + wire types),
**`ui`** (the design system), **`reader-core`** (PageProvider + reading logic).

One Go module (`server/`) and one pnpm workspace (`apps/*` + `packages/*`) coexist; Rust
lives inside each `src-tauri/`. Full specs are in [`docs/`](docs/) (`00`–`09`, indexed in
[README.md](README.md)) — consult them before large changes.

## Commands

All from the repo root unless noted. Shell is **PowerShell** (a Bash tool is also available).

| Task                                      | Command                                                                  |
| ----------------------------------------- | ------------------------------------------------------------------------ |
| Install JS deps                           | `pnpm install`                                                           |
| Typecheck everything                      | `pnpm typecheck`                                                         |
| Design-system adherence lint              | `pnpm lint:ds`                                                           |
| Full lint (per-package `tsc` + adherence) | `pnpm lint`                                                              |
| Format / check                            | `pnpm format` · `pnpm format:check`                                      |
| Regenerate format definitions             | `pnpm codegen` (verify: `pnpm codegen:check`)                            |
| Run the client (spawns server sidecar)    | `pnpm dev:client`                                                        |
| Run the reader                            | `pnpm dev:reader`                                                        |
| Build server binary                       | `cd server && go build -o bin/comichub-server.exe ./cmd/comichub-server` |
| Server tests / vet                        | `cd server && go test ./...` · `go vet ./...`                            |

Per-package `lint`/`typecheck` are just `tsc --noEmit` — there is no general JS linter;
the only ESLint config is the design-adherence one (below).

## Definition of done (match CI)

CI (`.github/workflows/ci.yml`) runs three jobs. Before considering a change complete:

- **web:** `pnpm codegen:check`, `pnpm typecheck`, `pnpm lint:ds`, then client + reader builds.
- **server:** `go build ./...`, `go vet ./...`, `go test ./...` (in `server/`).
- **rust:** `cargo check` in each `apps/*/src-tauri`.

`pnpm lint:ds` is non-blocking (warnings only) but is expected to stay free of **errors**.

## Design system — `packages/ui` is synced, not authored

The visual foundation and components in `packages/ui/src` are **synced verbatim from the
ComicHub Design System** on claude.ai/design (project
`c0e1bfbe-c5d5-422a-b364-afc6dcdde00b`, the source of truth). Do **not** hand-edit
`src/styles.css`, `src/tokens/**`, or `src/components/**` — they are listed in
[`.prettierignore`](.prettierignore) to stay byte-faithful so re-syncs diff cleanly.
`src/index.ts` (the barrel apps import from) **is** authored here.

### Workflow: syncing from the design system

1. Pull with the `DesignSync` read API (or the `/design-sync` skill), **one component at a
   time** — never a wholesale replace. `list_files` → `get_file` against project
   `c0e1bfbe…`. The canonical project may be reached via a "Design Preview" project that
   embeds a `_ds/comichub-design-system-…/` snapshot; prefer the real project id.
2. Write each component's `.jsx` + sibling `.d.ts` into `src/components/<group>/`, then
   re-export it from `src/index.ts`.
3. If the design system's component contracts changed, update the adherence rules (below).
4. `pnpm typecheck` to confirm the barrel + apps still compile.

Apps import only from the `@comichub/ui` barrel and `@comichub/ui/styles.css` (linked once
at each app's `main.tsx`). Reach for semantic token aliases (`--accent`, `--surface-card`,
`--text-primary`, `--cover-w-m`, `--font-display`, …), not the raw ramp.

### Enforcement — ESLint, **not** oxlint

`eslint.config.mjs` at the repo root ports the design system's adherence spec
(`_adherence.oxlintrc.json`) and is what `pnpm lint:ds` runs. It flags, in app code: raw
hex/px/non-DS fonts, off-contract component `variant`/`size`/`tone`/`Icon name` values,
unknown component props, and deep imports past the `@comichub/ui` barrel.

- **Do not switch this to oxlint.** oxlint silently no-ops `no-restricted-syntax` (the rule
  the entire spec is built on), so it would produce hollow, always-green enforcement. ESLint
  runs the esquery selectors for real. The header comment in the config explains this.
- When a design-system component's prop/variant contract changes, update the `PROPS` and
  `valueRules` data structures in `eslint.config.mjs` (it generates the selectors from them).
- Severity is `warn`, matching the source — violations surface but don't block builds.
  `packages/ui/src` is exempt (it's verbatim source and legitimately contains raw values).
- The **reader gets token-rules only**: it keeps its own local `Icon` / `IconButton`
  (reader-specific glyphs) that collide by name with the DS contracts. The **client** gets
  full contract enforcement.

## Generated format definitions — never hand-edit `*_gen.*`

[`tools/codegen/formats.json`](tools/codegen/formats.json) is the single source of truth for
supported comic formats. `pnpm codegen` renders it into three generated files (do not edit
by hand):

- `packages/reader-core/src/formats.gen.ts` (frontends)
- `apps/reader/src-tauri/src/formats_gen.rs` (reader core)
- `server/internal/domain/formats_gen.go` (server)

After changing `formats.json`: run `pnpm codegen`, and if you changed any `associate: true`,
also update `apps/reader/src-tauri/tauri.conf.json` `fileAssociations` to match (the
generator fails loudly until they agree). `pnpm codegen:check` enforces this in CI.

## Conventions & gotchas

- **Prettier:** single quotes, semicolons, `trailingComma: all`, width 100, 2-space tabs.
  Run `pnpm format` before committing TS/JSON/MD/CSS.
- **TS config** (`tsconfig.base.json`) is strict: `noUncheckedIndexedAccess`,
  `verbatimModuleSyntax`, `noUnusedLocals/Parameters`. Imports use explicit `.js` extensions
  (e.g. `from '../lib/client.js'`) per `Bundler` resolution + `verbatimModuleSyntax`.
- **Windows:** `pnpm install` can hit transient `EPERM` file locks (AV/indexer) mid-write —
  just retry. Don't run two `pnpm` mutations concurrently.
- **Git:** commit only when asked; branch off `main` first. End commit messages with the
  `Co-Authored-By: Claude` trailer.
