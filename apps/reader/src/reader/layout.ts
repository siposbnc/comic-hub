import type { Manifest, PageMeta } from '@comichub/reader-core';
import type { LayoutMode } from './types.js';

/** A unit shown together on screen: one page (single / wide) or a left+right pair. */
export type Spread = number[];

function isWide(page: PageMeta | undefined): boolean {
  if (!page) return false;
  if (page.double) return true;
  // Fallback heuristic when the manifest omits the flag: landscape pages stand alone.
  return page.w > 0 && page.h > 0 && page.w / page.h > 1.2;
}

/**
 * Groups pages into spreads for the given layout. In double mode a wide page or the lone
 * cover (when coverAlone) occupies its own spread; remaining pages pair up in reading
 * order. Pairing is reading-order based; the renderer flips left/right for RTL.
 */
export function buildSpreads(
  manifest: Manifest,
  layout: LayoutMode,
  coverAlone: boolean,
): Spread[] {
  const count = manifest.pageCount;
  // Single and continuous are both one page per unit; only double pairs pages.
  if (layout !== 'double') {
    return Array.from({ length: count }, (_, i) => [i]);
  }

  const byIdx = new Map(manifest.pages.map((p) => [p.idx, p]));
  const spreads: Spread[] = [];
  let i = 0;
  while (i < count) {
    if (i === 0 && coverAlone) {
      spreads.push([0]);
      i = 1;
      continue;
    }
    const page = byIdx.get(i);
    if (isWide(page)) {
      spreads.push([i]);
      i += 1;
      continue;
    }
    const next = byIdx.get(i + 1);
    if (i + 1 < count && !isWide(next)) {
      spreads.push([i, i + 1]);
      i += 2;
    } else {
      spreads.push([i]);
      i += 1;
    }
  }
  return spreads;
}

/** Index of the spread that contains `page`. */
export function spreadIndexOf(spreads: Spread[], page: number): number {
  const found = spreads.findIndex((s) => s.includes(page));
  return found >= 0 ? found : 0;
}
