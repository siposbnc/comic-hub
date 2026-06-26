# 05 — Client (Tauri + React)

The client is the management surface: browse, organize, search, edit metadata, manage
libraries, see stats. It also bundles and supervises the local server (embedded mode).

## 1. Tech choices

| Concern | Choice | Why |
|---------|--------|-----|
| Shell | Tauri 2 (Rust) | Tiny binary, low memory, native OS integration, mobile-later path. |
| UI | React 18 + TypeScript | Expressive, huge ecosystem for complex UI. |
| Build | Vite | Fast HMR + builds. |
| Server cache/state | TanStack Query | Caching, invalidation, optimistic updates, infinite lists. |
| UI/local state | Zustand | Lightweight global UI state (reader prefs, selection, layout). |
| Routing | TanStack Router (typed) | Type-safe routes + search params. |
| Virtualization | TanStack Virtual | 60fps grids over 10k+ covers. |
| Styling | Tailwind + design tokens (see [07](07-design-system.md)) | Fast, consistent, themable. |
| Forms | React Hook Form + Zod | Validated metadata editing. |
| i18n | `i18next` | Localization-ready from day one. |

## 2. Responsibilities split (Rust vs React)

**Rust (thin host):**
- Spawn/locate/health-check the sidecar server; own the handshake + token.
- Store the auth token in the OS keychain (Tauri stronghold/keyring).
- Native dialogs (folder picker for adding libraries), tray icon, notifications.
- Auto-update; deep-link + file-association registration (shared with reader).
- Launch the reader (spawn process / deep link).

**React (everything else):** all screens, data fetching, state, design.

IPC: a small, typed set of Tauri `commands` (`get_connection`, `pick_folder`,
`open_reader(bookId)`, `restart_server`, `get_token`) and `events`
(`server-status`, `deep-link`).

## 3. Information architecture / navigation

```
┌ Sidebar ───────────────┐ ┌ Main ───────────────────────────────┐
│ Home                    │ │  (route content)                     │
│ Libraries               │ │                                      │
│  • DC                   │ │                                      │
│  • Manga                │ │                                      │
│ Reading Lists           │ │                                      │
│ Collections             │ │                                      │
│ Smart Lists             │ │                                      │
│ ─────────               │ │                                      │
│ Stats                   │ │                                      │
│ Settings                │ │                                      │
└─────────────────────────┘ └──────────────────────────────────────┘
   Top bar: global search · scan/job status · user switcher · view controls
```

### Routes

| Route | Screen |
|-------|--------|
| `/` | **Home / Discover** — Continue Reading, On Deck, Recently Added, New Series, Random. |
| `/library/:id` | Series grid for a library, with filter/sort/group bar. |
| `/series/:id` | Series detail — header (cover, metadata, aggregate progress), issue grid/list. |
| `/book/:id` | Book detail — metadata, page strip, "Read"/"Continue" CTA, edit, history. |
| `/lists/reading/:id` | A personal reading list (ordered, drag-reorder). |
| `/collections/:id` | A curated collection. |
| `/smart/:id` | Smart list results (live query). |
| `/search?q=` | Grouped search results. |
| `/stats` | Reading stats dashboards. |
| `/settings/*` | Libraries, providers, users, appearance, server, about. |

## 4. Key screens

### 4.1 Home / Discover
- **Continue Reading** rail (most important): horizontally-scrolling cards with progress
  bars; one click → reader at the right page.
- **On Deck:** next unread issue in series you're actively reading.
- **Recently Added**, **New Series**, **Random pick**, per-library "Jump back in".
- Each card: cover (blur-up), title, series, progress overlay; primary action = read.

### 4.2 Library grid
- Virtualized cover grid; adjustable cover size; group by series; sort (name, added, year,
  unread count); filter (status, genre, tag, publisher, year, rating, format, reading dir).
- Multi-select for bulk actions (mark read/unread, add to list/collection, rematch metadata).
- Series cards show unread badge + read-progress ring.

### 4.3 Series detail
- Hero header: large cover, title/year/publisher, summary, aggregate progress
  ("12 of 30 read"), actions (read next, mark all, edit, rematch).
- Issues as grid or table (number, title, date, your status). Inline status toggles.

### 4.4 Book detail
- Cover + metadata panel (people, genres, tags, characters, file info).
- Page strip (thumbnails) with cover/story/ad markers.
- CTA: **Read** / **Continue at p.N**. Secondary: download file, edit metadata, match,
  add to list/collection, write sidecar.
- Reading history for this book.

### 4.5 Metadata editor
- Form (React Hook Form + Zod) over book/series metadata; "Match online" opens a candidate
  picker (provider results with covers + confidence); field-level lock toggles; bulk-edit
  mode for multi-select.

### 4.6 Settings
- **Libraries:** add/remove roots (native folder picker), scan now, watch toggle, naming rules.
- **Providers:** enable, enter API keys, test connection.
- **Users** (auth mode): create users, roles, age-rating ceilings.
- **Server:** mode (embedded/connect to remote URL), status, logs tail, restart.
- **Appearance:** theme (system/light/dark), accent, cover size defaults, reader defaults.

## 5. Data layer & realtime

- **TanStack Query** wraps the REST API with typed hooks (`useSeries`, `useBook`,
  `useContinueReading`, …). Query keys mirror resource identity.
- **WebSocket** events map to **cache invalidations**: `book.updated` → invalidate that
  book + its series; `job.progress` → update a jobs store; `progress.updated` → update
  Continue Reading + the book's progress (so finishing in the reader instantly reflects here).
- **Optimistic updates** for progress marks, list reordering, metadata edits; rollback on error.
- **Offline/disconnected UX:** if the server is unreachable, show a clear banner and retry;
  cached views remain visible (TanStack Query persistence to disk optional).

## 6. Server supervision UX (embedded)

- On first launch with no server configured, the client spawns the sidecar and shows a
  one-time "Setting up your library" welcome → "Add your first library" flow.
- A status pill in the top bar reflects server state (running/scanning/error). Scan progress
  is a non-blocking inline indicator with a popover (per-library job list, cancel).
- If the user switches to a **remote server** in Settings, the client stops the local sidecar
  and connects out; all screens are identical.

## 7. Performance practices

- Virtualize every long list/grid; lazy-load covers with `loading=lazy` + blur-up LQIP.
- Request right-sized covers (`?w=` matching the rendered size & DPR).
- Prefetch series detail on card hover/focus; prefetch the reader manifest + first pages on
  "Read" hover so the reader opens instantly.
- Code-split routes; keep the shell lightweight.

## 8. Accessibility & input

- Full keyboard navigation (grid arrow-key traversal, `/` to search, `Enter` to open,
  `R` to read). Visible focus rings. ARIA roles on grids/lists.
- Respect `prefers-reduced-motion` and `prefers-color-scheme`.
- Hit targets and spacing tuned for desktop now, touch-friendly for the mobile-later path.

## 9. Packaging & updates

- Tauri bundles the client + the `comichub-server` sidecar binary (per-OS) + libvips/MuPDF
  runtime deps.
- Signed installers (MSI/NSIS on Windows). Tauri auto-updater checks a release feed; server
  sidecar is versioned with the client and updated together (API is version-negotiated so a
  mismatched remote server still works).
