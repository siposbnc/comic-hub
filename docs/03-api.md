# 03 — API

One HTTP + WebSocket surface serves all clients in all deployment modes. REST is
resource-oriented and JSON; binary endpoints (pages, thumbs, covers) stream images with
content-addressed caching.

- **Base:** `/api/v1`
- **Auth:** `Authorization: Bearer <token>` (loopback token in embedded mode; JWT access
  token in auth mode). Image endpoints also accept a short-lived signed query token so
  `<img src>` works without headers.
- **Errors:** RFC-7807-ish `{ "error": { "code", "message", "details?" } }` with proper
  HTTP status.
- **Pagination:** cursor-based — `?limit=50&cursor=<opaque>` → `{ items, next_cursor }`.
- **Filtering/sort:** `?sort=field[:dir]&filter[field]=value` on list endpoints.
- **Concurrency:** mutating responses include `updated_at`; clients send `If-Unmodified-Since`
  semantics via `version` for optimistic concurrency where it matters (progress, lists).

## 1. Auth & session

| Method | Path | Purpose |
|--------|------|---------|
| `POST` | `/auth/login` | `{username,password}` → `{access, refresh, accessExpiry, user}`. Auth mode only. |
| `POST` | `/auth/refresh` | `{refresh}` → new `{access, refresh, …}` pair (refresh token rotated). |
| `POST` | `/auth/logout` | `{refresh}` → revoke that refresh-token session (204). |
| `GET`  | `/auth/handshake` | Returns the acting user: the authenticated user in auth mode, the implicit owner in embedded/auth-disabled mode. |

## 2. Server / system

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/healthz` / `/readyz` | Liveness / readiness. |
| `GET` | `/server/info` | Version, mode, capabilities, feature flags. |
| `GET` | `/server/stats` | Counts: libraries, series, books, pages, cache size. |
| `POST`| `/admin/shutdown` | Graceful shutdown (owner only; embedded client uses this). |

## 3. Libraries & scanning

| Method | Path | Purpose |
|--------|------|---------|
| `GET`   | `/libraries` | List libraries with summary counts. |
| `POST`  | `/libraries` | Create `{name, kind, roots[]}`. |
| `GET`   | `/libraries/{id}` | Detail + roots + scan status. |
| `PATCH` | `/libraries/{id}` | Update name/roots/scan options. |
| `DELETE`| `/libraries/{id}` | Remove from catalog (files untouched). |
| `POST`  | `/libraries/{id}/scan` | Start scan `{mode: full|incremental}` → `{jobId}`. |
| `POST`  | `/libraries/{id}/scan/cancel` | Cancel running scan. |
| `GET`   | `/libraries/{id}/health` | Orphans, corrupt files, unmatched, duplicates. |

## 4. Browse: series & books

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/series` | List/filter series (`?library=&sort=&filter[...]`). |
| `GET` | `/series/{id}` | Series detail + book list + aggregate progress. |
| `PATCH`| `/series/{id}` | Edit series metadata (sets `locked`). |
| `GET` | `/series/{id}/cover` | Series cover image (content-addressed). |
| `GET` | `/books` | List/filter books across libraries. |
| `GET` | `/books/{id}` | Book detail: metadata, pages summary, your progress. |
| `PATCH`| `/books/{id}` | Edit book metadata. |
| `GET` | `/books/{id}/cover?w=` | Cover thumbnail at requested width. |
| `GET` | `/books/{id}/file` | Download original file (range supported). |

### Browse response example — `GET /books/{id}`

```jsonc
{
  "id": "01J…",
  "series": { "id": "01J…", "name": "Saga", "readingDir": "ltr" },
  "title": "Chapter One",
  "number": "1",
  "volume": 1,
  "pageCount": 44,
  "releaseDate": "2012-03-14",
  "ageRating": "Mature",
  "format": "cbz",
  "people": [{ "role": "writer", "name": "Brian K. Vaughan" }],
  "genres": ["Sci-Fi", "Fantasy"],
  "tags": ["Image"],
  "metadataState": "matched",
  "progress": { "page": 12, "status": "in_progress", "percent": 27.3, "updatedAt": "…" },
  "covers": { "thumb": "/api/v1/books/01J…/cover?w=300", "full": "/api/v1/books/01J…/pages/0" }
}
```

## 5. Reading the book (the reader's endpoints)

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/books/{id}/manifest` | Ordered page list with dims, types, double-spread flags, reading dir. The reader's source of truth. |
| `GET` | `/books/{id}/pages/{idx}` | Full page image. Supports `?w=&fit=&fmt=webp\|avif\|jpeg&q=` for server-side resize/transcode. Range + ETag + immutable cache. |
| `GET` | `/books/{id}/pages/{idx}/thumb` | Tiny page thumbnail (page strip / scrubber). |
| `POST`| `/books/{id}/prefetch` | `{from, count}` hint → server warms page cache. |

### `GET /books/{id}/manifest`

```jsonc
{
  "bookId": "01J…",
  "pageCount": 44,
  "readingDir": "ltr",
  "pages": [
    { "idx": 0, "w": 1988, "h": 3056, "type": "FrontCover", "double": false },
    { "idx": 1, "w": 1988, "h": 3056, "type": "Story", "double": false },
    { "idx": 14, "w": 3976, "h": 3056, "type": "Story", "double": true }
  ]
}
```

## 6. Progress & history

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/me/continue` | "Continue Reading" rail: in-progress books, recency-ranked. |
| `GET` | `/me/progress/{bookId}` | Your progress for one book. |
| `PUT` | `/me/progress/{bookId}` | Upsert `{page, status?, device?}`. Idempotent; last-writer-wins by `updatedAt`. Also broadcast over WS. |
| `POST`| `/me/progress/batch` | Bulk upsert (reader flushes offline progress here). |
| `POST`| `/me/books/{id}/mark` | `{status: read\|unread}` convenience. |
| `GET` | `/me/history` | Reading history feed. |
| `GET` | `/me/stats` | Aggregate stats (books read, pages, streaks, by month/genre). |

Progress writes are **debounced & batched** by the reader (e.g. every N page turns or on
idle/blur) and reconciled by `updatedAt` + `device`.

## 7. Collections, reading lists, smart lists

| Method | Path | Purpose |
|--------|------|---------|
| `GET/POST` | `/collections` | List / create collections. |
| `GET/PATCH/DELETE` | `/collections/{id}` | Manage a collection. |
| `POST` | `/collections/{id}/items` | Add books `{bookIds[]}`. |
| `PATCH`| `/collections/{id}/items/reorder` | `{bookId, beforeId?}` → fractional reposition. |
| `DELETE`| `/collections/{id}/items/{bookId}` | Remove. |
| `GET/POST` | `/me/reading-lists` | Personal lists. |
| `…` | `/me/reading-lists/{id}/items…` | Same item ops, per-user. |
| `GET/POST` | `/smart-lists` | List / create rule-based lists. |
| `GET` | `/smart-lists/{id}/results` | Evaluate + return matching books (paginated). |

## 8. Search & discovery

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/search?q=&type=all\|series\|book\|person` | FTS5 search, grouped results. |
| `GET` | `/search/suggest?q=` | Type-ahead suggestions. |
| `GET` | `/discover` | Home feed: Continue Reading, Recently Added, On Deck (next unread in a series you're reading), New Series, Random. |

## 9. Metadata providers

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/providers` | Configured providers + auth status. |
| `POST`| `/books/{id}/match` | Trigger metadata match `{provider?, query?}` → `{jobId}` or candidates. |
| `GET` | `/books/{id}/match/candidates` | Ranked provider candidates for manual pick. |
| `POST`| `/books/{id}/match/apply` | Apply chosen candidate `{provider, providerId, fields[]}`. |
| `POST`| `/series/{id}/match…` | Same at series granularity. |
| `POST`| `/books/{id}/metadata/write-sidecar` | Write `ComicInfo.xml` back into the archive (opt-in). |

## 10. WebSocket — `/api/v1/ws`

Single multiplexed socket; client subscribes to topics. JSON frames:
`{ "type": "<event>", "topic": "<topic>", "data": {…} }`.

| Topic | Events | Use |
|-------|--------|-----|
| `jobs` | `job.progress`, `job.done`, `job.failed` | Scan/thumbnail/match progress bars. |
| `library` | `book.added`, `book.updated`, `book.removed`, `series.updated` | Live catalog updates → invalidate query cache. |
| `progress` | `progress.updated` | Cross-device "now reading" sync (reader ↔ client). |
| `presence` | `device.reading` | Optional: show what's being read where. |

Client→server frames: `subscribe {topics[]}`, `unsubscribe`, `ping`. Server heartbeats
every 30s; clients reconnect with exponential backoff and re-subscribe.

## 11. Image delivery rules

- Page/thumb/cover URLs are **content-addressed** (path or query includes a hash/version);
  responses set `Cache-Control: public, max-age=31536000, immutable` + strong `ETag`.
- Server-side resize params (`w`, `fit`, `fmt`, `q`) are part of the cache key.
- Negotiation: if client sends `Accept: image/avif`, server may serve AVIF; otherwise WebP,
  falling back to JPEG. The reader requests an explicit `fmt` to stay deterministic.
- Range requests supported for large images and original-file downloads.

## 12. Versioning & compatibility

- Path-versioned (`/api/v1`). Additive changes are non-breaking; breaking changes bump the
  version. `/server/info.capabilities` lets clients feature-detect (e.g. `avif`, `pdf`,
  `epub`, `multiuser`) so one client build works against many server versions.
