# 06 — Reader (Tauri + React)

The reader is a separate, lightweight Tauri app — a first-class product, not a webview
afterthought. It must open a loose `.cbz` instantly **with no server**, and sync progress
seamlessly **when a server is present**.

## 1. Two operating modes

|                | **Standalone**                                                  | **Connected**                                                |
| -------------- | --------------------------------------------------------------- | ------------------------------------------------------------ |
| Trigger        | Double-click `.cbz`/`.cbr`/… (file association)                 | Launched from client, or file-open while a server is running |
| Page source    | Read archive **directly from disk** (Rust side)                 | Server `/pages/{idx}` streaming                              |
| Manifest       | Built locally by the Rust core                                  | `GET /books/{id}/manifest`                                   |
| Progress       | Local store keyed by `content_hash` (`reader.db`)               | `PUT /me/progress`, live WS sync                             |
| Metadata       | `ComicInfo.xml` from the archive                                | Full catalog metadata                                        |
| Reconciliation | On next server contact, local progress merges by `content_hash` | n/a                                                          |

The UI is identical; only a `PageProvider` abstraction differs (see §4). Mode is decided at
launch and can upgrade (standalone → connected) if a server appears.

## 2. Launch flows

- **File association:** OS runs `comichub-reader "D:\Comics\Saga 001.cbz"`. Rust core opens
  the archive, computes `content_hash`, checks `connection.json` for a live server:
  - server + book known by hash → connected mode (sync progress).
  - else → standalone; load directly; restore local progress by hash.
- **From client:** deep link `comichub-reader://open?server=…&bookId=…&token=…&page=…`
  (token short-lived, single-book scope). Connected mode, jump to `page`.
- **Single-instance:** a second open reuses the running reader (new tab/replace), via Tauri
  single-instance plugin + IPC.

## 3. Reading experience

### 3.1 Page layout modes

- **Single page** (fit options below).
- **Double / spread** (two pages side-by-side; auto-detects wide/double-spread pages and
  shows them solo; configurable cover-alone for correct spread pairing).
- **Continuous vertical** (webtoon/manga scroll) — seamless stitched scroll.
- **Continuous horizontal**.

### 3.2 Fit modes

- Fit width, fit height, fit screen (contain), original size, and **smart fit** (fills the
  reading area, respecting max zoom). Per-book and global defaults; remembers last used.

### 3.3 Reading direction

- LTR / RTL (manga). Inherited from series/library (`reading_dir`) in connected mode;
  detected from `ComicInfo.xml` Manga flag in standalone; user-overridable. Affects page-turn
  direction, spread pairing, and scrubber orientation.

### 3.4 Navigation & input

- **Keyboard:** ←/→ or Space/Shift-Space (direction-aware), Home/End, `F` fullscreen, `+`/`-`
  zoom, `D` toggle double, `V` toggle vertical, `B` bookmark, `Esc` exit, `1-9` jump to %,
  `,`/`.` prev/next chapter (book in series, connected mode).
- **Mouse:** click left/right zones to turn; scroll to zoom or scroll-to-pan (mode-aware);
  drag to pan when zoomed.
- **Touch/trackpad (mobile-later):** swipe to turn, pinch-zoom, double-tap zoom, edge taps.
- **Page scrubber:** bottom bar with thumbnail preview on hover/drag; current page / total;
  jump anywhere.

### 3.5 Zoom & pan

- Smooth pinch/scroll zoom centered on cursor; momentum pan; double-click/tap to toggle
  fit↔100%; constrained to image bounds.

### 3.6 Visual comfort

- Background color (black/gray/white/sepia), optional page gap, two-page gutter handling.
- Brightness/dim overlay; optional auto-crop of uniform page borders.
- Color filters: grayscale, sepia, night/warmth (eye comfort). All client-side, non-destructive.

## 4. Page provider abstraction

```ts
interface PageProvider {
  manifest(): Promise<Manifest>; // page count, dims, types, reading dir
  page(idx: number, opts?: PageOpts): Promise<Blob>; // full image
  thumb(idx: number): Promise<Blob>; // scrubber thumbnail
  prefetch(from: number, count: number): void;
  saveProgress(p: Progress): void; // debounced upstream
  restoreProgress(): Promise<Progress | null>;
}
```

- **ServerPageProvider** → REST endpoints + WS progress, server-side resize via `?w=&fmt=`.
- **LocalPageProvider** → Tauri commands into the Rust core that extracts pages from the
  archive on disk and returns image bytes; progress to local `reader.db`.

The Rust core does the heavy archive/PDF work (off the UI thread); React renders. PDF pages
are rasterized by the core (MuPDF) at a DPI matched to zoom level.

## 5. Prefetching & rendering pipeline

The point of the reader is that **the next page is already there**.

- Maintain a sliding window: keep pages `[current-1 .. current+N]` decoded in memory
  (N configurable, default 3–5 forward, 1 back; direction-aware).
- Decode off the main thread (Rust core / web worker for the server-bytes path); hand React
  a ready `ImageBitmap`/object URL so a page turn is a swap, not a load.
- Render to a `<canvas>` (or `<img>` with `decoding=async`) for precise zoom/pan and filters;
  continuous mode uses a virtualized scroller with recycled page surfaces.
- Memory cap: LRU-evict pages outside the window; downscale very large pages to the viewport
  resolution for display (full-res only when zoomed in).
- Connected mode also calls `POST /prefetch` so the **server** warms its page cache ahead of you.

## 6. Progress & bookmarks

- Track current page continuously; **persist debounced** (every few turns or on idle/blur/quit)
  to avoid write storms.
- Mark `finished` when reaching the last page (configurable: last page vs near-end).
- **Bookmarks:** per-book named bookmarks (page + optional note), synced in connected mode.
- **Resume:** reopening jumps to last page with a subtle "Resume from p.N / Start over" affordance.
- Cross-device: WS `progress.updated` means if you read the same book elsewhere, the reader
  reconciles (last-writer-wins by `updatedAt`), prompting if there's a meaningful conflict.

## 7. Chapter/series flow (connected mode)

- At the last page: **"Next: {Series} #{n+1}"** card → one click loads the next book's
  manifest + first pages (already prefetched), continuing the binge with no trip back to the
  library. End-of-series shows completion + stats.

## 8. Settings (reader)

- Global defaults + per-book overrides: layout mode, fit, reading direction, background,
  page gap, prefetch depth, image format/quality (connected), filters, "mark finished" rule.
- Stored in user prefs (connected) and locally (standalone); merged on connect.

## 9. Performance targets

| Action                                  | Target                                |
| --------------------------------------- | ------------------------------------- |
| Cold open of a 40-page CBZ (standalone) | First page visible < 400ms            |
| Page turn (within prefetch window)      | < 16ms (next frame) — no spinner ever |
| Zoom interaction                        | 60fps                                 |
| Connected open (warm cache)             | First page < 250ms                    |
| Memory (single book, single mode)       | Bounded by LRU window, not page count |

## 10. Resilience

- Corrupt/missing page → placeholder + skip option, never a crash; report to Library Health
  in connected mode.
- Server drops mid-read (connected) → fall back to cached pages, queue progress writes, retry;
  if the same file is local, transparently degrade toward direct read.
- Standalone progress is durable (local DB) and reconciled later — closing without a server
  never loses your place.

## 11. Why a separate app (recap)

- **Instant, dependency-free** double-click reading (the brief's hard requirement) — no need
  to boot the full client or a server.
- **Smaller, focused binary**; can be updated independently; clean failure isolation.
- Shares the React design system + `PageProvider` server path with the client via a common
  package, so there's no duplicated reading code despite being a separate binary.
