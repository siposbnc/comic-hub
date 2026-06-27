# @comichub/ui

The ComicHub design system, consumed by the client and reader apps. The visual
foundation (tokens + the print-signature primitives) and components are **synced from
the ComicHub Design System** on `claude.ai/design` — they are not authored here.

## What's synced vs. authored

| Path                | Source                   | Notes                                                                                                                      |
| ------------------- | ------------------------ | -------------------------------------------------------------------------------------------------------------------------- |
| `src/styles.css`    | Design System (verbatim) | The single entry point apps import. `@import`s the token layers.                                                           |
| `src/tokens/*.css`  | Design System (verbatim) | colors, typography, spacing, effects, fonts, base (reset + `.ch-reg` / `.ch-spine-tab` / `.ch-progress` / `.ch-halftone`). |
| `src/components/**` | Design System (verbatim) | The full primitive set — JSX + sibling `.d.ts`. core, comic, forms, feedback, navigation.                                  |
| `src/index.ts`      | Authored here            | The barrel apps import from. Re-export every synced component here.                                                        |

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

## Enforcement

Use of the design system is enforced in app code (`apps/**`) by ESLint —
`eslint.config.mjs` at the repo root, run via `pnpm lint:ds` (and folded into
`pnpm lint`; CI runs it on every push). The rules are ported from the design
system's own adherence spec (`_adherence.oxlintrc.json`) and flag:

- raw hex colors and raw `px` values in string literals — use a token via `var()`;
- `font-family` values outside Archivo Expanded / Inter / IBM Plex Mono;
- props or `variant`/`size`/`tone`/etc. values that aren't in a component's declared
  contract (e.g. `<Button variant="fancy">`);
- deep imports into `@comichub/ui` internals instead of the barrel.

Violations are warnings (they surface, they don't block builds), matching the design
source. This package's own `src/` is verbatim design-system source and is exempt.
When the design system changes a component contract, re-port the selectors in
`eslint.config.mjs`.

## Syncing from the Design System

The design project (id `c0e1bfbe-c5d5-422a-b364-afc6dcdde00b`) is the source of truth.
Pull updates with the `/design-sync` skill (or the DesignSync read API), one component
at a time — never a wholesale replace. The fonts load from the Google Fonts CDN; for a
fully offline build, self-host the `.woff2` files and swap the `@import` in
`tokens/fonts.css` for local `@font-face` rules.
