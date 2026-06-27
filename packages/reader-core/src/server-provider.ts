import type { ComicHubClient } from '@comichub/api-client';

import type { Manifest, PageProvider, PageOpts, Progress } from './index.js';

/**
 * ServerPageProvider streams a book's pages from a ComicHub server and syncs progress
 * over REST (the reader's connected mode; also reused by the client to prefetch on
 * hover). Page bytes are fetched via the content-addressed image URLs so the browser
 * cache does the heavy lifting. See docs/06-reader.md §4.
 */
export class ServerPageProvider implements PageProvider {
  constructor(
    private readonly client: ComicHubClient,
    private readonly bookId: string,
    private readonly device = 'reader',
  ) {}

  async manifest(): Promise<Manifest> {
    const m = await this.client.manifest(this.bookId);
    return {
      bookId: m.bookId,
      pageCount: m.pageCount,
      readingDir: m.readingDir,
      pages: m.pages.map((p) => ({
        idx: p.idx,
        w: p.w,
        h: p.h,
        type: p.type as Manifest['pages'][number]['type'],
        double: p.double,
      })),
    };
  }

  async page(idx: number, opts?: PageOpts): Promise<Blob> {
    const url = this.client.pageUrl(this.bookId, idx, {
      width: opts?.w,
      // The pure-Go server emits JPEG/PNG; webp/avif requests fall back to JPEG. We
      // only ever ask for JPEG (or the untouched original when no format is given).
      format: opts?.fmt ? 'jpeg' : undefined,
      quality: opts?.q,
    });
    return fetchBlob(url);
  }

  async thumb(idx: number): Promise<Blob> {
    return fetchBlob(this.client.pageThumbUrl(this.bookId, idx));
  }

  prefetch(from: number, count: number): void {
    // Fire-and-forget: warm the server's page cache ahead of the reader.
    void this.client.prefetch(this.bookId, from, count).catch(() => {});
  }

  saveProgress(progress: Progress): void {
    void this.client
      .putProgress(this.bookId, {
        page: progress.page,
        status: progress.status,
        device: progress.device ?? this.device,
      })
      .catch(() => {});
  }

  async restoreProgress(): Promise<Progress | null> {
    try {
      const p = await this.client.getProgress(this.bookId);
      return { bookId: p.bookId, page: p.page, status: p.status, updatedAt: p.updatedAt };
    } catch {
      return null; // no progress yet (404) or offline
    }
  }
}

async function fetchBlob(url: string): Promise<Blob> {
  const res = await fetch(url);
  if (!res.ok) {
    throw new Error(`page fetch failed: HTTP ${res.status}`);
  }
  return res.blob();
}
