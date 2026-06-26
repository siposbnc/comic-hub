# 04 — Media Server (Go)

The server is a single static binary that owns the library, database, files, caches, and
background work. It's the only component that touches your comic files.

## 1. Tech choices

| Concern | Choice | Why |
|---------|--------|-----|
| Language | Go 1.22+ | Fast, simple concurrency, single static binary, great I/O. |
| HTTP | `net/http` + `chi` router | Stdlib-grade, fast, middleware-friendly, no heavy framework. |
| DB | SQLite (`modernc.org/sqlite` pure-Go, or `mattn/go-sqlite3`) | Embedded, zero-admin, WAL. Postgres via same repo interface for remote. |
| Query layer | `sqlc` (typed queries) | Compile-time-checked SQL, no ORM overhead. |
| Images | libvips via `govips` | Best-in-class speed/memory for thumbnail + resize + transcode. |
| Archives | `archive/zip` (CBZ), `nwaples/rardecode` (CBR), `bodgit/sevenzip` (CB7), `archive/tar` (CBT) | Pure-Go where possible; read-only RAR. |
| PDF | `go-fitz` (MuPDF) or `pdfium` binding | Page rasterization. Optional build tag (CGo). |
| Jobs | In-process worker pool + SQLite-backed queue | No external broker; survives restarts. |
| Logging | `log/slog` (JSON) | Structured, stdlib. |
| Config | `koanf`/`viper` (TOML + env + flags) | Layered config. |

> **CGo note:** libvips and MuPDF need CGo. We ship prebuilt Windows binaries with these
> statically linked (or as bundled DLLs alongside the binary). A `nocgo` build drops PDF +
> AVIF and uses a pure-Go image path for max portability — chosen via build tags.

## 2. Package layout

```
cmd/comichub-server/main.go        wiring, flags, lifecycle
internal/
  config/                          load + validate config
  transport/http/                  router, handlers, middleware, WS hub
  service/
    library/  reading/  lists/  metadata/  reader/  admin/  search/
  domain/                          entities, value objects, business rules
  store/
    sqlite/   (sqlc-generated + repo impls)   postgres/   migrations/
  scanner/                         walk, classify, hash, parse
  archive/                         ArchiveReader implementations
  image/                           libvips pipeline, thumbnailer, transcoder
  pdf/                             rasterizer (build-tagged)
  providers/                       comicvine/ gcd/ metron/ anilist/ (Provider iface)
  jobs/                            worker pool, queue, job types
  watch/                           fsnotify-based incremental watcher
  pkg/  (ulid, xxhash, sortkey, safepath, …)
```

## 3. Scanner

The scanner turns a folder tree into catalog rows. It is **incremental, parallel, resumable, and idempotent**.

### 3.1 Pipeline

```
walk roots ──▶ classify ──▶ change-detect ──▶ parse ──▶ persist ──▶ enqueue thumbs/match
 (fs.WalkDir)   (ext+depth)   (mtime+size,      (archive    (upsert    (jobs)
                               then hash)         + name)     book/series)
```

1. **Walk** each library root (`fs.WalkDir`), skipping hidden/system dirs and non-comic files.
   Bounded directory concurrency.
2. **Classify** by extension → format. Detect series from folder structure (parent dir =
   series by default; configurable depth/regex).
3. **Change-detect:** compare `(file_size, file_mtime)` against catalog. Unchanged → skip.
   Changed/new → compute `content_hash` (xxhash64; sampled for very large files) for dedup.
4. **Parse** (worker pool, size = `GOMAXPROCS`):
   - Open archive, list entries, natural-sort image entries → page list + count.
   - Read `ComicInfo.xml` if present → metadata.
   - Read cover page (first image or `FrontCover`-typed page) dims.
5. **Persist:** upsert series (create if new), upsert book, replace pages, set
   `metadata_state` (`sidecar` if ComicInfo found, else `none`).
6. **Enqueue** thumbnail generation and (if enabled) online metadata matching.

### 3.2 Filename parsing

When no `ComicInfo.xml`, derive metadata from path/filename with a layered parser:

- Series name from folder; fall back to filename stem.
- Issue number, volume, year, special markers (`Annual`, `One-Shot`, `TPB`, `v01`) via a
  set of ordered regexes (ComicTagger/Mylar-style heuristics).
- Produce `number` (string) + `sort_number` (real). Examples:
  `Saga 001 (2012).cbz` → series "Saga", number "1", year 2012.
  `Batman Annual 02.cbz` → number "Annual 2", sort after regular issues.

### 3.3 Resumability & safety

- Scan state persisted as a `job` with a cursor; restart resumes from last committed batch.
- Per-file failures (corrupt archive) are recorded (`is_corrupt`) and reported in Library
  Health — never abort the whole scan.
- Archive extraction guarded against zip-bombs: caps on entry count, per-entry and total
  uncompressed bytes, and nesting depth. All paths validated against root (no traversal).

### 3.4 File watching (incremental)

- `fsnotify` watches library roots; debounced events (coalesce bursts) trigger targeted
  incremental scans of affected paths.
- Moves/renames reconciled by `content_hash` so progress/lists follow the file.
- Watching is optional (config) for network shares where notifications are unreliable; a
  periodic incremental rescan is the fallback.

## 4. Archive & page abstraction

```go
type PageSource interface {
    PageCount() int
    Page(i int) (io.ReadCloser, PageInfo, error) // decoded-on-read image bytes
    Sidecar() (io.Reader, bool)                  // ComicInfo.xml if present
    Close() error
}
type ArchiveReader interface { Open(path string) (PageSource, error) }
```

- One implementation per format. The rest of the system is format-agnostic.
- PDF's `PageSource` rasterizes pages on demand at a target DPI (cached).
- Readers are pooled/cached (keep recently-opened archives' central directories warm) to
  avoid re-parsing on each page request.

## 5. Image pipeline (libvips)

The hot path for both grids and the reader.

- **Thumbnails:** generated at scan time at a few fixed widths (e.g. 200/300/450 for grids,
  plus a tiny page-scrubber size). Stored content-addressed as WebP (AVIF optional).
  `vips thumbnail` does decode+resize in one streamed pass (low memory).
- **Reader pages:** served at original resolution by default; `?w=&fit=&fmt=&q=` triggers a
  cached server-side resize/transcode. Useful for constrained clients / bandwidth.
- **Format negotiation:** WebP default (great ratio + universal-enough), AVIF when client
  advertises support and CPU budget allows, JPEG fallback.
- **Caching:** every derived image is keyed by `(content_hash, page, w, fit, fmt, q)`.
  Thumbnails persist; full resized pages live in an **LRU page cache** with a size cap.
- **Concurrency control:** a semaphore caps simultaneous vips operations to avoid memory
  spikes during a scan + active reading at once.

## 6. Metadata subsystem

### 6.1 Sources & precedence

1. **User edits** (`metadata_state = locked`) — never overwritten.
2. **Online provider** match (`matched`).
3. **`ComicInfo.xml`** sidecar (`sidecar`).
4. **Filename heuristics** (`none`).

Higher precedence wins per-field; a field locked by the user is sticky.

### 6.2 Providers (`Provider` interface)

```go
type Provider interface {
    Name() string
    SearchSeries(ctx, query) ([]SeriesCandidate, error)
    Issues(ctx, seriesID) ([]IssueCandidate, error)
    Issue(ctx, issueID) (IssueMeta, error)
}
```

- **Comic Vine** (western comics; rate-limited API key), **Grand Comics Database**,
  **Metron**, **AniList / MangaUpdates** (manga). All behind the interface; configured with
  server-side API keys (never exposed to clients).
- **Matching engine:** scores candidates by series name similarity, issue number, year,
  page count proximity. Auto-applies above a confidence threshold; otherwise surfaces ranked
  candidates for manual pick (`/match/candidates`).
- Respect provider rate limits with a token-bucket; cache responses on disk.

### 6.3 ComicInfo.xml

- Parse the full ComicRack/Anansi schema (Series, Number, Volume, Writer, Penciller, …,
  Web, AgeRating, Manga/RTL, PageType per page, DoublePage).
- Optional **write-back**: on explicit user action, inject/update `ComicInfo.xml` into the
  archive (CBZ only; never repacks RAR). Atomic write to temp + rename.

## 7. Job system

- SQLite-backed queue + in-process worker pool. Job types: `scan`, `thumbnail`,
  `metadata_match`, `watch`, `organize`, `export`, `cache_gc`.
- Each job reports `progress/total` → broadcast on WS `jobs` topic.
- Priorities: interactive (reader prefetch, on-demand match) preempt bulk (full scan).
- Cancelable via context; idempotent so a re-run after crash is safe.
- Backpressure: bounded queue; the scanner yields to keep the API responsive.

## 8. "Organize" (opt-in, never default)

- A dry-run-first feature to rename/move files into a consistent scheme
  (`{Series}/{Series} {Number} ({Year}).cbz`) and/or convert CBR→CBZ.
- Always preview the full plan; require explicit confirm; operate atomically per file with
  rollback on failure; update catalog paths in the same transaction.

## 9. Config (`config.toml`)

```toml
mode = "embedded"            # embedded | server
bind = "127.0.0.1:0"         # 0 = ephemeral (embedded)
data_dir = "%APPDATA%/ComicHub"

[database]
driver = "sqlite"            # sqlite | postgres
# dsn = "postgres://…"       # remote installs

[cache]
page_cache_max_mb = 2048
thumb_widths = [200, 300, 450]

[images]
prefer_format = "webp"       # webp | avif
allow_avif = true

[scan]
watch = true
hash_large_threshold_mb = 256

[providers.comicvine]
enabled = true
api_key = "env:COMICVINE_API_KEY"
```

Flags/env override file values; secrets resolvable from env so config can be committed.

## 10. Testing strategy

- **Unit:** filename parser, sort-key derivation, smart-list rule→SQL, metadata precedence,
  safepath, zip-bomb guards.
- **Golden files:** a corpus of real-world-messy filenames → expected parse; sample
  archives (incl. deliberately corrupt) for the scanner.
- **Integration:** spin server with a temp data dir + fixture library; assert scan results,
  API responses, image bytes (dimension/format assertions).
- **Bench:** scan throughput (books/sec), thumbnail latency, page-serve latency, cache hit
  rate. Tracked in CI to catch regressions.
- **Property tests:** progress reconciliation (last-writer-wins, offline batch merges).
