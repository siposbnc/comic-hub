# 08 — Roadmap

Phased delivery. Each phase is shippable and builds on the last. The MVP (Phase 1) is the
smallest thing that delivers the core promise: point at a folder, browse it, read in one
click, never lose your place.

## Phase 0 — Foundations (infra, ~weeks)

Plumbing that everything depends on. No user-facing features yet.

- Repo + monorepo layout (`/server` Go, `/client` Tauri, `/reader` Tauri, `/packages/shared` TS).
- Go server skeleton: chi router, config, slog, SQLite + migrations runner, `/healthz`.
- `Repository`, `ArchiveReader`, `PageProvider`, `Provider` interfaces stubbed.
- Tauri shells for client + reader; React + Vite + design tokens; shared API client package.
- Sidecar handshake (spawn, port/token, health) — embedded mode end-to-end empty.
- CI: build all three artifacts on Windows, lint, unit-test, bench harness.

**Exit:** client launches, spawns sidecar, both report healthy; reader opens an empty window.

## Phase 1 — MVP: Browse + Read (the core promise)

The minimum lovable product. Embedded mode, single implicit user.

**Server**
- Scanner: walk roots, classify, change-detect, parse **CBZ + CBR**, page lists, natural sort.
- ComicInfo.xml parsing; filename-heuristic fallback for series/number/year.
- libvips thumbnail generation at scan time; content-addressed cache.
- Page streaming endpoint + manifest; LRU page cache; prefetch hint.
- SQLite catalog: libraries, series, books, pages, progress.
- Progress upsert + Continue Reading + WS `progress`/`jobs` topics.

**Client**
- Add library (folder picker) → scan with live progress.
- Home (Continue Reading, Recently Added), Library grid, Series detail, Book detail.
- One-click **Read** launches the reader (connected mode).
- Design system v1: CoverCard, Rail, grid, hero, JobIndicator.

**Reader**
- Standalone (double-click `.cbz`/`.cbr`) **and** connected modes via `PageProvider`.
- Single + double page, fit modes, LTR/RTL, keyboard + mouse nav, scrubber, zoom/pan.
- Prefetch window (instant page turns), resume, mark-finished.
- Local progress store + reconcile-on-connect.

**Exit / success:** the [MVP success criteria](00-overview.md#7-success-criteria-mvp) all pass
on a 10k-issue test library.

## Phase 2 — A real library platform

Organization, metadata quality, more formats.

- **Formats:** CB7, CBT, and **PDF** (MuPDF rasterization, build-tagged).
- **Online metadata:** Comic Vine + GCD providers, matching engine, candidate picker,
  per-field locking, batch match. Optional ComicInfo.xml write-back.
- **Organization:** Collections, personal Reading Lists, Tags, Smart Lists (rule engine).
- **Search:** FTS5 full-text + type-ahead + `/discover` (On Deck, New Series).
- **File watching:** fsnotify incremental updates; move/rename reconciliation by hash.
- **Library Health:** corrupt files, unmatched, duplicates, orphans.
- **Bookmarks** + per-book reader overrides; continuous (webtoon) scroll mode.

**Exit:** a messy 30k library becomes clean, searchable, well-tagged with minimal manual work.

## Phase 3 — Multi-user & remote

Turn on the "optional server" half of the promise.

- **Auth mode:** accounts, argon2id, JWT access/refresh, OS-keychain token storage.
- **Roles:** owner/admin/member/restricted; per-user progress, lists, prefs.
- **Content restrictions:** age-rating ceilings for restricted users.
- **Remote deployment:** run server as a service (Windows Service / systemd / Docker);
  Postgres backend option; TLS guidance; reverse-proxy docs.
- **Cross-device sync:** live "now reading" presence; conflict-aware progress reconciliation.
- **Server discovery:** mDNS/Bonjour on LAN + manual URL pairing in the client.
- **Stats dashboards:** books/pages read, streaks, by genre/month.

**Exit:** a household runs one always-on server; each member reads independently from any client.

## Phase 4 — Polish, scale, reach

- **Mobile clients** (Tauri mobile / evaluate alternatives) — reader first, then browse.
- **Manga niceties:** AniList/MangaUpdates providers, RTL-first defaults, auto reading-dir.
- **Advanced reader:** auto-border-crop, color filters, two-page gutter tuning, dictionary/OCR hooks.
- **Organize/convert:** safe rename/move to a scheme, CBR→CBZ conversion (dry-run first).
- **Import/export:** catalog + reading state export; metadata backup.
- **Performance hardening:** AVIF pipeline, cache tuning, 100k+ library benchmarks.
- **EPUB (fixed-layout)** support; optional web client.

## Cross-cutting (every phase)

- Accessibility (keyboard, focus, reduced motion, contrast) is a per-phase gate, not a later task.
- Bench thresholds enforced in CI (scan throughput, page-serve latency, cache hit rate).
- Docs kept in lockstep: API changes update [03-api.md](03-api.md); decisions logged in
  [09-tech-decisions.md](09-tech-decisions.md).

## Sequencing rationale

We ship the **reading loop first** (Phase 1) because it's the core promise and the riskiest
performance work (instant page turns, scanner, image pipeline) — proving it early de-risks
everything. Organization (Phase 2) is high value but sits on top of a working catalog.
Multi-user/remote (Phase 3) is deliberately later: it's mostly auth + deployment surface and
shouldn't gate the single-user experience the brief centers on ("local-first, optional server").
