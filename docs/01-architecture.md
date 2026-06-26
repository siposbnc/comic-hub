# 01 — Architecture

## 1. System overview

Three deployable artifacts, one shared contract:

| Artifact | Tech | Role |
|----------|------|------|
| `comichub-server` | Go, single static binary | Owns library, DB, files, jobs, API. |
| `comichub` (client) | Tauri + React | Browse & manage. Bundles the server as a sidecar. |
| `comichub-reader` | Tauri + React | Read a book. Works with or without a server. |

All clients talk to the server over **HTTP + WebSocket**. The reader can _also_ read a
file directly from disk with no server (standalone mode). There is exactly **one** API
surface; "local" and "remote" differ only by which host:port the client points at and
which auth mode is active.

```
┌─────────────────────────── User's PC ───────────────────────────┐
│                                                                  │
│  ┌────────────────┐         spawns          ┌────────────────┐   │
│  │ ComicHub Client│ ───────────────────────▶│ comichub-server│   │
│  │  (Tauri/React) │  127.0.0.1:<port>  HTTP  │   (sidecar)    │   │
│  └───────┬────────┘ ◀──────────────────────▶└───────┬────────┘   │
│          │ launch w/ deep link                       │           │
│          ▼                                           ▼           │
│  ┌────────────────┐    HTTP/WS (connected)    ┌────────────┐     │
│  │ ComicHub Reader│ ─────────────────────────▶│  SQLite +  │     │
│  │  (Tauri/React) │                            │  files +   │     │
│  │                │ ── direct read (offline) ─▶│  cache     │     │
│  └────────────────┘                            └────────────┘     │
└──────────────────────────────────────────────────────────────────┘

                 …or, optionally, point the client at:

┌────────────── NAS / Home server (always-on) ──────────────┐
│  comichub-server  (same binary, auth ON)  :8080           │
│  SQLite/Postgres + libraries + cache                      │
└───────────────────────────────────────────────────────────┘
```

## 2. Deployment modes

ComicHub supports three modes with **no code forks** — only configuration differs.

### 2.1 Embedded (default, zero-config)

- Client launches → checks config for a configured server → none → **spawns the bundled
  sidecar** server binary.
- Server binds to `127.0.0.1:<ephemeral-port>`, writes the chosen port + a one-time
  **loopback token** to a handshake file the client reads.
- Auth is effectively "trust loopback + token". No accounts, no passwords. Single implicit user.
- Lifecycle: client owns the process; on client quit, it sends graceful shutdown. A
  lock file prevents two clients spawning two servers over the same data dir.

### 2.2 Local shared (opt-in)

- Same as embedded, but the server binds to `0.0.0.0:<port>` and auth is turned **on**
  so other devices on the LAN can connect. The client that spawned it is just one of N clients.
- Useful for: read on the desktop, manage on the laptop, same machine hosts data.

### 2.3 Remote server (opt-in)

- User runs `comichub-server` on a NAS/home server as a service (systemd, Windows
  Service, Docker). Auth **on**, real user accounts.
- Client/reader are pointed at `https://host:8080` via settings or discovery.
- Identical binary and API; just persistent and multi-user.

> **Design rule:** the server never assumes it's local. The client never assumes it's
> embedded. The handshake/token flow is the only thing that differs, and it's isolated
> in a small "connection" module on each side.

## 3. Process & lifecycle model

### Client ↔ sidecar handshake (embedded mode)

1. Client starts. Reads `~/.comichub/connection.json`. If a working server is reachable, reuse it.
2. Otherwise, acquire `~/.comichub/server.lock` (flock). If held by a live PID, attach to its port.
3. Spawn sidecar: `comichub-server --mode=embedded --data-dir=… --handshake-fd=…`.
4. Server picks a free loopback port, generates a 256-bit token, writes
   `{port, token, pid, version}` to the handshake path, then starts serving.
5. Client reads handshake, stores token in memory, begins API calls with
   `Authorization: Bearer <token>`.
6. Health: client polls `GET /healthz`; on sidecar crash, restarts it with backoff and
   surfaces a non-blocking toast.
7. Shutdown: client calls `POST /admin/shutdown`; server finishes in-flight jobs up to a
   deadline, flushes, exits. Lock released.

### Reader launch paths

- **From file association:** OS opens `comichub-reader "D:\…\Book.cbz"`. Reader tries to
  reach a running server (via `connection.json`) to record progress; if none, it reads the
  file directly and stores progress locally (see §6.3).
- **From client:** client opens a deep link `comichub-reader://open?bookId=…&server=…&token=…`
  (or passes args). Reader connects to that server, streams pages, syncs progress live.

## 4. Internal architecture — Media Server

Layered, dependency-inverted. Detailed in [04-server.md](04-server.md).

```
            ┌─────────────────────────────────────────────┐
 HTTP/WS ──▶│  Transport      (chi router, handlers, WS hub)│
            ├─────────────────────────────────────────────┤
            │  Services       (Library, Reading, Lists,    │
            │                  Metadata, Reader, Admin)     │
            ├─────────────────────────────────────────────┤
            │  Domain         (entities, rules, value objs) │
            ├─────────────────────────────────────────────┤
            │  Adapters       (SQLite repo, archive readers,│
            │                  image pipeline, providers,    │
            │                  file watcher, job queue)      │
            └─────────────────────────────────────────────┘
```

Cross-cutting: structured logging (slog), config, metrics, a background **job runner**
(scan, thumbnail, metadata, watch), and a **content cache** (thumbnails + extracted pages).

## 5. Internal architecture — Client & Reader

Tauri (Rust) provides the window, OS integration (file associations, deep links, tray,
auto-update, secure token storage), and spawns/locates the server. React owns all UI.

```
┌──────────────── Tauri (Rust core) ─────────────────┐
│ window mgmt · file assoc · deep links · OS keychain │
│ sidecar spawn · auto-update · native file dialogs   │
└───────────────────────┬─────────────────────────────┘
                        │ Tauri IPC (commands/events)
┌───────────────────────▼─────────────────────────────┐
│                   React + TypeScript                 │
│  Router · TanStack Query (server cache) · Zustand    │
│  (UI state) · Design system · Virtualized grids      │
└──────────────────────────────────────────────────────┘
```

Rust is kept thin — it's a host/bridge, not where features live. Detail in
[05-client.md](05-client.md) and [06-reader.md](06-reader.md).

## 6. Data & storage

### 6.1 Server data directory (`--data-dir`)

```
comichub/
  comichub.db          SQLite (WAL mode) — catalog, progress, lists, users
  config.toml          server config
  cache/
    thumbs/<hash>/…     generated thumbnails (sharded by hash prefix)
    pages/<hash>/…      extracted/transcoded page cache (LRU-evicted)
  covers/<hash>.webp    series/book cover art
  logs/
```

- **SQLite** is the default store (embedded, zero-admin, fast for this read-heavy
  workload). A **Postgres** option exists for large multi-user remote installs (same
  schema via a repo interface). Choice is config; the domain doesn't care.
- The **library files themselves are never moved or modified** by default. ComicHub is a
  catalog over your files. An opt-in "organize" feature can rename/move on explicit action.

### 6.2 Caches

- Thumbnails and resized pages are derived data — safe to delete; regenerated on demand.
- Page cache is LRU with a configurable size cap. Thumbnails persist (cheap, high reuse).

### 6.3 Reader standalone storage

When reading offline (no server), the reader keeps a tiny local store
(`~/.comichub/reader.db`) of progress keyed by a **content hash** of the file. When a
server later becomes available, the reader **reconciles** local progress into the server
(by content hash → book id), so reading a loose file and later importing it doesn't lose progress.

## 7. Networking & transport

- **HTTP/1.1** for API + page/image streaming (range requests supported for large pages).
- **WebSocket** for push: scan progress, job status, live "now reading" sync, library
  change events. One multiplexed socket per client with topic subscriptions.
- Compression: images are already compressed; JSON responses use gzip. ETags +
  `Cache-Control` on all immutable image assets (content-addressed → cache forever).
- Remote installs SHOULD sit behind TLS (reverse proxy or built-in cert). Local loopback
  is plain HTTP (never leaves the machine).

## 8. Security model

| Mode | AuthN | AuthZ | Transport |
|------|-------|-------|-----------|
| Embedded | Loopback + bearer token (handshake) | Single implicit owner | Plain HTTP on 127.0.0.1 |
| Local shared | Bearer token / account login | Per-user roles | Plain HTTP on LAN (warned) |
| Remote | Account login → JWT (access + refresh) | Roles: owner/admin/member/restricted | TLS required |

- **Roles:** `owner` (full), `admin` (manage libraries/users), `member` (read + own
  progress/lists), `restricted` (member + content rating ceiling, no settings).
- Tokens stored in the OS keychain via Tauri (never in plain files on the client).
- The server validates and sandboxes all file paths to configured library roots
  (no path traversal). Archive extraction is bounded (zip-bomb guards: entry count, total
  uncompressed size, per-entry size, nesting depth).
- Metadata provider API keys live server-side only; never shipped to clients.

## 9. Performance strategy

- **Catalog reads are SQLite + index hits**, served from RAM cache for hot views.
- **Thumbnails precomputed** during scan; grids stream content-addressed WebP/AVIF thumbs.
- **Reader prefetch:** server warms the next N pages into the page cache; client prefetches
  next/prev page blobs into memory. Target: page turn = already-decoded bitmap swap.
- **Scanning is incremental & parallel:** worker pool sized to CPU; only changed files
  (mtime+size, then content hash) are reprocessed. Resumable across restarts.
- **Virtualized UI:** grids/lists render only visible items; covers lazy-load with
  blur-up placeholders.

## 10. Observability

- Structured JSON logs (slog) with levels; client surfaces server log tail in a debug panel.
- `/healthz` (liveness) and `/readyz` (DB + cache ready).
- Optional Prometheus `/metrics` (scan throughput, cache hit rate, request latency, job queue depth).
- A "Library Health" view in the client: orphaned files, unmatched metadata, corrupt
  archives, duplicate detection.

## 11. Extensibility seams

- **Metadata providers** behind a `Provider` interface — add Comic Vine, GCD, Metron,
  AniList without touching the matching engine.
- **Format readers** behind an `ArchiveReader`/`PageSource` interface — add EPUB later.
- **Storage** behind a `Repository` interface — SQLite ↔ Postgres.
- **Clients** are just API consumers — a future mobile app or web client needs no server change.
