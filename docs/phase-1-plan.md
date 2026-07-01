# Phase 1 — Implementation Plan: Browse + Read

The working plan for delivering [Phase 1 of the roadmap](08-roadmap.md): _point at a
folder, browse it, read in one click, never lose your place_. Embedded mode, single
implicit owner. Exit = the [MVP success criteria](00-overview.md#7-success-criteria-mvp)
pass on a 10k-issue library.

This is a living document — milestone status is updated as work lands.

## Build order & rationale

Server backbone leads (the client/reader need real data; it's also the riskiest perf
work). The design system syncs in parallel (no server dependency). Then reader + client
wire onto the API.

```
M0 Design-system sync ─┐ (parallel)
                       ├─► C1 Client foundation ─► C2 Add+scan ─► C3 Browse ─► C4 Read CTA
S1 Catalog/Libraries ──┤
S2 Archives (CBZ/CBR) ─┼─► S3 Scanner+Jobs ─► S4 Image pipeline ─► S5 Browse+Progress+WS
                       └─► R1 Reader core (standalone) ─► R2 Reading UX ─► (connected via S4/S5)
```

## Key technical decisions

1. **Design system is the source of truth.** Tokens + components live in the ComicHub
   Design System (claude.ai/design); pulled into `packages/ui` incrementally via the
   DesignSync read API, kept Prettier-exempt for clean re-syncs. Adherence is **enforced**
   on app code via ESLint (`pnpm lint:ds`, in CI): raw hex/px/non-DS-font values and
   off-contract component props/variants are flagged (warn). See `CLAUDE.md`.
2. **Image pipeline: pure-Go now, govips later** (revised — the dev machine lacks a C
   toolchain/libvips, so govips can't build). Built behind an `image.Processor`
   interface; pure-Go impl (std image + x/image) ships in S4, govips swaps in behind the
   same interface once libvips + a compiler are installed (zero call-site changes).
3. **IDs = ULID; content hash = xxhash64** (sampled for large files).
4. **Hand-written typed repo methods** over `domain.Repository` (sqlc only if justified).
5. **PDF / AVIF / CB7 / CBT are Phase 2** — `capabilities` flags stay off in Phase 1.

## Server (Go) — `comichub-server`

- **S1 — Catalog store + Libraries API.** SQLite repos for Library/Series/Book/Page/
  Progress; pkg/ulid; library service (validation, path normalization); `GET/POST
/libraries`, `GET/DELETE /libraries/{id}`; implicit owner seed.
- **S2 — Archive readers.** `archive.Reader`/`PageSource` for CBZ (archive/zip) and CBR
  (nwaples/rardecode); natural page sort; ComicInfo.xml sidecar extraction; zip-bomb +
  traversal guards; registry dispatch by extension.
- **S3 — Scanner + job system.** Walk → classify → change-detect (mtime+size→hash) →
  parse → upsert series/book/pages. Filename heuristic parser + sort_number; ComicInfo
  parsing; SQLite-backed job queue + worker pool; `POST /libraries/{id}/scan` + cancel;
  resumable/idempotent; corrupt files flagged, never fatal.
- **S4 — Image pipeline + page streaming.** govips thumbnails at scan time
  (content-addressed WebP); LRU page cache; `GET /books/{id}/cover`, `/manifest`,
  `/pages/{idx}`, `/pages/{idx}/thumb`, `POST /prefetch`; ETag + immutable cache, ranges.
- **S5 — Browse + progress + WS.** `GET /series`, `/series/{id}`, `/books`,
  `/books/{id}`; `GET /me/continue`, `PUT /me/progress/{bookId}`, `POST
/me/books/{id}/mark`, `GET /discover`; WS hub `/api/v1/ws` (jobs/library/progress).

## Reader (Tauri + React) — `comichub-reader`

- **R1 — Reader core + providers.** `LocalPageProvider` (Rust core: open archive,
  content_hash, manifest, page bytes, `reader.db` progress) and `ServerPageProvider`
  (REST+WS) against the `PageProvider` interface. Standalone + connected launch.
- **R2 — Reading UX.** Single + double page, fit modes, LTR/RTL, keyboard + mouse nav,
  scrubber, zoom/pan, prefetch window, resume, mark-finished, reconcile-on-connect.

## Client (Tauri + React) — `comichub`

- **C1 — Foundation.** TanStack Query + Router + Zustand; consume `@comichub/ui`;
  AppShell (sidebar + utility bar) from the ui_kit; api-client over the handshake.
- **C2 — Add library + scan.** Native folder picker → `POST /libraries` → scan with live
  WS progress (JobIndicator).
- **C3 — Browse screens.** Home (Continue Reading + Recently Added), virtualized Library
  grid (CoverCard), Series detail (hero + issues), Book detail.
- **C4 — One-click Read.** Launch the reader in connected mode at the right page;
  progress reflects back live over WS.

## Cross-cutting

Accessibility gates per screen; bench thresholds in CI (scan throughput, page-serve
latency, cache hit); docs kept in lockstep (03-api.md, 09-tech-decisions.md).

## Status

| Milestone                            | Status                                                     |
| ------------------------------------ | ---------------------------------------------------------- |
| M0 — Design-system foundation synced | ✅ done                                                    |
| S1 — Catalog store + Libraries API   | ✅ done                                                    |
| S2 — Archive readers (CBZ + CBR)     | ✅ done                                                    |
| S3 — Scanner + job system            | ✅ done                                                    |
| S4 — Image pipeline + page streaming | ✅ done (pure-Go; govips swap later)                       |
| S5 — Browse + progress + WS          | ✅ done — **server backbone complete**                     |
| R1 / R2 — Reader                     | ✅ done (connected mode + UX; standalone CBZ/CBT via Rust) |
| C1–C4 — Client                       | ✅ done (shell, add+scan, browse, one-click read)          |

**Phase 1 is functionally complete and packaged** — server backbone, reader, and client all
built, integrated, and **building to installers**:

- Server bundled as a Tauri **sidecar** (`externalBin` + `tools/prepare-sidecar.mjs`); the
  client installer ships `comichub-server.exe` next to the app, so embedded zero-config mode
  works in an installed build. Embedded handshake (ephemeral port + token + serve + 401)
  verified.
- Reader registers the **`comichub-reader://`** deep-link scheme (+ cbz/cbr/cb7/cbt file
  associations); the client's one-click Read launches it.
- `tauri build` produces MSI + NSIS installers for **both** client and reader (WiX/NSIS).
- All TS packages typecheck; both Tauri shells `cargo build` clean.

**UI tracks Design Preview v2.** The client screens (home, library, series) and the
stylized "longbox" sidebar — vertical spine plate, `SpineTab` nav with registration-tick
hover and an active clipped cyan tab, and a live "continue reading" footer card — are built
to match the ComicHub Preview v2 client shell (`ClientPreview.jsx`). The design system was
re-synced in the process (full primitive set; Avatar, Dialog, Checkbox, Select, Switch
added). Mock-only elements (genre filters, writer/artist, the Lists/Stats nav) are omitted
until the data/features exist.

Remaining (follow-on, none blocking the core loop):

- **Verification:** ✅ _API-surface end-to-end done_ — drove the real server socket on a
  generated 1000-book/50-series library (add library → full scan → browse → manifest/cover/
  page/prefetch → progress → mark/Continue-Reading). MVP criteria 2 (browse <100ms cached),
  3 (page serve sub-ms + ETag immutable cache + prefetch), 4 (progress reflected in Continue
  Reading) all confirmed. The run surfaced and **fixed** a concurrent-scan data-integrity bug
  (duplicate series), scan double-processing, all-zero manifest dims, and read-percent <100%
  (commit `0553e00`). _Still to do:_ a GUI pixel run on a desktop session (window spawns
  sidecar → one-click → reader), MVP #5 (standalone double-click) at the GUI surface, the
  ~10k-library throughput/latency benchmark, and the accessibility audit.
- **Standalone CBR/CB7/PDF** in the reader (need native unrar/7z/mupdf; CBZ/CBT work today).
- **govips** image-pipeline swap (WebP/AVIF + scan-time thumbnail prewarm).
- ✅ _Done:_ the reader's control glyphs were folded into the design-system `Icon` and re-synced;
  the reader now uses DS `Icon` (only its `IconButton` stays local).
- Swap the Tauri **shell-outs** (folder dialog, URL open) for `tauri-plugin-dialog`/`-opener`.
- CI bench thresholds.

Design-system components (CoverCard, Rail, …) are pulled into `packages/ui` on demand
during the C-phase, one at a time.
