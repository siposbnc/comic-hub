# 10 — Deploying a remote server

How to run `comichub-server` as the household's always-on server (Phase 3, Milestone F).
One binary, three ways to keep it running, one optional database upgrade. Everything here
assumes **server mode with auth on** — real accounts, reachable beyond the machine it runs
on. (The desktop app's embedded sidecar needs none of this.)

The moving parts:

| Piece      | Default                | Where it's decided                                    |
| ---------- | ---------------------- | ----------------------------------------------------- |
| Mode       | `server`               | `--mode` / `COMICHUB_MODE`                            |
| Bind       | `0.0.0.0:8080`         | `--bind` / `COMICHUB_BIND`                            |
| Data dir   | OS config dir          | `--data-dir` / `COMICHUB_DATA_DIR`                    |
| Auth       | off — **turn it on**   | `--auth` / `COMICHUB_AUTH_ENABLED=true`               |
| First user | —                      | `COMICHUB_ADMIN_USERNAME` / `COMICHUB_ADMIN_PASSWORD` |
| Database   | SQLite in the data dir | `COMICHUB_DB_DRIVER` / `COMICHUB_DB_DSN` (Postgres)   |
| Discovery  | mDNS on (LAN only)     | `--mdns=false` to opt out, `--server-name` to label   |

The admin bootstrap envs are read **once, on first boot**, to create a login-capable
account; rotate the password from Settings afterwards. Liveness/readiness for any
supervisor: `GET /healthz` / `GET /readyz` (unauthenticated).

## 1. Docker (recommended)

[`server/Dockerfile`](../server/Dockerfile) builds a distroless image (pure-Go binary, no
shell); [`deploy/docker-compose.yml`](../deploy/docker-compose.yml) is the household setup:

```sh
cd deploy
COMICS_DIR=/path/to/comics COMICHUB_ADMIN_PASSWORD='pick-a-long-one' docker compose up -d
```

- Comics mount **read-only** at `/comics`; create the library in the client pointing there.
- Catalog + caches live in the `comichub-data` volume; back that up, not the container.
- mDNS discovery doesn't cross the container network boundary on most setups — clients
  type `http://host:8080` once (it's remembered). On Linux, `network_mode: host` restores
  discovery.

## 2. systemd (Linux, bare metal)

[`deploy/comichub-server.service`](../deploy/comichub-server.service) — hardened unit
(dedicated user, `ProtectSystem=strict`, comics read-only). Install steps are in the
unit's header comment; secrets go in a `systemctl edit` drop-in, not the unit file.

## 3. Windows Service

[`deploy/install-windows-service.ps1`](../deploy/install-windows-service.ps1) registers
the binary as an auto-start service (elevated shell):

```powershell
pwsh -ExecutionPolicy Bypass -File deploy\install-windows-service.ps1 `
  -Binary 'C:\ComicHub\comichub-server.exe' -DataDir 'C:\ComicHub\data'
```

It prompts for the first admin password, stores the bootstrap in the service-scoped
registry environment (never the command line), and configures crash restarts.

## 4. Postgres instead of SQLite (optional)

SQLite is the default and right for almost everyone — one file in the data dir, zero
administration. Postgres slots in behind the same `Repository` interface (ADR-005) when
you want the catalog in a managed database (shared storage, existing backup story):

```sh
COMICHUB_DB_DRIVER=postgres
COMICHUB_DB_DSN='postgres://comichub:secret@db:5432/comichub?sslmode=disable'
```

Migrations run automatically on boot, same as SQLite. There is **no automatic data
migration between the two** — pick one before your first scan (or rescan; the catalog is
derived from your files, only progress/lists/accounts are original data).

## 5. TLS & reverse proxy

The server speaks plain HTTP and expects TLS to be terminated in front of it — the same
posture as most self-hosted media servers. On a LAN this is usually skipped; the moment
the server is reachable from outside, put a proxy in front:

**Caddy** (automatic Let's Encrypt):

```caddy
comics.example.com {
    reverse_proxy 127.0.0.1:8080
}
```

**nginx:**

```nginx
server {
    listen 443 ssl;
    server_name comics.example.com;
    ssl_certificate     /etc/letsencrypt/live/comics.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/comics.example.com/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        # WebSocket upgrade (jobs/progress/presence push):
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        # Page images can be large; don't buffer streams to disk:
        proxy_buffering off;
        client_max_body_size 0;
    }
}
```

Notes that matter behind a proxy:

- **WebSockets** must be forwarded (`/api/v1/ws`) — presence, job progress, and
  cross-device sync ride on it.
- Bind the server to loopback (`--bind 127.0.0.1:8080`) when the proxy is on the same
  host, so nothing bypasses TLS.
- Never expose the server without `--auth` — without it every visitor is the owner.
