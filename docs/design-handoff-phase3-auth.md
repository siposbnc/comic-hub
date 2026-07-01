# Design handoff — Phase 3 (Multi-user & remote) UI

For **Claude Design** to produce preview screens in the **`Design Preview v2`** project
(`ef2d1724-12c0-48dd-98e8-996e5b3ee416`). The server side of these features is **already
built and tested** (Phase 3 Milestones A + B); this handoff is the client UI that consumes it.
The reader is unaffected — all screens here are in **`apps/client`**.

Build everything from the existing design system (`@comichub/ui` tokens + components — the
"longbox" client shell: vertical spine sidebar, calm dark surfaces, covers are the loud
thing). Reuse the established Settings screen patterns (cards, `Row`, `Seg`, `Toggle`, `Badge`,
`Button`, `Switch`, `Dialog`). Produce a `design_handoff_<feature>/` (a `README.md` spec + a
`ClientPreview.jsx`) per feature below, plus the needed states.

## Context: when does auth UI appear?

The desktop app is **local-first**. Two connection modes:

- **Embedded / auth-disabled** (today's default): the bundled sidecar, single implicit owner,
  no login. **None of these screens show.**
- **Server mode with auth enabled**: the client connects to a remote ComicHub server that
  requires accounts. Then login + account UI applies.

So all of this is **conditional UI** gated on `serverInfo.mode === 'server'` and the server
requiring auth (a 401 on the handshake). Design the screens; they simply don't render in
embedded mode.

---

## C1 — Connect to a server (pairing)

**Purpose:** point the client at a remote server by URL (LAN discovery comes later in
Milestone D; for now, manual entry). This precedes login.

**Layout:** a centered, focused "connect" card on an empty app background (no sidebar yet —
the user isn't connected). ComicHub wordmark/logo at top.

**Fields/states:**

- A single **Server URL** text field (`http://host:port`), with a primary **Connect** button.
- **Connecting** state (spinner on the button, field disabled).
- **Error** state: "Couldn't reach that server" (network) — inline, calm danger tone.
- A subtle hint line: "Running ComicHub on this device instead? It starts automatically."
  (for users who don't need a remote server).
- (Future slot, design but mark optional/disabled: a "Servers on your network" list for mDNS
  discovery — Milestone D. Show an empty/"coming soon" affordance or omit.)

**Data:** the URL is stored client-side; the next step is the login screen for that server.

---

## C2 — Login

**Purpose:** authenticate to a server in auth mode.

**Layout:** same centered-card treatment as C1 (continuation of the connect flow), showing the
server it's logging into (host shown small above/below the title, e.g. "Sign in to
comichub.home.lan").

**Fields/states:**

- **Username** + **Password** fields, primary **Sign in** button.
- **Submitting** state (button spinner, fields disabled).
- **Error** state: "Incorrect username or password" (the server returns a single 401 for both
  — do not distinguish which was wrong). Inline danger.
- A "← Use a different server" back link to C1.
- No "sign up" — accounts are created by an admin (see C4).

**Wire (already built):**
`POST /auth/login {username, password}` →
`{ access, refresh, accessExpiry, user: { id, username, displayName, role } }`.
401 on bad credentials. Tokens are stored in the OS keychain (client handles this); design
needs no "remember me".

---

## C3 — Account chip + sign out

**Purpose:** show who's signed in and let them sign out. Lives in the **app shell** once
connected (auth mode only).

**Placement:** the top utility bar (where the current "C" avatar/initial sits today). In auth
mode it becomes an **account chip**: avatar/initial + display name; click opens a small popover
with the username, role (a small `Badge` — owner/admin/member/restricted), and **Sign out**.

**States:** signed-in (default). Token-expired/disconnected → a gentle banner "Reconnect"
that returns to C2 (the client auto-refreshes tokens; this is the fallback when refresh fails).

**Wire:** `GET /auth/handshake` → current user; `POST /auth/logout {refresh}`.

---

## C4 — User management (admin)

**Purpose:** owners/admins create and manage accounts. **Admin-only** — hidden for
member/restricted users.

**Placement:** a new **"Users"** card/section in **Settings** (matches the existing Settings
card layout — see the "Metadata providers" card for the pattern).

**Layout:**

- A **list of users**: each row shows avatar/initial, display name, `@username`, a role
  `Badge`, and (for restricted users) their content ceiling. Row actions: **Edit**, **Delete**
  (Delete hidden/disabled for the owner account; confirm via `Dialog`).
- An **"Add user"** primary button → opens a **create dialog** (`Dialog`).

**Create / Edit dialog fields:**

- **Username** (create only; immutable after).
- **Display name**.
- **Role** — a `Seg`/select of: Owner · Admin · Member · Restricted.
- **Password** — set on create; on edit, an optional "Set new password" field (blank = keep).
- **Content rating ceiling** — shown **only when role = Restricted**: a select of age ratings
  (Everyone, Everyone 10+, Teen, Mature 17+, Adults Only 18+). This caps what the user can
  see/read. Include a one-line explainer: "This user only sees issues rated at or below this."

**States:** list (with rows), empty (just the owner), create dialog, edit dialog, delete
confirm, validation errors (e.g. "Password must be at least 8 characters", "Username already
taken" → 409).

**Wire (already built):**

- `GET /users` → `{ users: [{ id, username, displayName, role, ageRatingMax }] }`
- `POST /users {username, displayName, role, password, ageRatingMax}` → 201
- `PATCH /users/{id} {displayName?, role?, ageRatingMax?, password?}`
- `DELETE /users/{id}` → 204 (owner can't be deleted → 400)

**Note on restriction behavior (for empty/locked states elsewhere):** a restricted user
simply never sees over-rated content — it's filtered from grids, series, search, and the
reader. No "locked content" placeholder is shown (restricted items are invisible, not teased).
So no new "blocked" UI is needed; just ensure grids/empty-states read well when smaller.

---

## G1 — Stats dashboard (Milestone G — lower priority, design when convenient)

**Purpose:** a personal reading dashboard. Per-user.

**Placement:** a new top-level nav item ("Stats", in the sidebar "System"/below Lists) or a
section on Home — your call; propose what fits the longbox shell.

**Content (design the cards/charts; data shapes will follow the design):**

- **Headline numbers:** books read, pages read, current streak (days).
- **Over time:** a simple bar/area chart of issues read per month.
- **By genre / publisher:** a small ranked breakdown (top genres).
- Keep it calm and cover-forward where possible (e.g. recently finished covers).

Mark this **secondary** — C1–C4 are the priority (they unblock multi-user); G1 can come after.

---

## Deliverables checklist for Claude Design

1. `design_handoff_connect_login/` — C1 connect + C2 login (+ connecting/error states).
2. `design_handoff_account/` — C3 account chip/popover + reconnect banner.
3. `design_handoff_user_management/` — C4 Users settings card + create/edit/delete dialogs
   (incl. the role=Restricted → ceiling field).
4. `design_handoff_stats/` — G1 dashboard (secondary).

Each as a `README.md` spec + `ClientPreview.jsx`, using DS components/tokens, matching the
existing client shell. Once these exist in `Design Preview v2`, the client UI will be built
against them (per the project's preview-first rule).
