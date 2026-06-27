import type {
  AuthHandshakeResult,
  Connection,
  CreateLibraryInput,
  HealthStatus,
  Job,
  Library,
  ScanMode,
  ServerInfo,
  ServerStats,
} from './types.js';

/** Thrown when the server returns a non-2xx response. */
export class ApiError extends Error {
  constructor(
    public readonly status: number,
    public readonly code: string,
    message: string,
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

/**
 * ComicHubClient is the typed entry point both the client and reader use to talk to a
 * server, regardless of deployment mode. It carries the bearer token from the
 * connection and exposes one method per endpoint group (see docs/03-api.md).
 */
export class ComicHubClient {
  private readonly baseUrl: string;
  private readonly token: string;

  constructor(connection: Connection) {
    this.baseUrl = connection.baseUrl.replace(/\/$/, '');
    this.token = connection.token;
  }

  /** Liveness — unauthenticated. */
  health(): Promise<HealthStatus> {
    return this.request<HealthStatus>('GET', '/healthz', { auth: false });
  }

  serverInfo(): Promise<ServerInfo> {
    return this.request<ServerInfo>('GET', '/api/v1/server/info');
  }

  serverStats(): Promise<ServerStats> {
    return this.request<ServerStats>('GET', '/api/v1/server/stats');
  }

  authHandshake(): Promise<AuthHandshakeResult> {
    return this.request<AuthHandshakeResult>('GET', '/api/v1/auth/handshake');
  }

  /** Asks an embedded server to shut down gracefully. */
  async shutdown(): Promise<void> {
    await this.request<unknown>('POST', '/api/v1/admin/shutdown');
  }

  // ── Libraries ──────────────────────────────────────────────────────────────

  async listLibraries(): Promise<Library[]> {
    const res = await this.request<{ items: Library[] }>('GET', '/api/v1/libraries');
    return res.items;
  }

  createLibrary(input: CreateLibraryInput): Promise<Library> {
    return this.request<Library>('POST', '/api/v1/libraries', { body: input });
  }

  getLibrary(id: string): Promise<Library> {
    return this.request<Library>('GET', `/api/v1/libraries/${encodeURIComponent(id)}`);
  }

  async deleteLibrary(id: string): Promise<void> {
    await this.request<unknown>('DELETE', `/api/v1/libraries/${encodeURIComponent(id)}`);
  }

  // ── Scanning & jobs ──────────────────────────────────────────────────────────

  /** Starts a scan and returns the job id to poll (or follow over WS once available). */
  async scanLibrary(id: string, mode: ScanMode = 'incremental'): Promise<string> {
    const res = await this.request<{ jobId: string }>(
      'POST',
      `/api/v1/libraries/${encodeURIComponent(id)}/scan`,
      { body: { mode } },
    );
    return res.jobId;
  }

  cancelScan(id: string): Promise<{ canceled: number }> {
    return this.request<{ canceled: number }>(
      'POST',
      `/api/v1/libraries/${encodeURIComponent(id)}/scan/cancel`,
    );
  }

  getJob(id: string): Promise<Job> {
    return this.request<Job>('GET', `/api/v1/jobs/${encodeURIComponent(id)}`);
  }

  // ── Images ───────────────────────────────────────────────────────────────────
  // URL builders for <img src> — image endpoints accept the bearer token as a query
  // param so plain <img> tags authenticate without headers (docs/03-api.md §11).

  /** Absolute URL for a book's cover thumbnail (optional width). */
  coverUrl(bookId: string, width?: number): string {
    const w = width ? `&w=${width}` : '';
    return `${this.baseUrl}/api/v1/books/${encodeURIComponent(bookId)}/cover?token=${encodeURIComponent(this.token)}${w}`;
  }

  /** Absolute URL for a full page image, with optional server-side resize/transcode. */
  pageUrl(
    bookId: string,
    idx: number,
    opts: { width?: number; format?: 'jpeg' | 'png'; quality?: number } = {},
  ): string {
    const p = new URLSearchParams({ token: this.token });
    if (opts.width) p.set('w', String(opts.width));
    if (opts.format) p.set('fmt', opts.format);
    if (opts.quality) p.set('q', String(opts.quality));
    return `${this.baseUrl}/api/v1/books/${encodeURIComponent(bookId)}/pages/${idx}?${p.toString()}`;
  }

  /** Absolute URL for a page's scrubber thumbnail. */
  pageThumbUrl(bookId: string, idx: number): string {
    return `${this.baseUrl}/api/v1/books/${encodeURIComponent(bookId)}/pages/${idx}/thumb?token=${encodeURIComponent(this.token)}`;
  }

  /** Hints the server to warm pages [from, from+count) into its cache. */
  async prefetch(bookId: string, from: number, count: number): Promise<void> {
    await this.request<unknown>('POST', `/api/v1/books/${encodeURIComponent(bookId)}/prefetch`, {
      body: { from, count },
    });
  }

  private async request<T>(
    method: string,
    path: string,
    opts: { auth?: boolean; body?: unknown } = {},
  ): Promise<T> {
    const headers: Record<string, string> = { Accept: 'application/json' };
    if (opts.auth !== false && this.token) {
      headers.Authorization = `Bearer ${this.token}`;
    }
    if (opts.body !== undefined) {
      headers['Content-Type'] = 'application/json';
    }

    const res = await fetch(`${this.baseUrl}${path}`, {
      method,
      headers,
      body: opts.body !== undefined ? JSON.stringify(opts.body) : undefined,
    });

    if (!res.ok) {
      let code = 'http_error';
      let message = res.statusText;
      try {
        const data = (await res.json()) as { error?: { code: string; message: string } };
        if (data.error) {
          code = data.error.code;
          message = data.error.message;
        }
      } catch {
        // non-JSON error body; keep defaults
      }
      throw new ApiError(res.status, code, message);
    }

    if (res.status === 204) {
      return undefined as T;
    }
    return (await res.json()) as T;
  }
}
