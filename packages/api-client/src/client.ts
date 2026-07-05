import type {
  AuthHandshakeResult,
  AuthTokens,
  Bookmark,
  BookCard,
  BookDetail,
  BookManifest,
  Collection,
  CollectionDetail,
  Connection,
  CreateLibraryInput,
  Discover,
  GroupingDetail,
  HealthStatus,
  Job,
  ProviderSettings,
  ProviderSettingsUpdate,
  Library,
  LibraryHealth,
  NextContext,
  PresenceEntry,
  ManualListEntry,
  Progress,
  ProgressBatchItem,
  ProgressBatchResult,
  ProviderStatus,
  ReadingList,
  ReadingListDetail,
  ReadingStats,
  ReadStatus,
  ScanMode,
  SearchResults,
  SearchType,
  SeriesCard,
  SeriesDetail,
  SeriesMatchCandidate,
  ServerInfo,
  ServerStats,
  SmartList,
  SmartListResults,
  SmartRules,
  Tag,
  CreateUserInput,
  UpdateUserInput,
  UserAccount,
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
  private token: string;
  /** Called once on a 401 to obtain a fresh access token (refresh flow). Returns the new
   *  token, or null if refresh failed (the caller should then re-authenticate). */
  private onUnauthorized?: () => Promise<string | null>;

  /** The server base URL this client talks to (no trailing slash) — e.g. for keying
   *  per-server local state like the reader's offline progress queue. */
  get serverUrl(): string {
    return this.baseUrl;
  }

  constructor(connection: Connection) {
    this.baseUrl = connection.baseUrl.replace(/\/$/, '');
    this.token = connection.token;
  }

  /** Replace the bearer access token (after login/refresh). */
  setAccessToken(token: string): void {
    this.token = token;
  }

  /** Register the refresh hook used to recover from a 401 once per request. */
  setUnauthorizedHandler(fn: (() => Promise<string | null>) | undefined): void {
    this.onUnauthorized = fn;
  }

  // ── Authentication (auth mode) ───────────────────────────────────────────────

  /** Exchange credentials for a token pair. Unauthenticated. */
  login(username: string, password: string): Promise<AuthTokens> {
    return this.request<AuthTokens>('POST', '/api/v1/auth/login', {
      auth: false,
      body: { username, password },
    });
  }

  /** Rotate a refresh token for a fresh pair. Unauthenticated. */
  refreshTokens(refresh: string): Promise<AuthTokens> {
    return this.request<AuthTokens>('POST', '/api/v1/auth/refresh', {
      auth: false,
      body: { refresh },
    });
  }

  /** Revoke a refresh-token session. Unauthenticated (the token is the credential). */
  async logout(refresh: string): Promise<void> {
    await this.request<unknown>('POST', '/api/v1/auth/logout', { auth: false, body: { refresh } });
  }

  // ── User management (admin) ──────────────────────────────────────────────────

  async listUsers(): Promise<UserAccount[]> {
    const res = await this.request<{ users: UserAccount[] }>('GET', '/api/v1/users');
    return res.users ?? [];
  }

  createUser(input: CreateUserInput): Promise<UserAccount> {
    return this.request<UserAccount>('POST', '/api/v1/users', { body: input });
  }

  updateUser(id: string, input: UpdateUserInput): Promise<UserAccount> {
    return this.request<UserAccount>('PATCH', `/api/v1/users/${encodeURIComponent(id)}`, {
      body: input,
    });
  }

  async deleteUser(id: string): Promise<void> {
    await this.request<unknown>('DELETE', `/api/v1/users/${encodeURIComponent(id)}`);
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

  /** Deletes a series and re-catalogs its files from scratch (returns the scan job id).
   * Reading-list entries pointing at its issues go stale and re-attach automatically
   * when the rescan re-creates the same files. */
  async rescanSeries(id: string): Promise<string> {
    const res = await this.request<{ jobId: string }>(
      'POST',
      `/api/v1/series/${encodeURIComponent(id)}/rescan`,
    );
    return res.jobId;
  }

  getJob(id: string): Promise<Job> {
    return this.request<Job>('GET', `/api/v1/jobs/${encodeURIComponent(id)}`);
  }

  /** A library's maintenance report: corrupt / orphaned / unmatched / duplicate books. */
  libraryHealth(id: string): Promise<LibraryHealth> {
    return this.request<LibraryHealth>('GET', `/api/v1/libraries/${encodeURIComponent(id)}/health`);
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

  // ── Browse ───────────────────────────────────────────────────────────────────

  async listSeries(libraryId: string): Promise<SeriesCard[]> {
    const res = await this.request<{ items: SeriesCard[] }>(
      'GET',
      `/api/v1/series?library=${encodeURIComponent(libraryId)}`,
    );
    return res.items;
  }

  seriesDetail(id: string): Promise<SeriesDetail> {
    return this.request<SeriesDetail>('GET', `/api/v1/series/${encodeURIComponent(id)}`);
  }

  /** A story arc's header + its issues in reading order. */
  storyArc(seriesId: string, arcId: string): Promise<GroupingDetail> {
    return this.request<GroupingDetail>(
      'GET',
      `/api/v1/series/${encodeURIComponent(seriesId)}/story-arcs/${encodeURIComponent(arcId)}`,
    );
  }

  /** A derived volume's header + its issues (books tagged with that volume number). */
  volume(seriesId: string, volume: string | number): Promise<GroupingDetail> {
    return this.request<GroupingDetail>(
      'GET',
      `/api/v1/series/${encodeURIComponent(seriesId)}/volumes/${encodeURIComponent(String(volume))}`,
    );
  }

  bookDetail(id: string): Promise<BookDetail> {
    return this.request<BookDetail>('GET', `/api/v1/books/${encodeURIComponent(id)}`);
  }

  /** The reader's manifest (page list + reading direction) for a book. */
  manifest(id: string): Promise<BookManifest> {
    return this.request<BookManifest>('GET', `/api/v1/books/${encodeURIComponent(id)}/manifest`);
  }

  async recentBooks(libraryId?: string, limit?: number): Promise<BookCard[]> {
    const p = new URLSearchParams();
    if (libraryId) p.set('library', libraryId);
    if (limit) p.set('limit', String(limit));
    const qs = p.toString();
    const res = await this.request<{ items: BookCard[] }>(
      'GET',
      `/api/v1/books${qs ? `?${qs}` : ''}`,
    );
    return res.items;
  }

  discover(libraryId?: string): Promise<Discover> {
    const qs = libraryId ? `?library=${encodeURIComponent(libraryId)}` : '';
    return this.request<Discover>('GET', `/api/v1/discover${qs}`);
  }

  /** Full-text search across series and books, grouped and ranked best-first. */
  search(
    query: string,
    opts: { type?: SearchType; libraryId?: string; limit?: number } = {},
  ): Promise<SearchResults> {
    const p = new URLSearchParams({ q: query });
    if (opts.type && opts.type !== 'all') p.set('type', opts.type);
    if (opts.libraryId) p.set('library', opts.libraryId);
    if (opts.limit) p.set('limit', String(opts.limit));
    return this.request<SearchResults>('GET', `/api/v1/search?${p.toString()}`);
  }

  // ── Progress ─────────────────────────────────────────────────────────────────

  async continueReading(): Promise<BookCard[]> {
    const res = await this.request<{ items: BookCard[] }>('GET', '/api/v1/me/continue');
    return res.items;
  }

  getProgress(bookId: string): Promise<Progress> {
    return this.request<Progress>('GET', `/api/v1/me/progress/${encodeURIComponent(bookId)}`);
  }

  /**
   * Upserts progress. `updatedAt` (unix ms) stamps when the reading actually happened —
   * set it when replaying offline/standalone progress. Last-writer-wins by `updatedAt`:
   * a stale write is not applied and the response carries the authoritative (newer) row.
   */
  putProgress(
    bookId: string,
    input: { page: number; status?: ReadStatus; device?: string; updatedAt?: number },
  ): Promise<Progress> {
    return this.request<Progress>('PUT', `/api/v1/me/progress/${encodeURIComponent(bookId)}`, {
      body: input,
    });
  }

  /**
   * Bulk progress flush (≤500 items) — how a reader syncs offline progress. Items apply
   * independently; each result reports whether the write won (`applied`) and the
   * authoritative row.
   */
  async batchProgress(items: ProgressBatchItem[]): Promise<ProgressBatchResult[]> {
    const res = await this.request<{ items: ProgressBatchResult[] }>(
      'POST',
      '/api/v1/me/progress/batch',
      { body: { items } },
    );
    return res.items;
  }

  /**
   * Who's reading right now (household presence, most recent first). Live updates arrive
   * on the WS `presence` topic; entries above the viewer's content ceiling are withheld
   * server-side.
   */
  async presence(): Promise<PresenceEntry[]> {
    const res = await this.request<{ items: PresenceEntry[] }>('GET', '/api/v1/presence');
    return res.items;
  }

  /** The user's saved per-book reader overrides (opaque settings object; {} if none). */
  async getReaderPrefs(bookId: string): Promise<Record<string, unknown>> {
    const res = await this.request<{ settings: Record<string, unknown> }>(
      'GET',
      `/api/v1/me/books/${encodeURIComponent(bookId)}/reader-prefs`,
    );
    return res.settings ?? {};
  }

  async putReaderPrefs(bookId: string, settings: Record<string, unknown>): Promise<void> {
    await this.request<unknown>(
      'PUT',
      `/api/v1/me/books/${encodeURIComponent(bookId)}/reader-prefs`,
      { body: { settings } },
    );
  }

  /** The acting user's reading dashboard (headline numbers, month buckets, streaks…). */
  stats(): Promise<ReadingStats> {
    return this.request<ReadingStats>('GET', '/api/v1/me/stats');
  }

  markBook(bookId: string, status: 'read' | 'unread'): Promise<Progress> {
    return this.request<Progress>('POST', `/api/v1/me/books/${encodeURIComponent(bookId)}/mark`, {
      body: { status },
    });
  }

  // ── Bookmarks ────────────────────────────────────────────────────────────────

  /** A book's bookmarks for the user, ordered by page ascending. */
  async listBookmarks(bookId: string): Promise<Bookmark[]> {
    const res = await this.request<{ items: Bookmark[] }>(
      'GET',
      `/api/v1/me/books/${encodeURIComponent(bookId)}/bookmarks`,
    );
    return res.items;
  }

  /** Bookmark a page (idempotent: re-adding a page updates its note). */
  addBookmark(bookId: string, page: number, note = ''): Promise<Bookmark> {
    return this.request<Bookmark>(
      'POST',
      `/api/v1/me/books/${encodeURIComponent(bookId)}/bookmarks`,
      { body: { page, note } },
    );
  }

  /** Replace a bookmark's note. */
  updateBookmark(bookId: string, bookmarkId: string, note: string): Promise<Bookmark> {
    return this.request<Bookmark>(
      'PATCH',
      `/api/v1/me/books/${encodeURIComponent(bookId)}/bookmarks/${encodeURIComponent(bookmarkId)}`,
      { body: { note } },
    );
  }

  /** Remove a bookmark. */
  async removeBookmark(bookId: string, bookmarkId: string): Promise<void> {
    await this.request<unknown>(
      'DELETE',
      `/api/v1/me/books/${encodeURIComponent(bookId)}/bookmarks/${encodeURIComponent(bookmarkId)}`,
    );
  }

  // ── Metadata matching ──────────────────────────────────────────────────────────

  /** Configured metadata providers and whether each has credentials. */
  async providers(): Promise<ProviderStatus[]> {
    const res = await this.request<{ providers: ProviderStatus[] }>('GET', '/api/v1/providers');
    return res.providers;
  }

  /** Provider credential status for the settings screen (secrets never returned). */
  getProviderSettings(): Promise<ProviderSettings> {
    return this.request<ProviderSettings>('GET', '/api/v1/settings/providers');
  }

  /** Save provider credentials; the server reconfigures matching live and returns status. */
  updateProviderSettings(input: ProviderSettingsUpdate): Promise<ProviderSettings> {
    return this.request<ProviderSettings>('PUT', '/api/v1/settings/providers', { body: input });
  }

  /** Ranked provider candidates for a series (defaults the query to the series name). */
  async seriesMatchCandidates(
    seriesId: string,
    opts: { provider?: string; query?: string } = {},
  ): Promise<SeriesMatchCandidate[]> {
    const p = new URLSearchParams();
    if (opts.provider) p.set('provider', opts.provider);
    if (opts.query) p.set('query', opts.query);
    const qs = p.toString();
    const res = await this.request<{ candidates: SeriesMatchCandidate[] }>(
      'GET',
      `/api/v1/series/${encodeURIComponent(seriesId)}/match/candidates${qs ? `?${qs}` : ''}`,
    );
    return res.candidates;
  }

  /** Applies a chosen provider volume to a series (batch); returns the match job id. */
  async applySeriesMatch(
    seriesId: string,
    providerId: string,
    opts: { provider?: string; fields?: string[] } = {},
  ): Promise<string> {
    const res = await this.request<{ jobId: string }>(
      'POST',
      `/api/v1/series/${encodeURIComponent(seriesId)}/match/apply`,
      { body: { providerId, provider: opts.provider, fields: opts.fields } },
    );
    return res.jobId;
  }

  /** Applies a chosen provider issue's metadata to a single book (synchronous). */
  async applyBookMatch(
    bookId: string,
    providerId: string,
    opts: { provider?: string; fields?: string[] } = {},
  ): Promise<void> {
    await this.request<unknown>('POST', `/api/v1/books/${encodeURIComponent(bookId)}/match/apply`, {
      body: { providerId, provider: opts.provider, fields: opts.fields },
    });
  }

  // ── Collections ────────────────────────────────────────────────────────────────

  async listCollections(): Promise<Collection[]> {
    const res = await this.request<{ items: Collection[] }>('GET', '/api/v1/collections');
    return res.items;
  }

  createCollection(input: { name: string; description?: string }): Promise<Collection> {
    return this.request<Collection>('POST', '/api/v1/collections', { body: input });
  }

  collection(id: string): Promise<CollectionDetail> {
    return this.request<CollectionDetail>('GET', `/api/v1/collections/${encodeURIComponent(id)}`);
  }

  updateCollection(
    id: string,
    patch: { name?: string; description?: string; coverBookId?: string },
  ): Promise<Collection> {
    return this.request<Collection>('PATCH', `/api/v1/collections/${encodeURIComponent(id)}`, {
      body: patch,
    });
  }

  async deleteCollection(id: string): Promise<void> {
    await this.request<unknown>('DELETE', `/api/v1/collections/${encodeURIComponent(id)}`);
  }

  async addToCollection(id: string, bookIds: string[]): Promise<void> {
    await this.request<unknown>('POST', `/api/v1/collections/${encodeURIComponent(id)}/items`, {
      body: { bookIds },
    });
  }

  /** Moves bookId before beforeId (omit beforeId to move it to the end). */
  async reorderCollection(id: string, bookId: string, beforeId?: string): Promise<void> {
    await this.request<unknown>(
      'PATCH',
      `/api/v1/collections/${encodeURIComponent(id)}/items/reorder`,
      { body: { bookId, beforeId } },
    );
  }

  async removeFromCollection(id: string, bookId: string): Promise<void> {
    await this.request<unknown>(
      'DELETE',
      `/api/v1/collections/${encodeURIComponent(id)}/items/${encodeURIComponent(bookId)}`,
    );
  }

  // ── Reading lists (per-user) ─────────────────────────────────────────────────────

  async listReadingLists(): Promise<ReadingList[]> {
    const res = await this.request<{ items: ReadingList[] }>('GET', '/api/v1/me/reading-lists');
    return res.items;
  }

  createReadingList(name: string): Promise<ReadingList> {
    return this.request<ReadingList>('POST', '/api/v1/me/reading-lists', { body: { name } });
  }

  readingList(id: string): Promise<ReadingListDetail> {
    return this.request<ReadingListDetail>(
      'GET',
      `/api/v1/me/reading-lists/${encodeURIComponent(id)}`,
    );
  }

  renameReadingList(id: string, name: string): Promise<ReadingList> {
    return this.request<ReadingList>(
      'PATCH',
      `/api/v1/me/reading-lists/${encodeURIComponent(id)}`,
      {
        body: { name },
      },
    );
  }

  async deleteReadingList(id: string): Promise<void> {
    await this.request<unknown>('DELETE', `/api/v1/me/reading-lists/${encodeURIComponent(id)}`);
  }

  /** Make a reading list the active reading queue (clears any previous active one). */
  async setActiveReadingList(id: string): Promise<void> {
    await this.request<unknown>(
      'POST',
      `/api/v1/me/reading-lists/${encodeURIComponent(id)}/active`,
    );
  }

  /** The issue to read after `bookId` — by series order, or the active reading list. */
  async nextBook(bookId: string, context: NextContext = 'series'): Promise<BookCard | null> {
    const res = await this.request<{ book: BookCard | null }>(
      'GET',
      `/api/v1/me/books/${encodeURIComponent(bookId)}/next?context=${context}`,
    );
    return res.book;
  }

  async addToReadingList(id: string, bookIds: string[]): Promise<void> {
    await this.request<unknown>(
      'POST',
      `/api/v1/me/reading-lists/${encodeURIComponent(id)}/items`,
      { body: { bookIds } },
    );
  }

  /** Appends stale placeholder entries for issues not (yet) in the library. */
  async addManualToReadingList(id: string, manual: ManualListEntry[]): Promise<void> {
    await this.request<unknown>(
      'POST',
      `/api/v1/me/reading-lists/${encodeURIComponent(id)}/items`,
      { body: { manual } },
    );
  }

  /** Points an entry (usually a stale placeholder) at a real book. */
  async relinkReadingListItem(id: string, itemId: string, bookId: string): Promise<void> {
    await this.request<unknown>(
      'PATCH',
      `/api/v1/me/reading-lists/${encodeURIComponent(id)}/items/${encodeURIComponent(itemId)}/link`,
      { body: { bookId } },
    );
  }

  /** Moves an entry before another (omit `before` for the end). Refs are entry ids;
   * linked entries also accept their book id. */
  async reorderReadingList(id: string, ref: string, before?: string): Promise<void> {
    await this.request<unknown>(
      'PATCH',
      `/api/v1/me/reading-lists/${encodeURIComponent(id)}/items/reorder`,
      { body: { bookId: ref, beforeId: before } },
    );
  }

  /** Removes an entry by entry id (linked entries also accept their book id). */
  async removeFromReadingList(id: string, ref: string): Promise<void> {
    await this.request<unknown>(
      'DELETE',
      `/api/v1/me/reading-lists/${encodeURIComponent(id)}/items/${encodeURIComponent(ref)}`,
    );
  }

  // ── Tags ───────────────────────────────────────────────────────────────────────

  async listTags(): Promise<Tag[]> {
    const res = await this.request<{ items: Tag[] }>('GET', '/api/v1/tags');
    return res.items;
  }

  createTag(input: { name: string; color?: string }): Promise<Tag> {
    return this.request<Tag>('POST', '/api/v1/tags', { body: input });
  }

  updateTag(id: string, patch: { name?: string; color?: string }): Promise<Tag> {
    return this.request<Tag>('PATCH', `/api/v1/tags/${encodeURIComponent(id)}`, { body: patch });
  }

  async deleteTag(id: string): Promise<void> {
    await this.request<unknown>('DELETE', `/api/v1/tags/${encodeURIComponent(id)}`);
  }

  /** Books carrying a tag (newest-added first). */
  async tagBooks(id: string): Promise<BookCard[]> {
    const res = await this.request<{ items: BookCard[] }>(
      'GET',
      `/api/v1/tags/${encodeURIComponent(id)}/books`,
    );
    return res.items;
  }

  async assignTags(bookId: string, tagIds: string[]): Promise<void> {
    await this.request<unknown>('POST', `/api/v1/books/${encodeURIComponent(bookId)}/tags`, {
      body: { tagIds },
    });
  }

  async unassignTag(bookId: string, tagId: string): Promise<void> {
    await this.request<unknown>(
      'DELETE',
      `/api/v1/books/${encodeURIComponent(bookId)}/tags/${encodeURIComponent(tagId)}`,
    );
  }

  // ── Smart lists (rule-based) ─────────────────────────────────────────────────────

  async listSmartLists(): Promise<SmartList[]> {
    const res = await this.request<{ items: SmartList[] }>('GET', '/api/v1/smart-lists');
    return res.items;
  }

  createSmartList(input: { name: string; rules: SmartRules }): Promise<SmartList> {
    return this.request<SmartList>('POST', '/api/v1/smart-lists', { body: input });
  }

  updateSmartList(id: string, patch: { name?: string; rules?: SmartRules }): Promise<SmartList> {
    return this.request<SmartList>('PATCH', `/api/v1/smart-lists/${encodeURIComponent(id)}`, {
      body: patch,
    });
  }

  async deleteSmartList(id: string): Promise<void> {
    await this.request<unknown>('DELETE', `/api/v1/smart-lists/${encodeURIComponent(id)}`);
  }

  /** Evaluates a smart list and returns it with its matching books. */
  smartListResults(id: string): Promise<SmartListResults> {
    return this.request<SmartListResults>(
      'GET',
      `/api/v1/smart-lists/${encodeURIComponent(id)}/results`,
    );
  }

  private async request<T>(
    method: string,
    path: string,
    opts: { auth?: boolean; body?: unknown; retried?: boolean } = {},
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

    // Auth mode: a 401 on an authenticated call means the access token expired — refresh once
    // (via the registered hook) and retry. The refresh call itself is auth:false, so it never
    // recurses here.
    if (res.status === 401 && opts.auth !== false && !opts.retried && this.onUnauthorized) {
      const fresh = await this.onUnauthorized();
      if (fresh) {
        this.token = fresh;
        return this.request<T>(method, path, { ...opts, retried: true });
      }
    }

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
