import type {
  Manifest,
  PageOpts,
  PageProvider,
  Progress,
  ReadStatus,
} from '@comichub/reader-core';

export interface ServerProviderConfig {
  /** Server base URL, e.g. http://127.0.0.1:8099 (no trailing slash required). */
  baseUrl: string;
  /** Bearer token (loopback token in embedded mode, scoped token via deep link). */
  token: string;
  bookId: string;
  /** Device label recorded on progress writes for cross-device reconciliation. */
  device?: string;
}

interface WireProgress {
  page: number;
  status?: ReadStatus;
  percent?: number;
  updatedAt?: string;
  device?: string;
}

/**
 * ServerPageProvider — connected mode. Streams pages and syncs progress against the
 * REST surface in docs/03-api.md.
 *
 * NOTE (deferred to packages/*): canonically this belongs in @comichub/reader-core
 * built on an extended ComicHubClient, but the shipped ComicHubClient exposes no
 * book/page/progress methods and this subagent may not edit packages/*. It is kept
 * self-contained here so connected mode works end-to-end; lift it into reader-core
 * once the client gains those endpoints.
 */
export class ServerPageProvider implements PageProvider {
  private readonly base: string;
  private readonly token: string;
  private readonly bookId: string;
  private readonly device?: string;

  constructor(config: ServerProviderConfig) {
    this.base = config.baseUrl.replace(/\/$/, '');
    this.token = config.token;
    this.bookId = config.bookId;
    this.device = config.device;
  }

  private url(path: string): string {
    return `${this.base}/api/v1/books/${encodeURIComponent(this.bookId)}${path}`;
  }

  private authHeaders(extra?: Record<string, string>): Record<string, string> {
    const headers: Record<string, string> = { ...extra };
    if (this.token) {
      headers.Authorization = `Bearer ${this.token}`;
    }
    return headers;
  }

  async manifest(): Promise<Manifest> {
    const res = await fetch(this.url('/manifest'), {
      headers: this.authHeaders({ Accept: 'application/json' }),
    });
    if (!res.ok) {
      throw new Error(`Failed to load manifest (${res.status})`);
    }
    return (await res.json()) as Manifest;
  }

  async page(idx: number, opts?: PageOpts): Promise<Blob> {
    const params = new URLSearchParams();
    if (opts?.w) params.set('w', String(opts.w));
    if (opts?.fmt) params.set('fmt', opts.fmt);
    if (opts?.q) params.set('q', String(opts.q));
    const query = params.toString();
    const res = await fetch(this.url(`/pages/${idx}${query ? `?${query}` : ''}`), {
      headers: this.authHeaders(),
    });
    if (!res.ok) {
      throw new Error(`Failed to load page ${idx} (${res.status})`);
    }
    return res.blob();
  }

  async thumb(idx: number): Promise<Blob> {
    const res = await fetch(this.url(`/pages/${idx}/thumb`), {
      headers: this.authHeaders(),
    });
    if (!res.ok) {
      throw new Error(`Failed to load thumbnail ${idx} (${res.status})`);
    }
    return res.blob();
  }

  prefetch(from: number, count: number): void {
    // Fire-and-forget cache warm-up; failure here never affects reading.
    void fetch(this.url('/prefetch'), {
      method: 'POST',
      headers: this.authHeaders({ 'Content-Type': 'application/json' }),
      body: JSON.stringify({ from, count }),
    }).catch(() => undefined);
  }

  saveProgress(progress: Progress): void {
    void fetch(`${this.base}/api/v1/me/progress/${encodeURIComponent(this.bookId)}`, {
      method: 'PUT',
      headers: this.authHeaders({ 'Content-Type': 'application/json' }),
      body: JSON.stringify({
        page: progress.page,
        status: progress.status,
        device: progress.device ?? this.device,
      }),
    }).catch(() => undefined);
  }

  async restoreProgress(): Promise<Progress | null> {
    try {
      const res = await fetch(
        `${this.base}/api/v1/me/progress/${encodeURIComponent(this.bookId)}`,
        { headers: this.authHeaders({ Accept: 'application/json' }) },
      );
      if (res.status === 404) return null;
      if (!res.ok) return null;
      const data = (await res.json()) as WireProgress;
      return {
        bookId: this.bookId,
        page: data.page ?? 0,
        status: data.status ?? 'in_progress',
        updatedAt: data.updatedAt ? Date.parse(data.updatedAt) : Date.now(),
        device: data.device,
      };
    } catch {
      return null;
    }
  }
}
