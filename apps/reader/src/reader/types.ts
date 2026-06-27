import type { ReadingDirection } from '@comichub/reader-core';

/** How pages are laid out on screen. */
export type LayoutMode = 'single' | 'double';

/** How a page is sized within the reading area. */
export type FitMode = 'width' | 'height' | 'screen' | 'original' | 'smart';

/** Background of the reading gallery (visual comfort, docs/06-reader.md §3.6). */
export type ReaderBackground = 'black' | 'gray' | 'sepia' | 'white';

export interface ReaderSettings {
  layout: LayoutMode;
  fit: FitMode;
  direction: ReadingDirection;
  background: ReaderBackground;
  /** Treat the first page as a lone cover so spreads pair correctly in double mode. */
  coverAlone: boolean;
}

export const DEFAULT_SETTINGS: ReaderSettings = {
  layout: 'single',
  fit: 'screen',
  direction: 'ltr',
  background: 'black',
  coverAlone: true,
};
