import type { PageProvider } from '@comichub/reader-core';

/**
 * Loads scrubber thumbnails on demand. Thumbnails are tiny, so they are kept for the
 * session rather than evicted; only requested when a strip cell becomes visible/hovered.
 */
export class ThumbCache {
  private readonly urls = new Map<number, string>();
  private readonly inflight = new Map<number, Promise<void>>();
  private readonly listeners = new Set<() => void>();
  private disposed = false;

  constructor(private readonly provider: PageProvider) {}

  subscribe = (listener: () => void): (() => void) => {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  };

  private emit(): void {
    for (const l of this.listeners) l();
  }

  get(idx: number): string | undefined {
    return this.urls.get(idx);
  }

  ensure(idx: number): void {
    if (this.disposed || idx < 0 || this.urls.has(idx) || this.inflight.has(idx)) return;
    const p = this.provider
      .thumb(idx)
      .then((blob) => {
        if (this.disposed) return;
        this.urls.set(idx, URL.createObjectURL(blob));
        this.emit();
      })
      .catch(() => undefined)
      .finally(() => {
        this.inflight.delete(idx);
      });
    this.inflight.set(idx, p);
  }

  dispose(): void {
    this.disposed = true;
    for (const url of this.urls.values()) URL.revokeObjectURL(url);
    this.urls.clear();
    this.inflight.clear();
    this.listeners.clear();
  }
}
