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
   DesignSync read API, kept Prettier-exempt for clean re-syncs.
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

| Milestone | Status |
|-----------|--------|
| M0 — Design-system foundation synced | ✅ done |
| S1 — Catalog store + Libraries API | ✅ done |
| S2 — Archive readers (CBZ + CBR) | ✅ done |
| S3 — Scanner + job system | ✅ done |
| S4 — Image pipeline + page streaming | ⬜ pending (next) |
| S5 — Browse + progress + WS | ⬜ pending |
| R1 / R2 — Reader | ⬜ pending |
| C1–C4 — Client | ⬜ pending |

Design-system components (CoverCard, Rail, …) are pulled into `packages/ui` on demand
during the C-phase, one at a time.
