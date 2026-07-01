import type { ComicHubClient, ProgressBatchItem } from '@comichub/api-client';

import type { Manifest, PageProvider, PageOpts, Progress, ReadStatus } from './index.js';

/**
 * ServerPageProvider streams a book's pages from a ComicHub server and syncs progress
 * over REST (the reader's connected mode; also reused by the client to prefetch on
 * hover). Page bytes are fetched via the content-addressed image URLs so the browser
 * cache does the heavy lifting. See docs/06-reader.md §4.
 *
 * Progress writes that fail (server unreachable, session expired) are queued in
 * localStorage — keyed per server — and flushed through POST /me/progress/batch with
 * their original timestamps the next time the server answers (ADR-008: the server
 * resolves conflicts last-writer-wins by updatedAt, so a flush can never clobber newer
 * progress from another device).
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
        updatedAt: progress.updatedAt,
      })
      .then(() => {
        // The server is answering — drain anything queued while it wasn't.
        void flushPendingProgress(this.client);
      })
      .catch(() => {
        enqueuePendingProgress(this.client.serverUrl, {
          bookId: this.bookId,
          page: progress.page,
          status: progress.status,
          device: progress.device ?? this.device,
          updatedAt: progress.updatedAt ?? Date.now(),
        });
      });
  }

  async restoreProgress(): Promise<Progress | null> {
    // Flush first so the place we restore reflects what was read offline.
    await flushPendingProgress(this.client);
    try {
      const p = await this.client.getProgress(this.bookId);
      return { bookId: p.bookId, page: p.page, status: p.status, updatedAt: p.updatedAt };
    } catch {
      return null; // no progress yet (404) or offline
    }
  }
}

/* ── Offline progress queue ──────────────────────────────────────────────────────── */

interface QueuedWrite {
  bookId: string;
  page: number;
  status: ReadStatus;
  device?: string;
  updatedAt: number;
}

const QUEUE_PREFIX = 'comichub.reader.pending-progress:';

function queueKey(serverUrl: string): string {
  return QUEUE_PREFIX + serverUrl;
}

function hasStorage(): boolean {
  return typeof localStorage !== 'undefined';
}

function readQueue(serverUrl: string): Record<string, QueuedWrite> {
  if (!hasStorage()) return {};
  try {
    return JSON.parse(localStorage.getItem(queueKey(serverUrl)) ?? '{}') as Record<
      string,
      QueuedWrite
    >;
  } catch {
    return {};
  }
}

function writeQueue(serverUrl: string, queue: Record<string, QueuedWrite>): void {
  if (!hasStorage()) return;
  if (Object.keys(queue).length === 0) {
    localStorage.removeItem(queueKey(serverUrl));
    return;
  }
  localStorage.setItem(queueKey(serverUrl), JSON.stringify(queue));
}

/** Records a failed write, coalescing per book (only the latest place matters). */
export function enqueuePendingProgress(serverUrl: string, write: QueuedWrite): void {
  const queue = readQueue(serverUrl);
  const existing = queue[write.bookId];
  if (existing && existing.updatedAt > write.updatedAt) return;
  queue[write.bookId] = write;
  writeQueue(serverUrl, queue);
}

let flushing = false;

/**
 * Sends every queued write for this client's server as one batch. Entries are dropped
 * once the server has ruled on them — applied, superseded by newer progress from
 * another device (`applied: false`), or unresolvable (book deleted). Network failure
 * keeps the queue intact for the next attempt. Safe to call opportunistically.
 */
export async function flushPendingProgress(client: ComicHubClient): Promise<void> {
  if (flushing || !hasStorage()) return;
  const serverUrl = client.serverUrl;
  const queue = readQueue(serverUrl);
  const writes = Object.values(queue);
  if (writes.length === 0) return;
  flushing = true;
  try {
    const items: ProgressBatchItem[] = writes.map((w) => ({
      bookId: w.bookId,
      page: w.page,
      status: w.status,
      device: w.device,
      updatedAt: w.updatedAt,
    }));
    await client.batchProgress(items);
    // The server answered: every sent entry is settled. Only drop what we sent — a
    // page turned while the request was in flight stays queued.
    const after = readQueue(serverUrl);
    for (const w of writes) {
      const cur = after[w.bookId];
      if (cur && cur.updatedAt <= w.updatedAt) delete after[w.bookId];
    }
    writeQueue(serverUrl, after);
  } catch {
    // Still unreachable — keep the queue for next time.
  } finally {
    flushing = false;
  }
}

async function fetchBlob(url: string): Promise<Blob> {
  const res = await fetch(url);
  if (!res.ok) {
    throw new Error(`page fetch failed: HTTP ${res.status}`);
  }
  return res.blob();
}
