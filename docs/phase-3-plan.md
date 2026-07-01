# Phase 3 — Implementation Plan: Multi-user & remote

The working plan for [Phase 3 of the roadmap](08-roadmap.md): _turn on the "optional server"
half of the promise — a household runs one always-on server; each member reads independently
from any client_. Phases 1 (browse + read) and 2 (library platform) are complete. This is a
living document — milestone status is updated as work lands.

## What already exists (foundation laid in Phase 0/1)

Phase 3 is mostly **auth machinery + deployment**, not a data-model rewrite — the catalog was
built multi-user-ready from the start:

- **`user` table** already carries `id, username, display_name, role, password_hash,
age_rating_max, prefs, created_at` (`0001_init.sql`) — the full shape for accounts, roles,
  and content restrictions. `0002_seed_owner.sql` seeds the implicit `owner`.
- **All per-user data is already keyed by `userID`** — progress, bookmarks, reading lists,
  reader prefs all take a user id. The only stub is `currentUserID()` in `transport/http`,
  hardcoded to `domain.OwnerUserID`. Real auth plugs in at that single seam.
- **`config.ModeServer`** exists; `capabilities.multiuser` flips on in server mode; the
  loopback `tokenAuth` middleware and `/auth/handshake` endpoint exist (embedded identity).
- **`Repository` interface** abstracts the store (ADR-005: Postgres slots in behind it).
- **WS hub** (`jobs`/`progress`/`bookmarks` topics) is a clean place to add presence.
- Progress reconciliation is **last-writer-wins by `updatedAt`** (ADR-008) — cross-device
  conflict handling is largely already designed.

## Key technical decisions

1. **Dual-mode auth, one binary.** Embedded mode keeps the loopback token + implicit `owner`
   (no login — the local-first promise). Server mode requires real accounts. The middleware
   resolves the acting user into request context; `currentUserID()` reads it there instead of
   returning a constant. No per-handler changes — they already call `currentUserID(r)`.
2. **argon2id + JWT access/refresh** (per roadmap). Short-lived access token, long-lived
   rotating refresh token; `golang.org/x/crypto/argon2`. Client stores tokens in the **OS
   keychain** (Tauri keychain plugin), never plaintext.
3. **Roles enforced in middleware.** owner > admin > member > restricted. Admin-only routes
   (user management) gated by a role check. **Content restrictions:** `restricted` users get
   browse/read filtered by an `age_rating_max` ceiling — enforced server-side in the browse
   and reader services, not just hidden in the UI.
4. **Postgres behind the existing `Repository`** (ADR-005). A parallel implementation +
   migration parity; opt-in via config. SQLite stays the default. Sequenced late — it's
   independent of the auth surface.
5. **Discovery = mDNS + manual pairing.** Server advertises over mDNS/Bonjour in server mode;
   the client browses the LAN and also accepts a manually-typed URL. Pairing exchanges
   credentials for tokens via the login flow.
6. **UI is preview-gated.** Login, server pairing, account management, "now reading"
   presence, and stats dashboards are new screens — each needs a Design Preview v2 screen
   before it's built (the binding rule in CLAUDE.md). Backend/API lands first; UI follows the
   preview.

## Build order & rationale

Auth core leads — every other Phase 3 feature sits on real identity. Roles/restrictions and
the client auth UX follow. Discovery + sync make multi-device pleasant. Deployment (Docker/
service/Postgres/TLS) is independent and can land in parallel. Stats is last (nice-to-have).

```
A Auth core ─► B Roles + content restrictions ─► C Client auth UX (preview)
                                              ─► D Discovery ─► E Sync + presence
F Deployment (Docker/service/Postgres/TLS docs) runs in parallel
G Stats dashboards (preview) last
```

## Milestones

- **A — Auth core (server).** `user` repo CRUD; argon2id hash/verify; `POST /auth/login`
  (→ access+refresh), `POST /auth/refresh`, `POST /auth/logout`; JWT issue/verify; auth
  middleware that, in server mode, validates the JWT and puts the user in request context
  (embedded mode unchanged — owner). Rewire `currentUserID()` to read context. `/auth/handshake`
  reports the real user in server mode. Tests: hashing, token lifecycle, middleware, embedded
  still works.
- **B — Roles + content restrictions.** Role-gating middleware; admin user-management routes
  (`GET/POST/PATCH/DELETE /users`); `age_rating_max` ceiling applied in browse + reader for
  restricted users (server-enforced). Tests for each role's access matrix. **Carry-overs from
  the Milestone A security review** (must land before auth ships): role-gate `/admin/shutdown`
  (currently any authenticated user can call it); enforce the `restricted` ceiling (the role
  exists but isn't yet applied); and revoke a user's sessions (`Sessions().DeleteForUser`) on
  password change / role downgrade.
- **C — Client auth UX (preview-gated).** Login screen, server pairing (manual URL), account
  switcher, account management; OS-keychain token storage; refresh-on-401. Build against the
  Design Preview v2 screens.
- **D — Server discovery.** mDNS advertise (server) + LAN browse + manual URL pairing (client).
- **E — Cross-device sync + presence.** WS `presence` topic ("now reading"); confirm/extend
  conflict-aware progress reconciliation (ADR-008) across simultaneous devices.
- **F — Remote deployment.** Dockerfile + compose; Windows Service / systemd units; Postgres
  backend behind `Repository` (migration parity, config); TLS + reverse-proxy guidance docs.
- **G — Stats dashboards (preview-gated).** Aggregate endpoints (books/pages read, streaks,
  by genre/month) + dashboard UI against a Design Preview v2 screen.

## Status

| Milestone                                                         | Status                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| ----------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| A — Auth core (accounts, argon2id, JWT, middleware, context user) | ✅ done — server-side; opt-in via `--auth` (env bootstrap), tested e2e                                                                                                                                                                                                                                                                                                                                                                              |
| B — Roles + content restrictions                                  | ✅ done — role-gating + admin `/users` CRUD + age-ceiling enforcement (browse/search/reader); A-review carry-overs landed                                                                                                                                                                                                                                                                                                                           |
| C — Client auth UX (login / pairing / accounts) — preview-gated   | ✅ done — connect/login boot flow, account chip + sign-out, admin Users card; built to the Design Preview v2 handoffs, verified e2e (stats G1 still pending)                                                                                                                                                                                                                                                                                        |
| D — Server discovery (mDNS + manual pairing)                      | ✅ done — server advertises `_comichub._tcp` in server mode (`--mdns`, `--server-name`); client `discover_servers` browses + probes reachability; connect screen's "Servers on your network" list built to the updated connect_login handoff (scanning/results/empty/row-connecting states); open servers skip login, verified e2e in-browser                                                                                                       |
| E — Cross-device sync + presence                                  | 🟨 backend done — WS `presence` topic + `GET /presence` (derived from progress writes, TTL expiry, ceiling-filtered per viewer); progress/bookmarks WS delivery now per-user; ADR-008 LWW actually enforced (`updatedAt` on PUT, `POST /me/progress/batch` for offline flush, stale writes return the authoritative row); tested. Remaining: presence UI (preview-gated — handoff in docs/design-handoff-presence.md) + reader offline-flush wiring |
| F — Remote deployment (Docker / service / Postgres / TLS docs)    | ⬜ not started                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| G — Stats dashboards — preview-gated                              | ⬜ not started                                                                                                                                                                                                                                                                                                                                                                                                                                      |

## Cross-cutting

Accessibility gate per new screen; security review of the auth surface (token handling,
password storage, role checks) before shipping; docs in lockstep (03-api.md for the new auth

- users endpoints, 09-tech-decisions.md for an auth ADR); bench thresholds unaffected.

## Verification

Per milestone: unit tests (hashing, token lifecycle, role matrix, restriction filtering);
end-to-end auth flow against a running `--mode server` (login → access protected route →
refresh → logout) driven over the socket; multi-user data isolation (two users, independent
progress/lists). The Phase 1 GUI browser-mode harness extends to auth once the login UI lands.
