/**
 * reader-core holds the reading logic shared between the reader app's two modes
 * (standalone vs connected) and reused by the client for prefetch-on-hover. The
 * PageProvider abstraction is the seam: ServerPageProvider streams from a server,
 * LocalPageProvider reads an archive on disk via the Tauri core. See docs/06-reader.md.
 */

export type ReadingDirection = 'ltr' | 'rtl';

export type PageType = 'FrontCover' | 'Story' | 'Advertisement' | 'BackCover' | 'Other';

export interface PageMeta {
  idx: number;
  w: number;
  h: number;
  type?: PageType;
  double?: boolean;
}

export interface Manifest {
  bookId: string;
  pageCount: number;
  readingDir: ReadingDirection;
  pages: PageMeta[];
}

export type ReadStatus = 'unread' | 'in_progress' | 'read';

export interface Progress {
  bookId: string;
  page: number;
  status: ReadStatus;
  updatedAt: number;
  device?: string;
}

export interface PageOpts {
  /** Target width for server-side resize; omit for original resolution. */
  w?: number;
  fmt?: 'webp' | 'avif' | 'jpeg';
  q?: number;
}

/** Source of pages for the reader. Implemented per operating mode. */
export interface PageProvider {
  manifest(): Promise<Manifest>;
  page(idx: number, opts?: PageOpts): Promise<Blob>;
  thumb(idx: number): Promise<Blob>;
  /** Hint that pages [from, from+count) will be needed soon. */
  prefetch(from: number, count: number): void;
  /** Persist progress (debounced by the caller). */
  saveProgress(progress: Progress): void;
  restoreProgress(): Promise<Progress | null>;
}

export const DEFAULT_PREFETCH_AHEAD = 4;
export const DEFAULT_PREFETCH_BEHIND = 1;
