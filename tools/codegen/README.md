# codegen — supported formats

`formats.json` is the **single source of truth** for which comic formats ComicHub
supports. `gen-formats.mjs` renders that list into each language so the TypeScript, Rust,
and Go definitions can never drift apart.

## Edit a format

1. Change `formats.json`.
2. Run `pnpm codegen` from the repo root.
3. If you changed which formats `associate: true`, also update the reader's
   `apps/reader/src-tauri/tauri.conf.json` `fileAssociations` to match — the
   generator will fail loudly until they agree.
4. Commit the regenerated files alongside `formats.json`.

## Generated files (do not edit by hand)

| Target                                     | Consumes it                      |
| ------------------------------------------ | -------------------------------- |
| `packages/reader-core/src/formats.gen.ts`  | client + reader frontends        |
| `apps/reader/src-tauri/src/formats_gen.rs` | reader Tauri core (`formats.rs`) |
| `server/internal/domain/formats_gen.go`    | server (`formats.go`, scanner)   |

Each format has:

- `ext` — the file extension (lowercase, no dot)
- `kind` — `archive` (container of images) or `document` (e.g. PDF, rasterized)
- `container` — the underlying container format
- `associate` — whether the reader auto-registers as the OS default handler for it

## CI

`pnpm codegen:check` runs in CI and fails if any generated file is stale or if the
reader's file associations don't match `formats.json`.
