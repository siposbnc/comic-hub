# Handoff: Tracker screen

A dense, spreadsheet-style **reading matrix** — every tracked series as a row, every issue as a
clickable cell — so you can see completion across the whole collection at a glance and toggle
read/unread by clicking issue numbers. Think of the collector's wall-chart: rows of series,
columns of issue numbers, a wall of "done" you can read in one sweep.

## Where this lives (read first)

The Tracker is a **new top-level screen**. This doc is the design spec; the visual source of truth
is a preview screen to be built in the **`Design Preview v2`** project (`ef2d1724…`), matching the
existing `design_handoff_<feature>/` handoffs. Once approved, that project will hold:

- **`design_handoff_tracker/README.md`** — a copy of this spec.
- **`design_handoff_tracker/ClientPreview.jsx`** — the working prototype (built from this spec).
  Pieces to author: `TRACKER` mock data + a `TR(...)` builder, `TrackerScreen`, `TrackerGrid`,
  `TrackRow`, `IssueCell`, `TrackerToolbar`, `CellPopover`; routing wired in the shell (`tracker`
  route, a `Tracker` sidebar item under **System**).
- **`ComicHub Preview Screens.dc.html`** — a new canvas frame **· Tracker** mounting the above.

The design system (`_ds/…`, namespace `ComicHubDesignSystem_c0e1bf`) supplies the components and
tokens; this screen composes them. **No new colors.** Build with the grain: covers loud, chrome
quiet, cyan for progress, magenta reserved for "unread / new".

This is a **design reference** (HTML/React prototype with mock data), not production code — the
real client recreates it with TanStack routing/state and the DS components.

---

## The concept

A **frozen-panes grid**, exactly like the reference spreadsheet:

- **Rows = tracks.** A "track" is a tracked series. It either **links** to a ComicHub library
  `Series` (auto-added on scan) or is a **standalone** track the user created for a series they
  don't have in any library. Named sub-runs the collector keeps separate — Annuals, "Futures
  End", "Endgame", one-shots — are their **own track rows**, exactly as the sheet splits
  "Action Comics Vol. 2" / "…Annual" / "…Futures End".
- **Columns = an issue ruler** — the sorted union of every issue number across the visible tracks,
  sticky across the top, horizontally scrollable. Point issues (`#23.1`, `#23.2`) are **ordinary
  issues with their own column** (`… #23 #23.1 #23.2 #24 …`), not attached to a base; named
  specials get their own rows.
- **Cells = issues.** Click a cell to toggle read/unread. The fill tells the whole story at a
  glance: a wall of cyan is your progress; dim hollow cells are gaps you don't own yet.

**The Tracker owns its own data.** The library is an _input_, not the source: library series and
issues are auto-added, linked and tracked, but the user can also add issues to a track (issues
they read elsewhere, or that exist but aren't in a library) and add whole tracks with no library
series behind them. Read state on a tracked issue is independent of owning the file — you can mark
`#7` read even if ComicHub has no `.cbz` for it.

## Placement

A new **"Tracker"** item in the sidebar **System** section, above **Stats** (`icon="grid"` or
`"table"` — see Icon note). Full shell screen at route `tracker`. It is a peer of Stats: a
personal, cross-library overview.

---

## Layout (top → bottom)

### 1. Header

- Eyebrow `TRACKER` (mono, `var(--accent)`, 0.66rem, 0.16em tracking, uppercase).
- Title **Tracker** (`--font-display` 800, `--text-display-l`).
- Note (`--text-secondary`, max 560): _"Every series, every issue, at a glance. Click a number to
  mark it read."_

### 2. Summary + overall progress (a `--surface-raised` panel, hairline, radius 8, pad 16/18)

Mono row: `{tracks} series` · divider · `{issues} issues` · divider · `{read} read · {reading}
reading · {todo} to go` · divider · `{gaps} missing` · spacer · `Badge tone="accent" mono`
**{pct}% complete**. Then a `.ch-progress` bar, fill = `read / issues`.
(`gaps` = tracked issues with no library file — a "you're missing these" nudge.)

### 3. Toolbar (flex, wraps)

- **Add series** — primary `Button icon="plus"` → opens the Add flow (link a library series, or
  create a standalone track). Left-aligned.
- **Search** — `Input` (search glyph) filtering rows by series/track name as you type; clearable.
- **Scope filter** — a `Select` (or filter popover) scoping which tracks show by an existing
  grouping: `All` · a **Library** · a **Collection** · a **Reading list** · a **Smart list**.
  When scoped to a collection/list, only tracks whose series belong to that grouping appear (plus a
  `Standalone only` option). Grouped options, reusing the same sources the sidebar lists.
- **Hide read series** — a `Switch` (mono label `Hide read`). When on, fully-completed tracks
  (`read === total`) drop out so the grid shows only what's left to read. Off by default. This is
  the quick everyday filter; keep it prominent next to Search.
- **Status chips** — `Tag` toggles for finer slicing: `All` · `In progress` · `Incomplete` ·
  `Has gaps`. Single-select, `All` default. (`Completed` is covered by the Hide-read switch.)
- **Density** — S / M / L segmented control sizing the cells (see Cell sizing).
- **Legend** — a compact key of the four cell states (swatch + mono label), right-aligned. Doubles
  as the reading guide; keep it always visible, it's what makes the grid legible.

Search + scope + hide-read + status compose (all AND together); the summary panel counts reflect
the current filter, with a muted `of {allTracks}` when filtered.

### 4. The grid (frozen panes)

A scroll container with a **sticky left column** (row headers) and a **sticky top row** (the issue
ruler); the top-left **corner cell** is also sticky and holds the scope label (e.g. `NEW 52`, or
`ALL`). Horizontal scroll reveals higher issue numbers; vertical scroll pages through tracks. Rows
are zebra-free (flat), separated by a 1px `--border-hairline` seam — gallery-tight, like the cover
grid. This is the primary surface; give it the height.

**Column ruler (sticky top):** one label cell per issue in the sorted union of issue numbers across
the visible (filtered) tracks — `#0 #1 … #23 #23.1 #23.2 #24 … #N` — mono, `--paper-400`, tabular,
centered, on `--surface-raised`. Point issues get their own column like any other. Lazy-render
columns for very long runs (#0–#52+). A faint heavier tick every 5th integer column helps the eye
track down a column.

### 5. Track row (`TrackRow`)

Height follows density (see sizing). Two parts:

**Row header (sticky left, width ~220 at M):**

- Series/track **name** (`--font-body` 600, `--paper-100`, ellipsis). Click → the series detail
  screen (if linked) or the track's edit sheet (if standalone).
- Sub-line (mono, `--paper-600`): a **link glyph** — `Icon "library"` cyan if linked to a library
  series, or a `Tag size="sm"` **Manual** if standalone — then `{read}/{total}`, then a hairline
  `{pct}%`. A completed track (`read === total`) tints the whole header cell `--accent-soft` and
  shows a small `check` in `--accent` (mirrors the sheet's fully-green name cells).
- A slim per-row `.ch-progress` (height 3) can sit under the name at M/L density; hide at S.

**Cells:** one `IssueCell` per ruler column. A track that has no issue for that column (including
point-issue columns like `#23.1`) renders an **empty ruler slot** (blank, `--bg-app`, no border) —
nothing to mark there.

### 6. Issue cell (`IssueCell`) — the core interaction

A square-ish button, mono issue number centered, `.ch-reg` registration ticks on hover (the brand
hover), `cursor: pointer`, 2px cyan focus ring. Its fill encodes state:

| State              | Meaning                                           | Fill / treatment                                                                                                                                                                  | Number color            |
| ------------------ | ------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------- |
| **Read**           | you've read it                                    | solid `var(--accent)`                                                                                                                                                             | `var(--text-on-accent)` |
| **Reading**        | in progress                                       | `var(--accent-soft)` + 2px bottom `var(--accent)` underline bar (echoes the spine-tab "in progress" signature)                                                                    | `var(--accent)`         |
| **Unread · owned** | file in library, unread                           | `var(--surface-card)`, hairline border; a 5px `var(--unread)` **corner dot** (top-right) = "you have this, go read it" — magenta at its one reserved meaning, as a dot not a fill | `var(--paper-100)`      |
| **Unread · gap**   | issue exists but no file / read elsewhere-not-yet | transparent, **dashed** hairline outline, dim                                                                                                                                     | `var(--paper-600)`      |
| **Manual mark**    | not in a library, but you marked it read          | solid `var(--accent)` with a 1px dotted underline under the number = "tracked, no file"                                                                                           | `var(--text-on-accent)` |

**Read = cyan** (confirmed, not the sheet's green): cyan is ComicHub's semantic for _progress_, so
a wall of cyan reads as "how far through the collection you are," and it keeps the one-saturated-
color discipline. `--success` green stays for true status only; magenta stays reserved for the
unread nudge dot.

**Point issues (`#23.1`, `#23.2`):** treated as ordinary issues — each gets **its own cell in its
own ruler column**, independently clickable with its own state. No special stacking.

**Named specials** (`Annual`, `Futures End`, `Endgame`, one-shots): **their own track rows**, laid
out on their own short ruler (`#1 #2 …`), exactly as the reference sheet does. Do not try to fit
them on the main integer ruler.

### 7. Add affordances

- **End-of-row `+`** — a ghost cell after a track's last issue: _"Add issue"_. Opens a small popover
  to add a single number or a **range** (`#24–#52`), or _"Fill known issues"_ when the linked
  provider metadata knows the full run (creates gap cells for everything the series should have).
- **Add series** (toolbar) — a `Dialog`: tab 1 _Link a library series_ (search existing series,
  multi-select, they become linked tracks); tab 2 _Standalone track_ (name it, then add
  issues/ranges — name and issues only, no other metadata). Newly-scanned library series appear as
  tracks automatically without this flow.

---

## Interactions

- **Click a cell → toggle read/unread.** Owned issue → `markBook(bookId, 'read'|'unread')`.
  Gap/manual issue (no `bookId`) → toggle a tracked-read flag on the track item. Optimistic; the
  fill flips instantly. `Reading` toggles to `Read` on first click, `Unread` on the next.
- **Range mark.** Click a cell, then **Shift-click** another in the same row → the whole run flips
  to the target state (target = the _opposite_ of the anchor cell's state, so Shift-click reliably
  "marks up to here"). Drives a `batchProgress` call for owned issues. This is the bulk-catch-up
  path — mark `#1–#40` read in two clicks.
- **Hover → `CellPopover`** (a `--surface-raised` floating card, low cool shadow, blur scrim):
  cover thumb (`.ch-reg`, ~40×60), `{series} {number}`, mono `{pages} pp · {date}`, present/missing
  line, and for `reading` a `ProgressBar value={page} max={pages}`. Registration ticks on the cell
  itself while hovered.
- **Context menu** (right-click / long-press, or a `more` affordance in the popover): _Mark read_,
  _Mark unread_, _Mark up to here read_, _Open in reader_ (owned only → `openReader`),
  _Go to issue_ (issue detail), _Go to series_, _Add note_, _Remove issue_ (manual/gap only).
- **Keyboard:** grid is a roving-tabindex 2-D grid — arrow keys move the focused cell, `Enter`/
  `Space` toggles, `Shift+Enter` extends a range from the anchor. Focus ring always visible.

### Navigation (a required affordance — the grid is not a dead end)

Clicking a cell is reserved for the read/unread toggle, so navigation has its own explicit paths:

- **Row header → series.** The track name is a link → the **series detail** screen for linked
  tracks; a small trailing `chevron-right` `IconButton` (appears on row hover) makes it obvious.
  Standalone tracks open their edit sheet instead.
- **Cell → issue / reader.** The hover `CellPopover` carries the destinations: a primary
  **Open in reader** (owned → `openReader` at the right page) and a **Issue details →** link
  (`chevron-right`) to the issue screen. The same two live in the cell context menu.
- These reuse the app's existing `openReader` / series-detail / issue-detail routes so the Tracker
  is wired into the same navigation as the rest of the client.

All reader-opening actions reuse the app's `openReader`; all read-state writes go through the
progress API so the Tracker, sidebar "Continue reading", and Stats stay consistent.

---

## Cell sizing (density)

| Density     | Cell  | Number  | Row header | Row height |
| ----------- | ----- | ------- | ---------- | ---------- |
| S           | 22×22 | 0.6rem  | 180        | 26         |
| M (default) | 30×28 | 0.68rem | 220        | 34         |
| L           | 40×34 | 0.76rem | 260        | 42         |

Cells are gap-`1px` (a hairline seam, matching the gallery's 2px ink seam idea at grid scale),
square-cornered (radius 0–2 only) — these are catalog slots, not chips.

---

## Data shape (`TRACKER`)

```js
Track = {
  id,
  seriesId,             // linked library series id, or null for standalone
  name,                 // 'Batman Vol. 2'
  link,                 // 'library' | 'manual'
  publisher, year,      // optional meta (from the series/provider)
  issues: [Issue, …],   // ordered by sort
  // derived at render: read, total, pct, gaps
}
Issue = {
  id,
  number,               // '0' | '13' | '23.1' | 'Annual 1'  → render via issueLabel()
  sort,                 // numeric sort key (23.1 → 23.1) — also the ruler column key
  bookId,               // library file id, or null (gap / read-elsewhere / manual)
  state,                // 'read' | 'reading' | 'unread'
  page, pages,          // for 'reading'
  source,               // 'library' | 'manual'
}
```

Grid build: `ruler = sortedUnique(issue.sort across visible, filtered tracks)` — one column per
distinct issue number (`#23` and `#23.1` are separate columns). Each track maps `ruler → issue|null`.
Named-special issues (`Annual …`, `Futures End`) are split into their own synthetic tracks.

Prototype seeds ~14 DC tracks (New 52 flavor — Action Comics, All-Star Western, Aquaman, Batman,
Batwoman, Batgirl, Detective Comics, Demon Knights, …) with a realistic mix of read runs, a few
in-progress, owned-unread, and gaps, plus a couple of standalone tracks and Annual rows — enough to
show every cell state and the frozen-panes scroll.

---

## Data model & backend (engineering note — flag, not blocker)

The screen needs Tracker-owned persistence beyond today's `series`/`books`/`progress`:

- **`tracks`** — id, optional `series_id` link, name, `link` (library|manual), publisher/year,
  order. Library series auto-materialize a track on scan; deleting a library series leaves the
  track (like the deletion-proof reading lists) but drops the link.
- **`track_issues`** — id, track*id, number/sort/base, optional `book_id` link, `source`, and a
  **`read`** flag + optional `read_at` that is \_independent of a file* (so gap/manual issues carry
  read state). When `book_id` is set, read state mirrors `progress` for that book (single source of
  truth: the progress API); when null, it lives here.
- Likely REST: `GET /api/v1/me/tracker` (tracks + issues + derived counts),
  `POST /api/v1/me/tracker/tracks`, `POST …/tracks/{id}/issues` (single/range/fill-known),
  `POST …/issues/{id}/mark {read|unread}` (routes to `markBook` when linked), and a batch mark.
- Auto-link on scan + provider "known issues" fill reuse the existing matcher.

Design can proceed on mock data; this note is so the eng handoff scopes the store + endpoints.

---

## Empty / loading states

- **Empty:** `.ch-halftone` field + `EmptyState` — _"Nothing tracked yet. ComicHub tracks every
  series in your libraries here automatically — or add a series to start."_ + **Add series** CTA.
- **Loading:** the frozen header + a few skeleton rows (shimmer-free; quiet `--surface-card`
  blocks), so the grid shape is stable before data lands.
- **Filtered-to-empty:** _"No series match '{query}'."_ with a **Clear filter** ghost button.

---

## Components & tokens (all from the DS)

`Button`, `IconButton`, `Badge`, `Tag`, `Input`, `Select`, `Dialog`, `Tooltip`, `EmptyState`,
`ProgressBar`, `Icon` from `window.ComicHubDesignSystem_c0e1bf`; the `.ch-reg`, `.ch-progress`,
`.ch-halftone`, and spine-tab `clip-path` print signatures from `base.css`; semantic tokens only
(`--accent`, `--accent-soft`, `--unread`, `--surface-raised/-card`, `--border-hairline`,
`--paper-*`, `--text-on-accent`, `--radius-*`, `--font-display/-mono`). **No new colors.**

**Icon note:** the nav/toolbar want a `grid`/`table` glyph and a `plus`; if the registry lacks a
suitable `grid`, add it to `Icon.jsx` in the Lucide style (24×24, 1.5px stroke) rather than
importing another family — same rule as the rest of the system.

---

## Confirmed decisions

- **Read = cyan.** Brand colors hold; no green. (`--success` reserved for status, magenta for the
  unread dot only.)
- **No orange / "reading-from-the-sheet" state carried over.** The `reading` cell (cyan-soft +
  underline) still exists for genuinely in-progress issues via the progress API — it is the app's
  own state, not a port of the sheet's orange.
- **Point issues (`#23.1`) are ordinary issues** with their own cell and ruler column.
- **Filters/options in scope:** Search, Hide read series, Scope filter
  (library / collection / reading list / smart list / standalone), and explicit navigation to
  series & issues — all specified above.
- **Standalone-track header → a lightweight edit sheet** with only the **series name** and its
  **issues** (add/remove/mark) — no publisher/year or other metadata. Linked-track headers always
  go to series detail.
