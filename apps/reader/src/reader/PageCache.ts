import type { PageProvider } from '@comichub/reader-core';

export type PageStatus = 'idle' | 'loading' | 'ready' | 'error';

export interface CachedPage {
  idx: number;
  status: PageStatus;
  /** Object URL of the decoded image, present when status === 'ready'. */
  url?: string;
  /** Natural pixel size, read once the image decodes (for fit/zoom math). */
  naturalW?: number;
  naturalH?: number;
}

interface Entry extends CachedPage {
  promise?: Promise<void>;
}

/**
 * PageCache keeps a bounded, LRU-evicted window of fully-decoded pages so a page turn
 * within the window is a synchronous swap, never a load. It lives outside React state:
 * components subscribe and read snapshots, keeping the decode work off the render path.
 *
 * Decoding strategy: fetch the Blob, create an object URL, then force a full decode via
 * Image.decode() before marking the page ready — so the subsequent <img src> paints on
 * the next frame with no flash.
 */
export class PageCache {
  private readonly entries = new Map<number, Entry>();
  /** Stable snapshot per idx so useSyncExternalStore sees a new ref only on real change. */
  private readonly snaps = new Map<number, CachedPage>();
  /** Recency, least-recent first. */
  private readonly lru: number[] = [];
  private readonly listeners = new Set<() => void>();
  private disposed = false;

  constructor(
    private readonly provider: PageProvider,
    /** Max decoded pages held at once; must comfortably exceed the prefetch window. */
    private readonly capacity = 12,
  ) {}

  subscribe = (listener: () => void): (() => void) => {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  };

  private emit(): void {
    for (const l of this.listeners) l();
  }

  /** Snapshot for a page; safe to call from render and stable until the page changes. */
  get(idx: number): CachedPage {
    let snap = this.snaps.get(idx);
    if (!snap) {
      snap = { idx, status: 'idle' };
      this.snaps.set(idx, snap);
    }
    return snap;
  }

  private refreshSnap(entry: Entry): void {
    this.snaps.set(entry.idx, {
      idx: entry.idx,
      status: entry.status,
      url: entry.url,
      naturalW: entry.naturalW,
      naturalH: entry.naturalH,
    });
  }

  private touch(idx: number): void {
    const pos = this.lru.indexOf(idx);
    if (pos >= 0) this.lru.splice(pos, 1);
    this.lru.push(idx);
  }

  /** Kicks off a load for a page if not already loading/ready. Idempotent. */
  ensure(idx: number): void {
    if (this.disposed || idx < 0) return;
    const existing = this.entries.get(idx);
    if (existing && existing.status !== 'idle' && existing.status !== 'error') {
      this.touch(idx);
      return;
    }
    const entry: Entry = { idx, status: 'loading' };
    this.entries.set(idx, entry);
    this.refreshSnap(entry);
    this.touch(idx);
    this.emit();

    entry.promise = this.load(entry).catch(() => {
      entry.status = 'error';
      this.refreshSnap(entry);
      this.emit();
    });
  }

  private async load(entry: Entry): Promise<void> {
    const blob = await this.provider.page(entry.idx);
    if (this.disposed) {
      return;
    }
    const url = URL.createObjectURL(blob);
    // Pre-decode so the swap into <img> is instant.
    try {
      const img = new Image();
      img.src = url;
      await img.decode();
      entry.naturalW = img.naturalWidth;
      entry.naturalH = img.naturalHeight;
    } catch {
      // decode() can reject on some formats; the <img> tag will still try to render.
    }
    if (this.disposed) {
      URL.revokeObjectURL(url);
      return;
    }
    entry.url = url;
    entry.status = 'ready';
    this.refreshSnap(entry);
    this.emit();
  }

  /**
   * Declares the active window [center-behind .. center+ahead]; ensures those pages are
   * loading/ready and evicts everything outside the LRU capacity. Also asks the provider
   * to warm its own cache ahead.
   */
  setWindow(center: number, ahead: number, behind: number, total: number): void {
    const wanted: number[] = [];
    for (let i = center - behind; i <= center + ahead; i++) {
      if (i >= 0 && i < total) wanted.push(i);
    }
    // Load center first, then alternate outward for snappy perceived turns.
    const ordered = [center, ...wanted.filter((i) => i !== center)];
    for (const i of ordered) this.ensure(i);

    if (ahead > 0 && center + 1 < total) {
      this.provider.prefetch(center + 1, Math.min(ahead, total - center - 1));
    }

    this.evict(new Set(wanted));
  }

  private evict(keep: Set<number>): void {
    let changed = false;
    while (this.entries.size > this.capacity && this.lru.length > 0) {
      // Evict the least-recent entry that is not in the protected window.
      const victimPos = this.lru.findIndex((idx) => !keep.has(idx));
      if (victimPos < 0) break;
      const victim = this.lru[victimPos];
      this.lru.splice(victimPos, 1);
      if (victim === undefined) continue;
      const entry = this.entries.get(victim);
      if (entry?.url) URL.revokeObjectURL(entry.url);
      this.entries.delete(victim);
      this.snaps.set(victim, { idx: victim, status: 'idle' });
      changed = true;
    }
    if (changed) this.emit();
  }

  dispose(): void {
    this.disposed = true;
    for (const entry of this.entries.values()) {
      if (entry.url) URL.revokeObjectURL(entry.url);
    }
    this.entries.clear();
    this.snaps.clear();
    this.lru.length = 0;
    this.listeners.clear();
  }
}
