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

**The source of all design is the `Design Preview v2` project on claude.ai/design**
(project id `ef2d1724-12c0-48dd-98e8-996e5b3ee416`, owner Sickae). It holds everything:
the **preview screens** (`ComicHub Preview Screens.dc.html` + `ClientPreview.jsx`), the
**design kit** (`ComicHub Design Kit.dc.html` + `DesignKit.jsx`), per-feature **handoffs**
(`design_handoff_<feature>/README.md` + `ClientPreview.jsx`), and an embedded snapshot of
the design-system components/tokens under `_ds/comichub-design-system-c0e1bfbe…/`. Reach it
with the `DesignSync` read API (`list_files` / `get_file`). Look here first for any design.

The visual foundation and components in `packages/ui/src` are **synced verbatim** from that
design system (the embedded snapshot's canonical home is the design-system-type project
`c0e1bfbe-c5d5-422a-b364-afc6dcdde00b`; pull component/token files from there). Do **not**
hand-edit `src/styles.css`, `src/tokens/**`, or `src/components/**` — they are listed in
[`.prettierignore`](.prettierignore) to stay byte-faithful so re-syncs diff cleanly.
`src/index.ts` (the barrel apps import from) **is** authored here.

### Designing any UI — preview-screen first (REQUIRED, not optional)

The `Design Preview v2` project (`ef2d1724-12c0-48dd-98e8-996e5b3ee416`) is the source of
truth for how the app looks. Before writing or changing **anything that touches UI** —
screens, components, layouts, dialogs, empty states, or styling in `apps/client` or
`apps/reader` — follow this, every time:

1. **Look for a preview screen of it in `Design Preview v2` first.** `DesignSync`
   `list_files` → `get_file` against `ef2d1724…`. Preview screens live in
   `design_handoff_<feature>/` (a `README.md` spec + a `ClientPreview.jsx`), in
   `ClientPreview.jsx` / `ComicHub Preview Screens.dc.html` at the root, and the design kit
   (`DesignKit.jsx` / `ComicHub Design Kit.dc.html`). Build the UI to match that preview,
   using the design-system components and tokens.
2. **If no preview screen exists for what you're building, stop and ask the user to make
   one** in `Design Preview v2` that you can reference — then build against it. Do **not**
   invent the layout/visuals yourself.
3. **Only design UI yourself when the user explicitly approves** it for that specific
   piece. Absent that approval, the preview screen is mandatory.

This is a hard rule: no improvising UI from scratch. When in doubt whether something
"touches UI", treat it as yes and check `Design Preview v2`.

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

## Regenerating README screenshots / demo GIF

The user-facing [README.md](README.md) embeds real screenshots (`docs/assets/*.png`) and an
animated tour (`docs/assets/demo.gif`) captured from the app running against a throwaway demo
library. The generators live in [`.demo/`](.demo/) (PowerShell, Windows-only, GDI+/WPF — no
ffmpeg/ImageMagick needed). To reproduce:

1. **Generate a demo library** (DC-themed CBZ + `ComicInfo.xml`, covers drawn with GDI+):
   `pwsh .demo/gen-library.ps1` → writes to `%TEMP%\comichub-demo-library`.
2. **Run the server standalone** so a browser can reach it:
   `comichub-server.exe --mode server --bind 127.0.0.1:8099 --data-dir <tmp>` (auth stays off
   without `--auth`; the client's web fallback in `apps/client/src/connection.ts` targets
   `127.0.0.1:8099`).
3. **Create + scan the library** via REST: `POST /api/v1/libraries {name,roots}` then
   `POST /api/v1/libraries/{id}/scan`; poll `GET /api/v1/jobs/{id}`. Seed some
   `PUT /api/v1/me/progress/{bookId}` so Continue Reading populates.
4. **Series headers** (publisher/year/description + a green "matched" badge) come from an
   online provider, which the demo has no key for — seed them straight into the demo SQLite
   (`UPDATE series SET publisher=…, year=…, description=…, metadata_state='matched',
match_provider='comicvine'`). Book-level metadata already comes from the sidecar.
5. **Run the frontends** on their fixed dev ports and drive them with the Playwright MCP:
   client `pnpm dev:client`-style Vite on `:1420`, reader `:1421` (open a book with
   `?bookId=<id>&server=http://127.0.0.1:8099`). The reader's auto-hiding chrome can be forced
   on for a screenshot by adding `chrome-on` to `.reader-shell`.
6. **Assemble the GIF:** `pwsh .demo/make-gif.ps1 -Frames <pngs> -Out docs/assets/demo.gif`
   (WPF `GifBitmapEncoder` + manual NETSCAPE-loop / per-frame-delay byte injection).

`.demo/` is committed tooling; the generated library/data lives in `%TEMP%` and is never
committed. `.playwright-mcp/` scratch is gitignored.
