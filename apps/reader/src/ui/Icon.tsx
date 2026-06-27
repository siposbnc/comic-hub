import type { SVGProps } from 'react';

/**
 * Icon — minimal inline icon set for the reader toolbar. Local to apps/reader because the
 * shared @comichub/ui package does not yet export an Icon primitive (deferred to packages/*).
 * Strokes use currentColor so buttons control color/contrast.
 */
export type IconName =
  | 'chevron-left'
  | 'chevron-right'
  | 'single-page'
  | 'double-page'
  | 'fit'
  | 'zoom-in'
  | 'zoom-out'
  | 'fullscreen'
  | 'fullscreen-exit'
  | 'direction'
  | 'settings'
  | 'check'
  | 'close'
  | 'book'
  | 'alert'
  | 'restart';

const PATHS: Record<IconName, JSX.Element> = {
  'chevron-left': <path d="M15 6l-6 6 6 6" />,
  'chevron-right': <path d="M9 6l6 6-6 6" />,
  'single-page': <rect x="7" y="4" width="10" height="16" rx="1" />,
  'double-page': (
    <>
      <rect x="3" y="4" width="8" height="16" rx="1" />
      <rect x="13" y="4" width="8" height="16" rx="1" />
    </>
  ),
  fit: (
    <>
      <path d="M4 9V5a1 1 0 011-1h4" />
      <path d="M20 9V5a1 1 0 00-1-1h-4" />
      <path d="M4 15v4a1 1 0 001 1h4" />
      <path d="M20 15v4a1 1 0 01-1 1h-4" />
    </>
  ),
  'zoom-in': (
    <>
      <circle cx="11" cy="11" r="7" />
      <path d="M21 21l-4.3-4.3M11 8v6M8 11h6" />
    </>
  ),
  'zoom-out': (
    <>
      <circle cx="11" cy="11" r="7" />
      <path d="M21 21l-4.3-4.3M8 11h6" />
    </>
  ),
  fullscreen: <path d="M4 9V4h5M20 9V4h-5M4 15v5h5M20 15v5h-5" />,
  'fullscreen-exit': <path d="M9 4v5H4M15 4v5h5M9 20v-5H4M15 20v-5h5" />,
  direction: <path d="M4 12h16M12 6l8 6-8 6M8 6L4 12l4 6" />,
  settings: (
    <>
      <circle cx="12" cy="12" r="3" />
      <path d="M19.4 15a1.7 1.7 0 00.3 1.9l.1.1a2 2 0 11-2.8 2.8l-.1-.1a1.7 1.7 0 00-1.9-.3 1.7 1.7 0 00-1 1.5V21a2 2 0 11-4 0v-.1a1.7 1.7 0 00-1.1-1.5 1.7 1.7 0 00-1.9.3l-.1.1a2 2 0 11-2.8-2.8l.1-.1a1.7 1.7 0 00.3-1.9 1.7 1.7 0 00-1.5-1H3a2 2 0 110-4h.1a1.7 1.7 0 001.5-1.1 1.7 1.7 0 00-.3-1.9l-.1-.1a2 2 0 112.8-2.8l.1.1a1.7 1.7 0 001.9.3H9a1.7 1.7 0 001-1.5V3a2 2 0 114 0v.1a1.7 1.7 0 001 1.5 1.7 1.7 0 001.9-.3l.1-.1a2 2 0 112.8 2.8l-.1.1a1.7 1.7 0 00-.3 1.9V9a1.7 1.7 0 001.5 1H21a2 2 0 110 4h-.1a1.7 1.7 0 00-1.5 1z" />
    </>
  ),
  check: <path d="M5 13l4 4L19 7" />,
  close: <path d="M6 6l12 12M18 6L6 18" />,
  book: <path d="M4 5a1 1 0 011-1h6v16H5a1 1 0 01-1-1zM20 5a1 1 0 00-1-1h-6v16h6a1 1 0 001-1z" />,
  alert: (
    <>
      <path d="M12 3l9 16H3z" />
      <path d="M12 10v4M12 17h.01" />
    </>
  ),
  restart: (
    <>
      <path d="M3 12a9 9 0 109-9 9 9 0 00-6.4 2.6L3 8" />
      <path d="M3 4v4h4" />
    </>
  ),
};

export interface IconProps extends Omit<SVGProps<SVGSVGElement>, 'name'> {
  name: IconName;
  size?: number;
}

export function Icon({ name, size = 20, ...props }: IconProps) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.7}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      focusable="false"
      {...props}
    >
      {PATHS[name]}
    </svg>
  );
}
