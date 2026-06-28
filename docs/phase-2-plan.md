# Phase 2 — Implementation Plan: A real library platform

The working plan for [Phase 2 of the roadmap](08-roadmap.md): _a messy library becomes
clean, searchable, well-tagged with minimal manual work_. Phase 1 (browse + read) is
complete and verified. This is a living document — milestone status is updated as work lands.

## Build order & rationale

User-prioritized: **online metadata first** (the richest payoff and the riskiest external
surface — proving it early de-risks the phase), then **more formats incl. PDF** (we set up a
C toolchain this cycle), then search, organization, file-watching/health, and reader extras.

```
M Metadata (providers → matcher → schema/apply → API → client) ─► F Formats (CB7/CBT, PDF)
                                                                 ─► S Search ─► O Organization
                                                                 ─► W Watch+Health ─► R Reader extras
```

## What already exists (Phase 0/1 scaffolding to build on)

- **`providers.Provider`** interface + `SeriesCandidate`/`IssueCandidate`/`IssueMeta`
  (`server/internal/providers/provider.go`) — stubbed, ready for concrete providers.
- **`domain.MetadataState`** precedence `none < sidecar < matched < locked`
  (`server/internal/domain/models.go`); Series/Book already carry the metadata fields.
- **`series.provider_ids` JSON** column already in `0001_init.sql`; `tag`/`book_tag`,
  `collection`/`reading_list` tables already exist (organization foundation).
- Forward-only migrations (`store/sqlite/migrations/NNNN_*.sql`), the job runner + WS `jobs`
  topic, and the **DS `Dialog`** primitive (just synced) for the candidate picker.
- The **API contract is already specified** in [03-api.md §9](03-api.md) and the server
  design in [04-server.md §6](04-server.md) — this plan implements them.

## Key technical decisions

1. **Metadata uses the existing seams.** Concrete providers behind `providers.Provider`;
   API keys **server-side only** (config `api_key = "env:COMICVINE_API_KEY"`); a token-bucket
   rate limiter + on-disk response cache per provider. Matching runs as a `metadata_match`
   background job (interactive on-demand match preempts bulk), reusing the Phase 1 runner.
2. **Precedence + per-field locks.** `matched` never overwrites `locked` or any field the
   user pinned. Add per-field locks (a `locked_fields` JSON column on series/book, mirroring
   `provider_ids`) on top of the coarse `MetadataState`.
3. **PDF via MuPDF behind the `PageSource` seam, build-tagged (cgo).** Set up an mingw-w64 C
   toolchain now; the `kind:'document'` source rasterizes pages to images. Same seam as the
   deferred **govips** swap, so call sites don't change — and it unblocks govips too.
4. **Normalized people/genre/character tables** (deferred from Phase 1's denormalized
   ComicInfo parse) arrive with the metadata migration.

## M — Metadata (first track)

- **M1 — Provider framework + Comic Vine.** Matching engine (pure scoring: name similarity +
  year + issue-count/number) — `internal/providers/match.go` (+ tests, no key needed). Comic
  Vine `Provider` impl: HTTP client, server-side key, token-bucket, disk cache; fixtures +
  unit tests. (GCD second, same interface.)
- **M2 — Schema + apply pipeline.** Migration `0003_metadata.sql`: `person`+`book_person(role)`,
  `genre`+`book_genre`, `character`+`book_character`, `book.provider_ids`, `*.locked_fields`.
  Apply respecting locks; **batch match** across a series/library; `metadata_match` job +
  WS progress.
- **M3 — API.** `GET /providers`; `POST /books|series/{id}/match` (→ job or candidates);
  `GET …/match/candidates`; `POST …/match/apply {provider, providerId, fields[]}`;
  `PATCH /series|books/{id}` (sets `locked`); `POST /books/{id}/metadata/write-sidecar`
  (ComicInfo write-back, opt-in). Add `api-client` methods + wire types.
- **M4 — Client.** Candidate-picker (DS `Dialog`), per-field lock toggles on Series/Book
  detail, batch-match action with live `JobIndicator` progress, provider status in Settings.
  Flip the `pdf` capability surfacing once F2 lands.

## F — Formats

- **F1 — CB7 + CBT.** Pure-Go `bodgit/sevenzip` + stdlib `archive/tar` readers into the
  archive registry (`internal/archive/`); update `registry_test`. Codegen already emits them;
  flip nothing else.
- **F2 — PDF (MuPDF).** Set up mingw-w64; build-tagged `document` PageSource (rasterize via
  go-fitz/MuPDF) behind the `archive`/`reader` seam; flip the `pdf` capability flag; add the
  cgo build tag to CI.

## Later tracks (sequenced after M/F)

- **S — Search.** FTS5 virtual table + sync triggers; `GET /search?q=`; wire the existing
  TopBar search box → type-ahead results.
- **O — Organization.** Collections, Reading Lists, Tags, Smart Lists (rule engine) + CRUD
  APIs + the **real sidebar Lists nav** (replacing the mock nav omitted in the client redesign).
- **W — File-watching + Health.** fsnotify incremental updates; move/rename reconciliation by
  content hash; `GET /libraries/{id}/health` (orphans/corrupt/unmatched/duplicates).
- **R — Reader extras.** Bookmarks, per-book reader overrides, continuous (webtoon) scroll mode.

## Cross-cutting

Accessibility gate per screen; CI bench thresholds; docs kept in lockstep (03-api.md,
09-tech-decisions.md); ADRs for provider choice + the MuPDF build-tag split.

## Status

| Milestone | Status |
|-----------|--------|
| M1 — Provider framework + Comic Vine + matcher | ✅ done |
| M2 — Metadata schema + apply pipeline | ✅ done |
| M3 — Metadata API (candidates / apply / job) | ✅ done |
| M4 — Metadata client UI (candidate picker) | ✅ done — **metadata track complete** |
| F1 — CB7 + CBT | ✅ done |
| F2 — PDF (MuPDF + C toolchain) | ⏳ deferred (skipped for now) |
| S — Search (FTS5 + `/search` + TopBar type-ahead) | ✅ done |
| O — Organization | ✅ done — collections, reading lists, tags, smart lists (rule engine) end-to-end: server + SDK + client (sidebar Lists/Tags nav, index/detail screens, smart-list rule builder, Add-to-list + Edit-tags on Book) |
| W / R — watch/health, reader | ⏳ pending |

Remaining metadata polish (non-blocking): GCD as a second provider, ComicInfo
`write-sidecar`, per-field lock toggles in the UI, and a book-level candidate picker.

## Verification

Per milestone: unit tests (matcher scoring; provider response parsing from fixtures — no live
key needed). Live match needs a `COMICVINE_API_KEY`. End-to-end: drive the candidate picker on
the running client (`localhost:1420`) via Playwright; for F2, open a `.pdf` in the reader.
