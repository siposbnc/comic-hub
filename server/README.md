# comichub-server

The ComicHub media server — a single Go binary that owns the library catalog, database,
files, caches, and background work. See [docs/04-server.md](../docs/04-server.md) for the
full design.

## Run (dev)

```sh
# from /server
go build -o bin/comichub-server.exe ./cmd/comichub-server

# embedded mode (loopback, ephemeral port, auto-generated token, handshake file)
./bin/comichub-server.exe --mode=embedded --data-dir=../.data --log-format=text
```

The server writes `<data-dir>/connection.json` with the chosen port and token; the client
reads that to connect. In embedded mode the API requires `Authorization: Bearer <token>`
(except `/healthz` and `/readyz`).

## Flags

| Flag | Default | Notes |
|------|---------|-------|
| `--mode` | `embedded` | `embedded` \| `server` |
| `--bind` | `127.0.0.1:0` (embedded) / `0.0.0.0:8080` (server) | listen address |
| `--data-dir` | `%APPDATA%/ComicHub` | data directory |
| `--token` | auto (embedded) | loopback bearer token; empty disables auth |
| `--handshake-file` | `<data-dir>/connection.json` | where to publish the connection |
| `--db` | `<data-dir>/comichub.db` | SQLite path |
| `--log-level` | `info` | debug\|info\|warn\|error |
| `--log-format` | `json` | json\|text |

All flags can also be set via `COMICHUB_<UPPER_FLAG>` environment variables.

## Layout

See [docs/04-server.md §2](../docs/04-server.md). Phase 0 establishes the entrypoint,
config, logging, SQLite + migration runner, the HTTP surface (`/healthz`, `/readyz`,
`/api/v1/server/*`, `/api/v1/admin/shutdown`, handshake auth), and the stubbed domain,
archive, and provider interfaces. Scanner, image pipeline, and metadata land in Phase 1+.
