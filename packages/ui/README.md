# @comichub/ui

The ComicHub design system, consumed by the client and reader apps. The visual
foundation (tokens + the print-signature primitives) and components are **synced from
the ComicHub Design System** on `claude.ai/design` — they are not authored here.

## What's synced vs. authored

| Path                             | Source                   | Notes                                                                                                                      |
| -------------------------------- | ------------------------ | -------------------------------------------------------------------------------------------------------------------------- |
| `src/styles.css`                 | Design System (verbatim) | The single entry point apps import. `@import`s the token layers.                                                           |
| `src/tokens/*.css`               | Design System (verbatim) | colors, typography, spacing, effects, fonts, base (reset + `.ch-reg` / `.ch-spine-tab` / `.ch-progress` / `.ch-halftone`). |
| `src/components/**`              | Design System (verbatim) | Pulled **incrementally, one component at a time, as screens need them** — JSX + sibling `.d.ts`.                           |
| `src/Button.tsx`, `src/index.ts` | Authored here            | Thin local shims/exports until the full component set is synced.                                                           |

Synced files are listed in the repo `.prettierignore` so they stay byte-faithful to
the design source and re-syncs diff cleanly.

## Usage

```ts
import '@comichub/ui/styles.css'; // once, at the app entry (main.tsx)
import { Button } from '@comichub/ui';
```

Reach for the semantic token aliases (`--accent`, `--surface-card`, `--text-primary`,
`--cover-w-m`, `--font-display`, …), not the raw ramp. Theme via `[data-theme='light']`
and `[data-accent='magenta'|'amber']` on a root element.

## Syncing from the Design System

The design project (id `c0e1bfbe-c5d5-422a-b364-afc6dcdde00b`) is the source of truth.
Pull updates with the `/design-sync` skill (or the DesignSync read API), one component
at a time — never a wholesale replace. The fonts load from the Google Fonts CDN; for a
fully offline build, self-host the `.woff2` files and swap the `@import` in
`tokens/fonts.css` for local `@font-face` rules.
