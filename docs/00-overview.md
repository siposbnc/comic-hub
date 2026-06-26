# 00 — Overview

## 1. Vision

ComicHub is a **library platform** for comics, not merely a reader. It does for comics
what Plex/Jellyfin do for video: it ingests a messy folder of files, turns it into a
clean, browsable, richly-tagged library, tracks your reading across devices, and gives
you a great way to actually read.

The product is three cooperating pieces:

```
        ┌──────────────────────┐
        │   ComicHub Client    │  Browse, manage, organize
        │  (Tauri + React)     │
        └──────────┬───────────┘
                   │ HTTP / WS
        ┌──────────▼───────────┐
        │  ComicHub Server     │  Owns the library + DB + files
        │     (Go binary)      │
        └──────────┬───────────┘
                   │ HTTP / WS (or direct file read)
        ┌──────────▼───────────┐
        │  ComicHub Reader     │  Read a book, sync progress
        │  (Tauri + React)     │
        └──────────────────────┘
```

## 2. Goals

- **G1 — One-click reading.** From "I see a comic" to "I'm reading it" in a single action.
- **G2 — Effortless library.** Point at a folder; get a clean, deduplicated, metadata-rich
  library without manual data entry.
- **G3 — Real tracking.** Per-page progress, read/unread state, history, "continue reading",
  and completion stats that are always correct.
- **G4 — Organization that scales.** Series, collections, smart (rule-based) lists, tags,
  reading lists for tens of thousands of issues.
- **G5 — Performance as a feature.** Browsing and reading feel instant.
- **G6 — Local-first, server-optional.** Works fully offline on one PC; scales to a
  remote always-on server with the same binary.
- **G7 — Open & portable.** Standard formats, standard metadata, easy export, no lock-in.

## 3. Non-goals (for now)

- **NG1 — Acquisition / piracy tooling.** No torrent clients, no download grabbers. ComicHub
  manages files you already have.
- **NG2 — DRM'd store formats.** No Comixology/Kindle DRM. Open archives + PDF only.
- **NG3 — Editing/lettering tools.** Not a creation tool.
- **NG4 — Public multi-tenant SaaS.** It's self-hosted/personal; multi-user means
  _your household_, not the internet.
- **NG5 — Mobile apps in v1.** Architecture must not preclude them; we just don't ship them first.

## 4. Personas

- **The Collector (primary).** Has 5k–80k issues across nested folders, mixed quality of
  filenames, some with `ComicInfo.xml`, some not. Wants order, dedup, and good metadata.
- **The Reader.** Mostly wants "continue where I left off" and a comfortable reading
  experience with sane defaults.
- **The Householder.** Runs the server on an always-on box; spouse/kids each have their
  own progress and (optionally) content restrictions.
- **The Tinkerer.** Wants an API, file-system transparency, and config they can version.

## 5. Key concepts & glossary

| Term | Meaning |
|------|---------|
| **Library** | A named root folder (or set of roots) ComicHub scans. e.g. "Manga", "DC". |
| **Series** | A grouping of issues that belong together (e.g. _Saga_). Usually a folder. |
| **Volume** | A numbered run/version of a series, usually distinguished by relaunch year (e.g. _Wonder Woman_ Vol. 1 (1942) vs Vol. 2 (1987)). Groups issues within a series and disambiguates renumbered runs. May also denote a collected edition where publishers use it that way. |
| **Issue / Book** | A single comic file — one `.cbz`/`.cbr`/`.pdf`. The atomic readable unit. |
| **Page** | One image inside a book. |
| **Collection** | A manually-curated, ordered grouping (e.g. "Crisis on Infinite Earths crossover"). |
| **Reading List** | A user's personal ordered queue ("To Read"). |
| **Smart List** | A saved query (rule-based) that auto-populates (e.g. "Unread DC 2024"). |
| **Tag** | A free-form label (genre, character, mood). |
| **Progress** | Per-user, per-book reading state: current page, %, status, timestamps. |
| **Metadata** | Structured info about a book/series: title, number, writer, date, etc. |
| **Sidecar** | `ComicInfo.xml` embedded in the archive — the de-facto metadata standard. |
| **Provider** | An external metadata source (Comic Vine, GCD, Metron, AniList). |
| **Sidecar binary** | The Go server shipped _inside_ the client for local-first mode. |

## 6. Format support

| Format | Container | Read | Metadata | Notes |
|--------|-----------|------|----------|-------|
| CBZ | ZIP | ✅ | `ComicInfo.xml` | Primary, fastest. |
| CBR | RAR | ✅ | `ComicInfo.xml` | Read-only (no RAR writing). |
| CB7 | 7z | ✅ | `ComicInfo.xml` | |
| CBT | TAR | ✅ | `ComicInfo.xml` | Rare but cheap to support. |
| PDF | PDF | ✅ | PDF info dict | Rasterized per-page; slower. |
| EPUB | ZIP/XHTML | ⚠️ later | OPF | Fixed-layout comics only; phase 3+. |

Page images inside archives: JPEG, PNG, WebP, AVIF, GIF, BMP. Natural sort order by
filename determines page order unless `ComicInfo.xml` specifies otherwise.

## 7. Success criteria (MVP)

- Scan a 10k-issue library in a reasonable, observable, resumable way.
- Browse that library with **<100ms** perceived interaction latency on cached views.
- Open any supported book and turn pages with **no visible load** for the next page.
- Reading progress is never lost and always reflected in "Continue Reading".
- Double-clicking a loose `.cbz` opens the reader with **no server required**.
