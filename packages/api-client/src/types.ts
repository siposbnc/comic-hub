// Wire types shared by the client and reader. Mirrors docs/03-api.md. These are kept
// hand-written in Phase 0; later they can be generated from the server's OpenAPI spec.

export type ServerMode = 'embedded' | 'server';

export interface HealthStatus {
  status: 'ok';
  version: string;
}

export interface ServerCapabilities {
  avif: boolean;
  pdf: boolean;
  epub: boolean;
  multiuser: boolean;
}

export interface ServerInfo {
  name: string;
  version: string;
  commit: string;
  mode: ServerMode;
  capabilities: ServerCapabilities;
}

export interface ServerStats {
  libraries: number;
  series: number;
  books: number;
}

export type UserRole = 'owner' | 'admin' | 'member' | 'restricted';

export interface User {
  id: string;
  username: string;
  displayName: string;
  role: UserRole;
}

export interface AuthHandshakeResult {
  mode: ServerMode;
  user: User;
}

export interface ApiErrorBody {
  error: {
    code: string;
    message: string;
  };
}

export type LibraryKind = 'comic' | 'manga';

/** A named set of root folders ComicHub scans (see docs/02-data-model.md). */
export interface Library {
  id: string;
  name: string;
  kind: LibraryKind;
  roots: string[];
  createdAt: number;
  updatedAt: number;
}

/** Request body for creating a library. */
export interface CreateLibraryInput {
  name: string;
  kind?: LibraryKind;
  roots: string[];
}

export type ScanMode = 'full' | 'incremental';

export type JobState = 'queued' | 'running' | 'done' | 'failed' | 'canceled';

/** A background job (scan, thumbnail, …); poll via getJob until WS jobs topic lands. */
export interface Job {
  id: string;
  type: string;
  state: JobState;
  progress: number;
  total: number;
  done: number;
  error?: string;
  createdAt: number;
  startedAt?: number;
  finishedAt?: number;
}

export type ReadStatus = 'unread' | 'in_progress' | 'read';

/** Progress shown on a card (no book/page-count context). */
export interface ProgressSummary {
  page: number;
  status: ReadStatus;
  percent: number;
  updatedAt: number;
}

/** Full progress for a single book (from the /me/progress endpoints). */
export interface Progress {
  bookId: string;
  page: number;
  pageCount: number;
  status: ReadStatus;
  percent: number;
  updatedAt: number;
}

/** A series in the library grid. */
export interface SeriesCard {
  id: string;
  name: string;
  year?: number;
  readingDir: 'ltr' | 'rtl';
  bookCount: number;
  readCount: number;
  coverBookId?: string;
}

/** A book in a list/rail/grid. */
export interface BookCard {
  id: string;
  seriesId: string;
  number?: string;
  title?: string;
  pageCount: number;
  format: string;
  isCorrupt?: boolean;
  progress?: ProgressSummary;
}

export interface SeriesDetail {
  id: string;
  name: string;
  year?: number;
  publisher?: string;
  summary?: string;
  readingDir: 'ltr' | 'rtl';
  bookCount: number;
  readCount: number;
  books: BookCard[];
}

export interface BookDetail {
  id: string;
  seriesId: string;
  seriesName: string;
  number?: string;
  title?: string;
  volume?: number;
  pageCount: number;
  format: string;
  readingDir: 'ltr' | 'rtl';
  releaseDate?: number;
  ageRating?: string;
  language?: string;
  summary?: string;
  isCorrupt?: boolean;
  progress?: ProgressSummary;
  /** Credits keyed by role (writer, penciler, …) — from an online match. */
  credits?: Record<string, string[]>;
  genres?: string[];
  characters?: string[];
  /** User-applied organizational tags. */
  tags?: Tag[];
  /** Collections this book already belongs to. */
  collectionIds?: string[];
  /** The acting user's reading lists this book already belongs to. */
  readingListIds?: string[];
}

/** A free-form label applied across books. */
export interface Tag {
  id: string;
  name: string;
  color?: string;
  bookCount: number;
}

/** Fields a smart-list rule can test. */
export type SmartField = 'tag' | 'series' | 'publisher' | 'format' | 'ageRating' | 'readStatus';
/** Operators a smart-list rule can use. */
export type SmartOp = 'is' | 'isNot' | 'contains';

export interface SmartRule {
  field: SmartField;
  op: SmartOp;
  value: string;
}

export interface SmartRules {
  /** How to combine rules: all = AND, any = OR. */
  match: 'all' | 'any';
  rules: SmartRule[];
}

/** A saved rule set whose contents are evaluated on demand. */
export interface SmartList {
  id: string;
  name: string;
  rules: SmartRules;
  /** Result count for the acting user. */
  bookCount: number;
  createdAt: number;
  updatedAt: number;
}

/** A smart list plus its evaluated books. */
export interface SmartListResults {
  smartList: SmartList;
  books: BookCard[];
}

/** A problem book in a library health report. */
export interface HealthItem {
  id: string;
  seriesId: string;
  number?: string;
  title?: string;
  path: string;
}

export interface DuplicateGroup {
  contentHash: string;
  books: HealthItem[];
}

export interface HealthCounts {
  books: number;
  corrupt: number;
  orphans: number;
  unmatched: number;
  duplicateGroups: number;
}

/** A library's maintenance snapshot. */
export interface LibraryHealth {
  libraryId: string;
  counts: HealthCounts;
  corrupt: HealthItem[];
  orphans: HealthItem[];
  unmatched: HealthItem[];
  duplicates: DuplicateGroup[];
}

/** The Home feed. */
/** The next issue to read from the active reading list. */
export interface NextUp {
  book: BookCard;
  listId: string;
  listName: string;
}

export interface Discover {
  continueReading: BookCard[];
  recentlyAdded: BookCard[];
  nextUp?: NextUp;
}

/** Where to look for the issue after the current one. */
export type NextContext = 'series' | 'readingList';

/** A series matched by full-text search. */
export interface SeriesHit {
  id: string;
  name: string;
  year?: number;
  coverBookId?: string;
}

/** A book matched by full-text search (carries its series name for display). */
export interface BookHit {
  id: string;
  seriesId: string;
  seriesName?: string;
  number?: string;
  title?: string;
  format: string;
}

/** Grouped, ranked results from `GET /search`. */
export interface SearchResults {
  query: string;
  series: SeriesHit[];
  books: BookHit[];
}

/** What to search for; omit for all. */
export type SearchType = 'all' | 'series' | 'book';

/** A curated, ordered, shared shelf of books. */
export interface Collection {
  id: string;
  name: string;
  description?: string;
  coverBookId?: string;
  bookCount: number;
  createdAt: number;
  updatedAt: number;
}

/** A collection plus its books in display order. */
export interface CollectionDetail {
  collection: Collection;
  books: BookCard[];
}

/** A personal, ordered reading list owned by one user. */
export interface ReadingList {
  id: string;
  name: string;
  /** The user's active reading queue (at most one list is active). */
  active: boolean;
  bookCount: number;
  createdAt: number;
  updatedAt: number;
}

/** A reading list plus its books in display order. */
export interface ReadingListDetail {
  readingList: ReadingList;
  books: BookCard[];
}

/** One page in a book manifest. */
export interface ManifestPage {
  idx: number;
  w: number;
  h: number;
  type?: string;
  double?: boolean;
}

/** The reader's source of truth for a book (page list + reading direction). */
export interface BookManifest {
  bookId: string;
  pageCount: number;
  readingDir: 'ltr' | 'rtl';
  pages: ManifestPage[];
}

/** Connection descriptor the client obtains from the embedded sidecar handshake. */
export interface Connection {
  baseUrl: string;
  token: string;
  port?: number;
  pid?: number;
  version?: string;
}

/** A configured metadata provider and whether it has credentials (GET /providers). */
export interface ProviderStatus {
  name: string;
  label: string;
  configured: boolean;
}

/** A ranked provider series (volume) candidate for matching (GET …/match/candidates). */
export interface SeriesMatchCandidate {
  providerId: string;
  name: string;
  year: number;
  publisher: string;
  issueCount: number;
  coverUrl: string;
  /** Matcher confidence, 0..1. */
  score: number;
}
