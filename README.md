# ComicHub

> A fully-featured library platform for your comics — think **Plex, but for comics**.

ComicHub manages your local collection of comics, tracks what you're reading, lets you
build reading lists, and ships with a fast, comfortable reader you can launch in one click.

## What it is

- **Media server** — a single Go binary that owns your library: scanning, metadata,
  thumbnails, page streaming, progress tracking, and background jobs.
- **Client** — a Tauri + React desktop app (Windows first, mobile later) for browsing
  and managing your collection.
- **Reader** — a separate, lightweight Tauri app. Double-click a `.cbz` to read it
  standalone, or launch it from the client with full progress sync.

## Design pillars

1. **Local-first.** Everything works offline on one machine out of the box. The server
   can _optionally_ run on a separate always-on host (NAS / home server) for remote
   access — same binary, no rewrite.
2. **Fast.** Instant library browsing, sub-100ms page turns, aggressive prefetching.
3. **Comfortable.** The reader is a first-class product, not an afterthought.
4. **Open formats.** CBZ / CBR / CB7 / CBT / PDF, `ComicInfo.xml` metadata, no lock-in.

## Documentation

| Doc | Contents |
|-----|----------|
| [00 — Overview](docs/00-overview.md) | Vision, goals, personas, glossary, non-goals |
| [01 — Architecture](docs/01-architecture.md) | System architecture, deployment model, process model, security |
| [02 — Data Model](docs/02-data-model.md) | Entities, SQLite schema, relationships |
| [03 — API](docs/03-api.md) | REST + WebSocket contracts |
| [04 — Media Server](docs/04-server.md) | Scanner, formats, image pipeline, metadata, jobs |
| [05 — Client](docs/05-client.md) | Screens, navigation, state, IPC |
| [06 — Reader](docs/06-reader.md) | Standalone vs connected, rendering, modes, prefetch |
| [07 — Design System](docs/07-design-system.md) | Visual language, tokens, components |
| [08 — Roadmap](docs/08-roadmap.md) | Phased delivery plan, MVP definition |
| [09 — Tech Decisions](docs/09-tech-decisions.md) | ADRs and rationale |

## Tech stack at a glance

| Layer | Choice |
|-------|--------|
| Media server | Go (stdlib `net/http` + chi router), SQLite, libvips (govips) |
| Client / Reader shell | Tauri 2 (Rust wrapper) |
| Client / Reader UI | React + TypeScript + Vite |
| Client state | TanStack Query + Zustand |
| Transport | HTTP/1.1 + WebSocket (local loopback or LAN/WAN) |
| Packaging | Server bundled as Tauri sidecar; standalone server binary for remote |
