# Design handoff — fix: `Select` dropdown popup is unreadable (dark theme)

For **Claude Design** to fix in the **design-system project**
(`c0e1bfbe-c5d5-422a-b364-afc6dcdde00b`), component **`Select`** (synced into the app as
`packages/ui/src/components/forms/Select.jsx`). After the fix lands, the app re-syncs it via
`/design-sync` — please don't hand-edit the synced copy.

## Symptom

When a `Select` is opened on the **dark theme**, the native option list renders **light grey
text on a white background** — effectively unreadable (see the Role dropdown in the user
dialog: "Owner / Admin / Member / Restricted" are barely visible). The **closed** control is
fine; only the open popup is broken.

## Root cause

`Select` styles a wrapper `<div>` (dark `--surface-card` fill, border, chevron) and makes the
inner native `<select>` **`background: transparent`** with `color: var(--text-primary)` (light
on dark). The colour tokens already set `color-scheme: dark` on the root, but with a
**transparent** select background Chromium/WebView2 still paints the native option popup on a
**white** surface, while the `<option>` text inherits the select's light colour → light-on-white.
A component can't style option rows it receives as `children`, so the readable-popup styling
has to come from the design system itself.

## Fix (pick one; A is simplest and global)

**Option A — global option styling in the design-system stylesheet** (e.g. `base.css`),
theme-aware via the existing tokens:

```css
/* Make native dropdown popups (select options) match the theme, not the OS default. */
option,
optgroup {
  background-color: var(--surface-card);
  color: var(--text-primary);
}
```

This fixes **every** `Select` in one place and inherits the light/dark token values
automatically.

**Option B — give the native `<select>` a solid background in the `Select` component**
(`forms/Select.jsx`): change the inner `<select>`'s `background: 'transparent'` to
`background: 'var(--surface-card)'` (the wrapper already paints the same colour, so the closed
control looks identical) and add `color: 'var(--text-primary)'` (already present). A solid
select background gives the popup a readable surface under `color-scheme: dark`.

Either way, keep the closed-control appearance unchanged (ink fill, hairline border, cyan
focus ring, chevron) — only the open option list should change.

## Acceptance

- Open any `Select` on the **dark** theme → options are clearly legible (light text on an
  ink/`--surface-card` background, matching the rest of the UI), with the selected/hovered row
  distinguishable.
- Same check on the **light** theme → dark text on the light surface (no regression).
- The closed control is visually unchanged in both themes.

## Where it shows up in the app (for QA)

Library sort (`Library.tsx`), Smart-list rule builder (`smartlist.tsx`), and the new admin
**Users** dialog (Role + content-rating-ceiling selects, `UsersCard.tsx`). All use the DS
`Select`, so all are fixed by the one change.
