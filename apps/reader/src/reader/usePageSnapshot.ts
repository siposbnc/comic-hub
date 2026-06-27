import { useSyncExternalStore } from 'react';
import { useReaderStore } from './store.js';
import type { CachedPage } from './PageCache.js';

const EMPTY: CachedPage = { idx: -1, status: 'idle' };

/**
 * Subscribes a component to a single page's decode state in the PageCache, without routing
 * image bytes through React/zustand. The cache emits on change; we read a cheap snapshot.
 * This keeps the decode hot path off the React render path (instant turns).
 */
export function usePageSnapshot(idx: number): CachedPage {
  const cache = useReaderStore((s) => s.pages);
  return useSyncExternalStore(
    (cb) => (cache ? cache.subscribe(cb) : () => undefined),
    () => (cache ? cache.get(idx) : EMPTY),
  );
}

/** Subscribes to a scrubber thumbnail's object URL. */
export function useThumbUrl(idx: number): string | undefined {
  const cache = useReaderStore((s) => s.thumbs);
  return useSyncExternalStore(
    (cb) => (cache ? cache.subscribe(cb) : () => undefined),
    () => (cache ? cache.get(idx) : undefined),
  );
}
