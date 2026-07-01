# Design handoff — Server discovery list (Phase 3, Milestone D)

For **Claude Design** to produce a preview in the **`Design Preview v2`** project
(`ef2d1724-12c0-48dd-98e8-996e5b3ee416`). The discovery backend is **already built and
verified e2e**: servers advertise themselves over mDNS, and the client can browse the LAN
and returns reachable, ready-to-connect URLs. This handoff is the one missing UI piece —
the **active "Servers on your network" list** on the connect screen.

## Where it lives

This extends **C1 (Connect)** from `design_handoff_connect_login/` — same `AuthScaffold`,
same 408px centered card. That handoff already designed the section as a
**designed-but-disabled future slot**: a hairline-divided "Servers on your network" heading
with a `Soon` chip and a dashed 60%-opacity "Automatic discovery arrives in a later
update." row. **Replace that placeholder with the live version.** Everything else on C1
(URL field, Connect button, hint line, connecting/error states) is unchanged and already
built in the client.

Client only (`apps/client`); the reader is unaffected.

## How discovery behaves (already built — design to this)

- On showing the connect screen the client starts a **browse sweep** that lasts **~2.5 s**
  and then resolves with everything found (possibly nothing). It is a discrete sweep, not
  a live stream — results arrive as one batch when the sweep ends.
- Sweeps can be re-run ("Scan again"). Servers may come and go between sweeps.
- Each discovered server carries:
  - `name` — human-readable server name (e.g. `Living-Room-NAS`; the server's
    `--server-name`, default its hostname).
  - `url` — ready-to-connect base URL, e.g. `http://192.168.1.10:8080` (already probed
    for reachability by the backend; treat it as trustworthy).
  - `version` — server version string, possibly absent.
  - `auth_required` — whether that server needs a login. Picking one routes to C2 (login)
    after connecting; picking an open server goes straight into the app.
- Discovery only exists in the desktop app. (In a plain-browser dev run the list is
  simply absent — no state needed for that beyond "section hidden".)

## What to design

The section under the manual URL field (manual entry stays primary — discovery is the
convenience path, not a replacement):

- **Section header:** "Servers on your network" (keep the hairline divider treatment from
  the placeholder) + a quiet **re-scan** affordance (icon-button or text button; shows the
  scanning state while a sweep runs).
- **Server row** (the core new piece): server name, its URL (mono, tertiary — the same
  visual voice as the URL input), and when `auth_required` a small **"Sign-in required"**
  indication (e.g. a `lock`/`shield` glyph + label or a quiet Badge — your call). Version
  can appear as a tertiary detail or tooltip; it's low-priority metadata. The whole row is
  clickable → it connects to that server (same behavior as typing the URL and pressing
  Connect).
- **Row → connecting:** picking a row enters the existing C1 connecting treatment. Decide
  whether the row itself shows a spinner or the card falls back to the global connecting
  state — either is fine, pick what reads calmer.

### States

1. **Scanning** — first sweep in flight (and re-scans): a quiet in-progress row/shimmer in
   the section (small `Spinner` + "Looking for servers…"). Should read as ambient, not
   blocking — the URL field stays usable throughout.
2. **Results** — 1–3 rows typical; design with 2 so spacing reads. No pagination; a
   household realistically has one or two servers.
3. **Nothing found** — calm empty line ("No servers found on your network.") + the re-scan
   affordance. Must not read as an error — manual entry right above it is the answer.
4. **Row connecting** (per the decision above).

A failed connect after picking a row reuses C1's existing error state (danger border +
"Couldn't reach that server…"), with the picked URL filled into the field so the user can
see/edit what was tried — no new error design needed, but show one frame of it if cheap.

## Components & tokens

Everything from the design system as in `design_handoff_connect_login/`: tokens
(`--surface-raised`, `--border-hairline`, `--accent`, `--text-tertiary`, `--radius-md`…),
`Spinner`, `Badge`, the `server` / `shield` glyphs already introduced there. No new colors,
no new component contracts expected.

## Deliverable

Update `design_handoff_connect_login/` in place (README + `ClientPreview.jsx` states —
extend the `screen="connect"` frames with `discovery="scanning" | "results" | "empty"`
or equivalent), or produce a sibling `design_handoff_discovery/` if that's cleaner on the
canvas — either works. Once it exists, the client list gets built against it and Milestone
D closes.
