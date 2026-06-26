# 07 — Design System

> Design direction for ComicHub's client + reader. This is the brief Claude Design builds
> against. The guiding tension: **the covers are already loud — the app must be quiet.**

## 1. Concept: "The Longbox"

A collector's comics live in a longbox: a calm, neutral container whose entire job is to
present a wall of vivid spines and covers. ComicHub's chrome is that longbox — a restrained,
ink-dark gallery that never competes with the artwork. Every comic cover is a saturated,
high-contrast image; if the UI is also colorful, the result is noise. So the interface is
disciplined and neutral, and we spend our one bold idea on a motif drawn from how comics are
actually _made_: **CMYK process printing and its registration marks.**

This avoids the templated AI looks (cream+serif+terracotta, black+acid-green, broadsheet
hairlines): our boldness is a printer's-cyan spot color and crop/registration geometry,
both specific to this subject and earned by it.

## 2. Palette

Anchored in a cool ink-black "press" environment with a **process-cyan** spot color (the
heritage color of comic printing, and notably _not_ the AI-default vermilion/acid-green).
Magenta — the second process ink — is the single "new / unread" highlight. Yellow appears
only as a rare warning/key accent.

| Token | Hex | Use |
|-------|-----|-----|
| `ink-900` | `#0C0E12` | App background (the longbox interior). |
| `ink-800` | `#13161C` | Raised surfaces / sidebar. |
| `ink-700` | `#1C212B` | Cards, inputs, hover fills. |
| `ink-600` | `#2A313D` | Borders, dividers, hairlines. |
| `paper-100` | `#ECEEF2` | Primary text on ink. |
| `paper-400` | `#9BA3B0` | Secondary text, metadata. |
| `paper-600` | `#5C6573` | Tertiary / disabled. |
| `cyan-500` | `#16B9E6` | **Primary accent** — actions, focus, read-progress, links. Process cyan. |
| `cyan-600` | `#0E93B8` | Pressed / accent border. |
| `magenta-500` | `#E6398B` | **Unread / new** highlight only. Process magenta. |
| `yellow-400` | `#F4C13C` | Key / warning accent (rare). |
| `danger-500` | `#E5484D` | Destructive. |
| `success-500` | `#46A758` | Confirmations. |

- **Light theme** inverts to a true paper white (`#F7F6F2`, a hair warm — newsprint, not
  cream) with ink text; cyan/magenta keep their hues but darken one step for contrast.
- Covers always render against `ink-800/700` so saturated art pops; never tint cover imagery.
- Contrast floor: all text meets WCAG AA on its surface; cyan on ink passes for large/UI text.

## 3. Typography

Three roles, each doing one job — chosen to feel like **comic production**, not a generic UI kit.

| Role | Family | Rationale |
|------|--------|-----------|
| **Display** | **Archivo Expanded** (wide, 700/800) | Wide grotesque reads like a cover logo / masthead. Used sparingly: page titles, series heroes, the wordmark. |
| **Body / UI** | **Inter** (400/500/600) | Workhorse for dense catalog data; neutral so it disappears. |
| **Data / labels** | **IBM Plex Mono** (500) | Issue numbers, page counts, dates, file info — the "spine label / catalog card" voice. Monospace makes `#001`, `44 pp`, `2012` tabular and unmistakable. |

Type scale (1.25 ratio, rem):

```
display-xl  3.05  / 1.05  Archivo Expanded 800   — series hero titles
display-l   2.44  / 1.1   Archivo Expanded 700   — page titles
title       1.95  / 1.15  Archivo Expanded 700   — section headers
heading     1.25  / 1.3   Inter 600              — card titles, dialogs
body        1.00  / 1.5   Inter 400/500          — default
small       0.80  / 1.4   Inter 500              — captions
label       0.75  / 1.3   IBM Plex Mono 500, +2% tracking, uppercase — eyebrows, data
```

- Issue numbers in the UI are **always** Plex Mono on a tinted tab (see Signature) — this is
  the consistent "this is a comic's number" signal everywhere it appears.

## 4. Signature element — Registration & the Spine Tab

The one thing ComicHub is remembered by, drawn straight from print production:

1. **Registration corner ticks.** On focus/hover, a cover gets thin crop-mark ticks at its
   four corners (cyan, 1px) — the marks a printer uses to align CMYK plates and trim the page.
   It frames the art like it's on the press bed. Subtle, geometric, meaningful — not decoration.
2. **The spine tab.** Issue/volume numbers sit in a small mono tab clipped to the cover's
   lower-left, echoing a longbox divider / spine label. Unread issues get a magenta tab; read
   issues a hollow/ink tab; in-progress shows a cyan progress underline.
3. **Halftone, used once.** Empty states and the loading splash use a single large Ben-Day
   halftone-dot field (cyan→magenta) — the only place the "comic print" texture appears, so it
   stays special.

Everything else stays quiet: flat ink surfaces, hairline `ink-600` dividers, generous space.

## 5. Layout & structure

- **Shell:** fixed left sidebar (nav), top utility bar (search, scan status, user), fluid
  content. 8px spacing base; section rhythm in multiples of 8.
- **Cover grid:** the primary surface. Covers at their true 2:3-ish aspect, `2px` ink gap,
  no drop shadows (gallery, not skeuomorph). Density slider (S/M/L). Virtualized.
- **Radius:** `6px` on cards/inputs (soft, modern) but covers themselves are square-cornered
  with the registration ticks — the print artifact, not a rounded thumbnail.
- **Series hero:** full-bleed blurred cover wash behind `ink-900/85%` scrim, sharp cover at
  left, Archivo Expanded title, mono metadata row, progress, primary `Read` action.

```
┌ Series hero ───────────────────────────────────────────────┐
│ ░░░ blurred cover wash, ink scrim ░░░                       │
│ ┌──────┐  SAGA                              ⌐ registration  │
│ │cover │  IMAGE · 2012– · 54 issues          ticks on hover │
│ │ 2:3  │  ┌#012┐ mono spine tab                              │
│ └──────┘  12 of 54 read ▁▁▁▁▔▔▔▔  [ Read #13 ]  [ ··· ]     │
└─────────────────────────────────────────────────────────────┘
```

## 6. Core components (Claude Design scope)

- **CoverCard** — image, spine tab (number + read state), progress underline, registration
  ticks on hover/focus, multiselect checkbox. The atom of the whole app.
- **Rail** — horizontally-scrolling CoverCard row with a mono section label (Continue Reading,
  On Deck, Recently Added).
- **FilterBar** — sort + faceted filters (status/genre/tag/publisher/year/format), result count.
- **MetadataPanel / Editor** — read view + RHF/Zod edit form with field-level lock toggles.
- **MatchPicker** — provider candidate list with cover, confidence meter (cyan fill), fields diff.
- **JobIndicator** — top-bar pill + popover with per-job progress (cyan bar), cancel.
- **ScrubBar** (reader) — page thumbnails, current/total in mono, drag-to-seek preview.
- **Toast / EmptyState / Dialog / Toggle / Slider / Badge** — neutral, ink-themed primitives.

## 7. Motion

Restrained and purposeful (`prefers-reduced-motion` fully honored — all of the below collapse
to instant):

- **Cover hover:** registration ticks draw in (120ms), card lifts 2px. No scale-bounce.
- **Page load splash:** halftone dots resolve from scattered → registered grid once (600ms),
  then never again that session.
- **Reader page turn:** none by default in single/double (instant swap is the feature);
  optional 90ms slide for users who want it. Continuous mode is native scroll.
- **Route transitions:** 80ms cross-fade max. Lists/grids never animate item reflow.

## 8. Iconography & imagery

- Line icons, 1.5px stroke, square-ish (Lucide), tinted `paper-400`, cyan when active.
- No stock illustration. The only "art" the app draws is the halftone field and registration
  geometry; everything else is the user's covers.

## 9. Voice & copy

- Sentence case, plain verbs, active voice. Actions keep their name through the flow
  (`Read` → reading; `Match` → "Matched to Comic Vine").
- Name things the collector recognizes: "issues", "series", "reading list", "continue
  reading" — never "media items" or "entities".
- Empty states invite action: _"No libraries yet. Point ComicHub at a folder of comics to
  begin."_ Errors are concrete: _"Couldn't open Saga 001.cbz — the archive looks corrupt.
  [Skip] [Show details]"_

## 10. Theming & tokens

- All values are CSS custom properties under a `[data-theme]` root; Tailwind config maps to
  the tokens above. Light/dark/system, plus an accent override (cyan default; magenta/amber
  alternates) for personalization without breaking contrast rules.
- The reader ships its own background set (black/gray/sepia/white) independent of app theme,
  because reading comfort ≠ app chrome.
