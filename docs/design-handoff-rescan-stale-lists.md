# Design handoff — Series rescan + deletion-proof reading lists

For **Claude Design** to produce previews in the **`Design Preview v2`** project
(`ef2d1724-12c0-48dd-98e8-996e5b3ee416`). The backend is built and live (data shapes
below are final); this handoff covers **four client surfaces**.

## The feature, in one story

A series was cataloged wrong (bad filenames, an old scanner bug). The user opens the
series and hits **Rescan** — ComicHub deletes the series and re-catalogs its files from
scratch. Any reading list that referenced those issues does **not** lose them: each entry
stays in place as a **stale placeholder** (its snapshot — series, number, title — was
captured when it was added). When the rescan re-creates the same files, placeholders
**re-attach automatically** (content-hash match) — usually the user never notices.
When a file is truly gone (deleted library, replaced file), the placeholder persists
indefinitely: the reading order is sacred. The user can **link** a placeholder to a real
issue whenever one appears, and can even **add placeholders by hand** for issues they
don't own yet — planning a reading order ahead of acquiring the books.

## Backend behavior (design to this)

- `POST /series/{id}/rescan` → deletes the series + books, starts a library scan, returns
  a `jobId` (normal job — the existing `JobIndicator` tracks it). Progress on those
  issues resets; matched metadata for that series is intentionally discarded (fresh start).
- Reading list detail now returns ordered `items`:
  `{ id, stale, seriesName?, number?, title?, addedAt, book?: BookCard }`.
  `stale: true` → no `book`; render from the snapshot fields. Stale entries hold their
  slot, drag-reorder like any row, and can be removed — but **cannot be opened/read**;
  Resume/"Next unread" and the reader's queue **skip over them**.
- `POST …/items` accepts `manual: [{seriesName, number, title}]` (placeholders); at least
  a series name or a title is required per entry.
- `PATCH …/items/{itemId}/link {bookId}` links a placeholder (or re-points any entry) to
  a real issue; rejected with a friendly message if that issue is already in the list.
- Auto-relink is silent and server-side; no UI needed beyond the row simply becoming
  normal again after a rescan.

## Surfaces to design

### 1. Series detail — "Rescan series" action

An action on the series screen (`Series.tsx` today has the hero with Match metadata
etc.). It is semi-destructive: local metadata + read progress for that series reset,
reading lists are safe (say so!). Needs a confirm step (dialog or equivalent) that
explains exactly that in one calm sentence, then hands off to the normal job indicator.
Placement/affordance is your call (overflow menu vs. secondary button — it should not
compete with **Read**/**Match**).

### 2. Reading list detail — stale rows

Extend the existing Reading List screen (see `design_handoff_reading_list`): a stale row
renders from its snapshot (`{seriesName} {number}`, title if any), keeps its order index
and drag handle, but has **no cover art, no progress, no Read action**. It needs:

- a clear-but-quiet "missing" treatment (the row must read as intentionally kept, not
  broken — this is a feature, not an error), and
- a **Link issue** action that opens a picker (the `AddIssuesDialog` search pattern,
  single-select) to attach a real issue,
- Remove stays available.

States to show: linked (normal), stale, stale-being-linked (picker open), and the list's
stats line (`n issues · …`) counting stale entries distinctly (e.g. "8 issues · 2
missing" — wording yours).

### 3. Reading list detail — "Add missing issue" (manual placeholder)

An affordance next to/inside the existing **Add issues** flow for creating a placeholder:
series name (required-ish), number, title (at least one of series/title). Design where it
lives: a tab/secondary mode inside the Add Issues dialog, or a separate small dialog.
It should feel like jotting a want-list line, not filling a form.

### 4. Reading Lists index screen (refresh)

The list-of-lists screen (`ReadingLists.tsx`) is currently bare: name + count + create
field. Give it a real design pass consistent with the longbox shell — the user asked for
this screen to get a proper preview. Consider: cover collage or first-covers strip per
list, active-queue badge, read-progress summary per list, stale/missing count when > 0,
create flow, empty state. Content available per list: name, `bookCount`, active flag,
created/updated, and (via detail fetch) items with covers + progress.

## Constraints

- Design-system components + semantic tokens only (`--accent`, `--surface-raised`, …);
  covers loud, chrome quiet; magenta reserved for unread/new — a stale marker should
  probably NOT be magenta (it's not "new content"; tertiary/warning-muted territory).
- Reader is untouched; all four surfaces are client-only.
- Wire shapes above are final; if a design needs another field, flag it in the handoff
  README rather than assuming it.
