// ComicHub design-system adherence lint.
//
// Ported from the ComicHub Design System's shipped adherence spec
// (`_adherence.oxlintrc.json`, claude.ai/design project
// c0e1bfbe-c5d5-422a-b364-afc6dcdde00b). Keeps app code on the design system:
// no raw hex/px/fonts, primitives imported from the `@comichub/ui` barrel, and
// component `variant`/`size`/`tone`/… values restricted to each component's
// declared contract.
//
// Why ESLint and not oxlint: the spec is built on `no-restricted-syntax` with
// esquery value-regex selectors, which oxlint does not implement (it silently
// ignores them). ESLint runs them for real.
//
// Severity is `warn`, matching the design source — violations surface, builds
// are not blocked. `packages/ui/src` is verbatim design-system source and is
// exempt. When the design system changes a component contract, update PROPS /
// VALUE_RULES below.

import tseslint from 'typescript-eslint';

// Props every DS primitive forwards to its underlying DOM node via {...rest}.
// These are always allowed on top of a component's design-specific props, so
// the contract flags genuinely-unknown props without fighting normal DOM usage.
const PASSTHROUGH = [
  'key',
  'ref',
  'className',
  'style',
  'children',
  'on[A-Z][A-Za-z]*',
  'aria-[a-z-]+',
  'data-[a-z-]+',
  'id',
  'title',
  'role',
  'tabIndex',
  'hidden',
  'dir',
  'lang',
  'draggable',
  'type',
  'name',
  'value',
  'defaultValue',
  'checked',
  'defaultChecked',
  'placeholder',
  'disabled',
  'readOnly',
  'required',
  'autoFocus',
  'autoComplete',
  'min',
  'max',
  'step',
  'htmlFor',
  'href',
  'src',
  'alt',
  'target',
  'rel',
  'form',
  'maxLength',
  'minLength',
  'pattern',
  'inputMode',
  'spellCheck',
];

// Design-specific props per component (the DOM passthrough set is added to each).
const PROPS = {
  Avatar: ['name', 'src', 'size', 'accent'],
  Badge: ['tone', 'mono', 'dot'],
  Button: ['variant', 'size', 'icon', 'iconRight', 'fullWidth'],
  Checkbox: ['checked', 'onChange', 'label', 'indeterminate'],
  CoverCard: [
    'cover',
    'title',
    'subtitle',
    'number',
    'status',
    'progress',
    'size',
    'selectable',
    'selected',
    'onSelect',
  ],
  Dialog: ['open', 'title', 'footer', 'onClose', 'width'],
  EmptyState: ['title', 'action'],
  Icon: ['name', 'size', 'color', 'strokeWidth'],
  IconButton: ['icon', 'variant', 'size', 'active', 'label'],
  Input: ['icon', 'size', 'invalid'],
  ProgressBar: ['value', 'max', 'label', 'tone', 'height', 'showCount'],
  Rail: ['label', 'action'],
  Select: ['size'],
  SidebarItem: ['icon', 'label', 'count', 'active'],
  Slider: ['value', 'min', 'max', 'step', 'onChange'],
  Switch: ['checked', 'onChange', 'label'],
  Tag: ['removable', 'onRemove', 'accent'],
  Toast: ['tone', 'title', 'action', 'onClose'],
  Tooltip: ['label', 'side'],
};

// "<C> doesn't accept that prop" — any attribute not in the declared set ∪ passthrough.
const propRules = Object.entries(PROPS).map(([name, props]) => ({
  selector: `JSXOpeningElement[name.name='${name}'] > JSXAttribute > JSXIdentifier[name!=/^(?:${[...props, ...PASSTHROUGH].join('|')})$/]`,
  message: `<${name}> doesn't accept that prop. Declared props: ${props.join(', ')}.`,
}));

// Enum/value contracts — a prop is set to a string literal off its allowed set.
const valueRules = [
  { c: 'Avatar', p: 'size', allow: 'sm|md|lg' },
  { c: 'Badge', p: 'tone', allow: 'neutral|accent|unread|success|warning|danger' },
  { c: 'Button', p: 'variant', allow: 'primary|secondary|ghost|danger' },
  { c: 'Button', p: 'size', allow: 'sm|md|lg' },
  { c: 'CoverCard', p: 'status', allow: 'unread|reading|read' },
  { c: 'CoverCard', p: 'size', allow: 's|m|l' },
  { c: 'IconButton', p: 'variant', allow: 'ghost|solid|accent' },
  { c: 'IconButton', p: 'size', allow: 'sm|md|lg' },
  { c: 'Input', p: 'size', allow: 'sm|md|lg' },
  { c: 'ProgressBar', p: 'tone', allow: 'accent|unread|success' },
  { c: 'Select', p: 'size', allow: 'sm|md|lg' },
  {
    c: 'Icon',
    p: 'name',
    allow:
      'home|library|list|layers|collection|bookmark|stats|settings|search|x|check|plus|minus|chevron-right|chevron-left|chevron-down|more-horizontal|book-open|filter|sort|grid|columns|sun|moon|alert-triangle|info|trash|edit|star|folder|refresh|user|clock|download|link|maximize|single-page|double-page|fit|zoom-in|zoom-out|fullscreen-exit|direction|book',
  },
  { c: 'Toast', p: 'tone', allow: 'info|success|warning|danger' },
  { c: 'Tooltip', p: 'side', allow: 'top|bottom|left|right' },
].map(({ c, p, allow }) => ({
  selector: `JSXOpeningElement[name.name='${c}'] > JSXAttribute[name.name='${p}'] > Literal[value!=/^(?:${allow})$/]`,
  message: `<${c}> ${p} must be one of ${allow
    .split('|')
    .map((v) => `'${v}'`)
    .join(' | ')}.`,
}));

// Raw token rules — apply to every app file.
const tokenRules = [
  {
    selector: 'Literal[value=/#[0-9a-fA-F]{3,8}\\b/]',
    message: 'Raw hex color — use a design-system color token via var().',
  },
  {
    selector: 'Literal[value=/\\b\\d+px\\b/]',
    message: 'Raw px value — use a design-system spacing token via var().',
  },
  {
    selector:
      'Literal[value=/font-family\\s*:\\s*(?![\'"]?(?:Archivo Expanded|Inter|IBM Plex Mono))/i]',
    message:
      'Font not provided by the design system. Available: Archivo Expanded, Inter, IBM Plex Mono.',
  },
];

const barrelImport = [
  'warn',
  {
    patterns: [
      {
        group: ['@comichub/ui/src/*', '@comichub/ui/dist/*', '**/packages/ui/src/**'],
        message:
          "Import design-system components from the '@comichub/ui' barrel, not its internals.",
      },
    ],
  },
];

// Register the rule names the reader's existing inline disable comments use, so
// ESLint doesn't error on them. The rules stay off — this is an adherence linter,
// not a general React linter.
const reactHooksStub = {
  rules: {
    'exhaustive-deps': { create: () => ({}) },
    'rules-of-hooks': { create: () => ({}) },
  },
};

export default tseslint.config(
  {
    ignores: ['**/dist/**', '**/target/**', '**/src-tauri/**', '**/*.gen.ts', 'packages/**'],
  },
  {
    files: ['apps/**/*.{ts,tsx}'],
    plugins: { 'react-hooks': reactHooksStub },
    languageOptions: {
      parser: tseslint.parser,
      parserOptions: { ecmaFeatures: { jsx: true } },
    },
    linterOptions: { reportUnusedDisableDirectives: 'off' },
    rules: { 'no-restricted-imports': barrelImport },
  },
  {
    // Client uses DS primitives directly (no local shadows) — full contract enforcement.
    files: ['apps/client/**/*.{ts,tsx}'],
    rules: { 'no-restricted-syntax': ['warn', ...tokenRules, ...valueRules, ...propRules] },
  },
  {
    // Reader keeps its own local IconButton (an extra `hint` prop) that collides by
    // name with the DS contract, so it gets token enforcement only. (Its Icon is now
    // the shared DS Icon.)
    files: ['apps/reader/**/*.{ts,tsx}'],
    rules: { 'no-restricted-syntax': ['warn', ...tokenRules] },
  },
);
