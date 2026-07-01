# Design handoff — "Now reading" presence (Phase 3, Milestone E)

For **Claude Design** to produce a preview in the **`Design Preview v2`** project
(`ef2d1724-12c0-48dd-98e8-996e5b3ee416`). The presence backend is being built now
(WS `presence` topic + REST snapshot — data shape below is final); this handoff is the
client surface that shows it.

## What presence is

Ambient household awareness: when several people share one ComicHub server, you can see
that _Jordan is reading Nova Tide #012_ right now. It is **calm, glanceable, and
low-stakes** — not a feed, not notifications, no interaction beyond maybe jumping to the
book. Presence only exists in **server mode with auth on** (multiple real accounts);
embedded/single-owner installs never show it.

## How the backend behaves (design to this)

- A user is "now reading" while they turn pages; their entry **fades out ~5 minutes**
  after the last page turn, on finishing the book, and on marking it read/unread. No
  "online/offline" — only reading activity.
- Delivered as a REST snapshot (initial render) + live WS `presence` events
  (`presence.updated` / `presence.cleared`). Entries update as readers turn pages
  (page/percent creeps up live).
- **Each entry:** `userId`, `displayName`, `bookId`, `bookTitle`, `seriesTitle`,
  `page`, `pageCount`, `updatedAt`. (Cover art is fetchable from `bookId` via the normal
  covers endpoint.)
- **The viewer's own entry is included** — the UI decides whether to show "you".
- Content restrictions are enforced server-side: a restricted viewer never receives
  entries for books above their ceiling (invisible, not teased — same rule as browse).
- Typical scale: 0–4 concurrent readers. Zero readers is the _normal_ state most of the
  day — the empty state is "absence of the feature", not an empty-state illustration.

## What to design

**Placement is your call** — propose what fits the longbox shell. Candidate surfaces (pick
one primary; a secondary echo is fine if it stays quiet):

1. **Home screen** — a small "Now reading" strip/cluster near the top (e.g. beside the
   greeting or above Continue Reading): avatar + name + book, cover-forward if it earns
   the space.
2. **Sidebar** — a compact block in the `SidebarLongbox` (it already has a "now reading"
   resume slot for _you_; household presence could sit near it).
3. **Account popover area** — presence as part of the server/account surface (see
   `design_handoff_account`).

**Row/chip anatomy (whatever the placement):** avatar (the `Avatar` + per-user accent
pattern from `design_handoff_user_management`), display name, book (`Series #num` or book
title), and a subtle progress hint (`p.18/28`, a thin progress bar, or percent — pick one,
keep it mono/tertiary). Consider a tiny "live" cue (pulsing dot?) only if it stays calm.

**Interactions:** clicking a presence entry may open that book's detail screen (it's
already visible in the library) — or be inert; your call. No messaging, no reactions.

### States

1. **Nobody reading (default)** — the surface collapses/disappears entirely. Show a frame
   proving the layout is stable without it.
2. **One reader** — the common case.
3. **Several readers (3–4)** — must not crowd the host surface; cap + overflow ("+2 more"?)
   if the placement needs it.
4. **You + others** — decide whether "you" appears (labelled `You` like the user-management
   rows) or is filtered out.
5. **Live update** — an entry's page count ticking up / an entry fading out when a reader
   stops. Static frames are fine; note the intended motion (respect the brand-motion
   guideline — quiet, no bouncing).

## Components & tokens

Everything from the design system: `Avatar` (+ the `ACCENTS` per-user colors used in the
user-management handoff), `Badge`, mono micro-labels, `--surface-*` / `--border-hairline` /
`--accent` tokens, existing cover treatments (`ch-reg`) if covers appear. No new colors, no
new component contracts expected.

## Deliverable

`design_handoff_presence/` (a `README.md` spec + `ClientPreview.jsx`), with frames for the
states above mounted in `ComicHub Preview Screens.dc.html`. Once it exists, the client UI
gets built against it; the backend (WS topic, snapshot endpoint, restriction filtering)
will already be live.
