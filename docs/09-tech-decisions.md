# 09 — Tech Decisions (ADRs)

Short architecture decision records: the choice, the context, the rejected alternatives, and
the consequences. These capture _why_, so future changes are deliberate.

---

## ADR-001 — Three artifacts, one API (server / client / reader)

**Decision:** Ship a Go media server, a Tauri client, and a separate Tauri reader, all sharing
one HTTP/WS API.

**Context:** The brief requires a media server backend, a thin client, and a reader that can
open `.cbz` files directly (file association) _or_ be launched from the client.

**Alternatives rejected:**

- _Monolith (one app does everything):_ can't satisfy instant, server-independent double-click
  reading; couples reader updates to the whole app; heavier to launch.
- _Reader as a window inside the client:_ opening a loose file would boot the entire client and
  ideally a server — too heavy for "one-click read," fragile when no server is running.

**Consequences:** Clean failure isolation and independent updates; a tiny bit of duplicated
shell setup, mitigated by a shared TS package for the design system + server `PageProvider`.

---

## ADR-002 — Go for the media server

**Decision:** Go (1.22+), chi router, sqlc, libvips via govips.

**Context:** The hard work is systems-shaped (scan thousands of files, parse archives, generate
images, serve concurrent streams, run background jobs). Performance is a stated priority, and
the team must implement complex features _confidently_.

**Alternatives rejected:**

- _Rust:_ best raw numbers but the slowest place to deliver evolving, complex features
  confidently — the feature gap would hurt more than microseconds.
- _Node/TS:_ fastest to write and shares types with the frontend, but CPU-bound image/archive
  work forces native addons and a long-running media server stresses the single-threaded model.
- _.NET:_ strong on Windows but heavier to deploy as a drop-anywhere single binary for the NAS
  future.

**Consequences:** Single static binary (great for sidecar + NAS), excellent concurrency for the
scanner/job system, simple deployment. CGo needed for libvips/MuPDF — mitigated with prebuilt
binaries and a `nocgo` fallback build.

---

## ADR-003 — Tauri + React + TypeScript for client & reader

**Decision:** Tauri 2 shell, React 18 + TypeScript, Vite, TanStack Query/Router/Virtual, Zustand.

**Context:** UX-heavy surfaces (reader gestures, virtualized 10k-cover grids, metadata editing).
Windows-first with a mobile-later requirement; performance emphasis.

**Alternatives rejected:**

- _Electron:_ most mature but larger memory/disk footprint and no first-class mobile path.
- _Flutter / .NET MAUI:_ viable, but pull us out of the richest web-UI ecosystem where the
  complex reader/library UI is fastest and best to build.

**Consequences:** Tiny binaries, low memory, native OS integration (file assoc, deep links,
keychain), and a credible mobile-later story. Rust is kept thin (host/bridge only).

---

## ADR-004 — Local-first, server-optional with the same binary

**Decision:** One server binary runs three ways — embedded (sidecar), local-shared, remote —
differing only by config (bind address + auth mode). The client bundles and supervises the
sidecar; it can instead point at a remote URL.

**Context:** The user chose "local-first, optional server." We must work fully offline on one PC
yet scale to an always-on NAS without a rewrite.

**Alternatives rejected:**

- _Embedded-only:_ no remote streaming to other devices — fails the "optional server" goal.
- _Server-required:_ heavier first-run; breaks dependency-free double-click reading.

**Consequences:** A small handshake/connection module is the _only_ place that differs per mode;
the server never assumes it's local, the client never assumes it's embedded. Slightly more care
needed in auth (token in loopback vs JWT remote), isolated behind one boundary.

---

## ADR-005 — SQLite default, Postgres optional

**Decision:** SQLite (WAL) embedded by default; Postgres behind the same `Repository` interface
for large multi-user remote installs.

**Context:** Read-heavy catalog workload; zero-admin local experience is paramount; remote
installs may want concurrency/scale.

**Consequences:** No DB to install locally; fast indexed reads; FTS5 for search. Postgres path
exists when needed without touching domain code. Care to avoid SQLite-only SQL in shared queries.

---

## ADR-006 — Catalog over files; never mutate by default

**Decision:** ComicHub is a catalog _over_ the user's files. It does not move/rename/repack
files unless the user explicitly runs the opt-in "Organize" action (dry-run first).

**Context:** Collectors are protective of their carefully-arranged files and metadata.

**Consequences:** Safe, trustworthy by default; deletions on disk mark books `missing` (progress
and lists survive) rather than cascading. Write-back of `ComicInfo.xml` and organize are explicit,
previewed, reversible operations.

---

## ADR-007 — libvips for the image pipeline

**Decision:** Generate thumbnails at scan time and serve resized/transcoded pages via libvips
(govips), content-addressed and cached; WebP default, AVIF optional, JPEG fallback.

**Context:** Image work is the hottest path for both grids and the reader; performance is a
feature.

**Alternatives rejected:** Pure-Go image libs (slower, more memory for decode+resize); per-request
decode without caching (wasteful, janky).

**Consequences:** Fast, low-memory thumbnailing; instant grids; deterministic cache keys. Adds a
CGo dependency (see ADR-002 mitigation).

---

## ADR-008 — Progress reconciliation by content hash, last-writer-wins

**Decision:** Reading progress is keyed by `(user, book)`; the reader debounces writes and, when
offline/standalone, stores progress locally keyed by file `content_hash`, reconciling into the
server when one is available. Conflicts resolve last-writer-wins by `updatedAt` (+ device).

**Context:** A loose file read standalone today might be imported tomorrow; the same book may be
read on two devices.

**Consequences:** "Never lose your place," even across standalone→server and multi-device. Hash
keying also powers dedup and move/rename reconciliation. Edge conflicts surface a gentle
"resume here / there?" prompt rather than silently clobbering.

---

## ADR-009 — In-process job queue (no external broker)

**Decision:** A SQLite-backed queue + in-process Go worker pool runs scans, thumbnails, matches,
watches, etc. Priorities let interactive work preempt bulk.

**Context:** Local-first means no Redis/RabbitMQ to install; jobs must survive restarts and report
progress.

**Consequences:** Zero extra infrastructure; resumable, cancelable, observable jobs over WS. For
very large remote installs this is still sufficient; if it ever isn't, the `jobs` package is a
clean seam to swap.

---

## ADR-010 — Hosted GCD as the default, zero-config metadata provider

**Decision:** Stand up a ComicHub-operated metadata service backed by a slimmed copy of the
**Grand Comics Database** (GCD, comics.org) data dump, exposed through a thin read-only REST API
that conforms to the existing `providers.Provider` interface. Ship it **default-on with no user
credentials**. Keep Comic Vine and Metron as **optional enrichment** providers (the matcher
already merges and ranks candidates across all configured providers).

**Context:** Today metadata works only after a user obtains a Comic Vine API key or a Metron
account — a real onboarding wall. GCD is the deepest bibliographic index in existence (≈228k
series, ≈2.2M issues, ≈4.5M stories, ≈109k creators) and the strongest source for series/issue/
publisher/date/credits/genres. Its dump is CC BY-SA 4.0 and regenerated bi-weekly. Self-hosting
it centrally turns "get an API key first" into "metadata just works."

**Alternatives rejected:**

- _Local per-server import_ of the dump (8–15 GB + multi-hour imports + bi-weekly refresh on
  every user's machine) — far too heavy for a desktop-first app.
- _GCD-only, removing CV/Metron_ — loses normalized characters (GCD stores them as free text),
  clean story-arc entities, and provider cover thumbnails. CV/Metron cost nothing (user keys) and
  fill exactly these gaps, so they stay optional.
- _Scraping comics.org_ — fragile and against the site's spirit; the dump is the sanctioned path.

**Consequences:**

- Zero-config metadata for everyone; CV/Metron become opt-in "extra character/arc data."
- Introduces ComicHub-operated infrastructure: a slim Postgres (~5–8 GB) + thin API + a
  cron that downloads the dump → loads a staging DB → atomic swap. Est. **~$10/mo** DIY VPS to
  **~$25/mo** managed; reads are static between refreshes and **highly cacheable**, so DB load
  stays low as usage grows. This mirrors the existing "call an external provider API" model — it
  is not a new architectural dependency class.
- **Attribution required** (CC BY-SA 4.0); surface a GCD credit in the UI/docs. **Do not** serve
  GCD cover _images_ (separate, unclear image rights) — ComicHub already draws covers from the
  comic files themselves; provider cover URLs only feed the match-picker thumbnail.
- Character/story-arc coverage from GCD is weaker; those tabs stay best-served by an enrichment
  provider until/unless GCD free-text parsing proves good enough.

See the implementation plan in [phase-2-plan.md](phase-2-plan.md#deferred-metadata-polish-backlog-future-updates)
(the GCD backlog item) for the phased build.

---

## Open questions / to revisit

- **Product name** — "ComicHub" is provisional; revisit before public release (trademark check).
- **AVIF cost/benefit** at scale vs WebP — measure CPU during scans before enabling by default.
- **CBR longevity** — RAR is read-only and a fading format; consider nudging users to convert
  (opt-in) in Phase 4.
- **Mobile shell** — Tauri mobile maturity to be re-evaluated at Phase 4; React Native is the
  fallback for the reader.
- **PDF reliability** — MuPDF licensing/CGo footprint; validate the build-tag split early.
