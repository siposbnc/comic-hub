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

/** A token pair + the signed-in user, returned by login/refresh (auth mode). */
export interface AuthTokens {
  access: string;
  refresh: string;
  /** Unix seconds when the access token expires. */
  accessExpiry: number;
  user: User;
}

/** An account as seen by admin user-management (includes the restriction ceiling). */
export interface UserAccount {
  id: string;
  username: string;
  displayName: string;
  role: UserRole;
  /** Content rating ceiling for restricted users (empty/absent = unrestricted). */
  ageRatingMax?: string;
}

/** Admin create-account input. */
export interface CreateUserInput {
  username: string;
  displayName?: string;
  role: UserRole;
  password: string;
  ageRatingMax?: string;
}

/** Admin update-account input; omitted fields are unchanged. */
export interface UpdateUserInput {
  displayName?: string;
  role?: UserRole;
  ageRatingMax?: string;
  password?: string;
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

/** One item of a bulk progress flush (POST /me/progress/batch). Identify the book by
 *  `bookId`, or by `contentHash` for standalone-mode progress (keyed by the file's
 *  hash) — the server resolves the hash to its catalog book(s). */
export interface ProgressBatchItem {
  bookId?: string;
  contentHash?: string;
  page: number;
  status?: ReadStatus;
  device?: string;
  /** Unix ms of when the reading happened; last-writer-wins by this (ADR-008). */
  updatedAt?: number;
}

/** Per-item result of a bulk progress flush. */
export interface ProgressBatchResult {
  /** The resolved book (for contentHash items: the first match); absent on error. */
  bookId?: string;
  /** False when the write was stale (older than stored) or errored. */
  applied: boolean;
  /** The authoritative row after the flush (absent when the item errored). */
  progress?: Progress;
  error?: string;
}

/**
 * One household member's current reading activity (GET /presence + the WS `presence`
 * topic). Entries above the viewer's content ceiling never arrive.
 */
export interface PresenceEntry {
  userId: string;
  displayName: string;
  bookId: string;
  bookTitle: string;
  seriesId?: string;
  seriesTitle?: string;
  page: number;
  pageCount: number;
  updatedAt: number;
}

/** One bar of the stats dashboard's issues-per-month chart (oldest first). */
export interface StatsMonth {
  /** Short month label, e.g. "Jul". */
  m: string;
  n: number;
}

/** One bar of the stats dashboard's issues-per-day chart (last 30 days, oldest first). */
export interface StatsDay {
  /** Short date label, e.g. "Jul 2". */
  d: string;
  n: number;
}

/** A ranked aggregate row (top genres / publishers). */
export interface StatsNameCount {
  name: string;
  n: number;
}

/** A recently finished issue for the stats dashboard's cover rail. */
export interface StatsFinished {
  bookId: string;
  title?: string;
  number?: string;
  seriesId: string;
  seriesName: string;
  finishedAt: number;
}

/** The per-user reading dashboard (GET /me/stats — design_handoff_stats shape). */
export interface ReadingStats {
  booksRead: number;
  pagesRead: number;
  thisYear: number;
  /** Consecutive reading days ending today/yesterday (approximated from progress). */
  streak: number;
  bestStreak: number;
  months: StatsMonth[];
  days: StatsDay[];
  genres: StatsNameCount[];
  publishers: StatsNameCount[];
  finished: StatsFinished[];
}

/** A user's saved place in a book: a page with an optional short note. */
export interface Bookmark {
  id: string;
  bookId: string;
  page: number;
  note: string;
  createdAt: number;
  updatedAt: number;
}

/** Where a series/book's metadata came from. `incomplete` = auto-match found no 100%
 *  match and the user should match it manually. */
export type MetadataState = 'none' | 'sidecar' | 'matched' | 'locked' | 'incomplete';

/** A series in the library grid. */
export interface SeriesCard {
  id: string;
  name: string;
  year?: number;
  readingDir: 'ltr' | 'rtl';
  bookCount: number;
  readCount: number;
  coverBookId?: string;
  metadataState?: MetadataState;
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
  metadataState?: MetadataState;
  genres?: string[];
  characters?: string[];
  volumes?: GroupingCard[];
  storyArcs?: GroupingCard[];
  books: BookCard[];
}

/** A browsable grouping summary on the series page (a story arc or a volume). */
export interface GroupingCard {
  id: string;
  name: string;
  year?: number;
  issueCount: number;
  description?: string;
}

/** A story-arc/volume detail: header + its issues (same BookCard shape as series issues). */
export interface GroupingDetail {
  id: string;
  kind: 'arc' | 'volume';
  name: string;
  seriesId: string;
  seriesName: string;
  year?: number;
  description?: string;
  readingDir: 'ltr' | 'rtl';
  issueCount: number;
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
  /** Disambiguates same-named series that exist in more than one library. */
  libraryName?: string;
}

/** A book matched by full-text search (carries its series name for display). */
export interface BookHit {
  id: string;
  seriesId: string;
  seriesName?: string;
  number?: string;
  title?: string;
  format: string;
  libraryName?: string;
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

/** One ordered entry of a reading list. Stale entries (their book was deleted, or they
 * were added manually for an issue not in the library) have no `book`: they render from
 * the snapshot fields and hold their slot in the order, but can't be opened. */
export interface ReadingListEntry {
  id: string;
  stale: boolean;
  seriesName?: string;
  number?: string;
  title?: string;
  addedAt: number;
  book?: BookCard;
}

/** A placeholder entry for an issue that isn't in the library (yet). */
export interface ManualListEntry {
  seriesName?: string;
  number?: string;
  title?: string;
}

/** A reading list plus its entries in display order. `books` repeats the linked entries'
 * cards for older callers; `items` is the full order including stale placeholders. */
export interface ReadingListDetail {
  readingList: ReadingList;
  items: ReadingListEntry[];
  books: BookCard[];
}

/** A standalone (manual) track on the Tracker — a series the user follows that is in no
 * library. Library series aren't tracks; they're projected live into the tracker view. */
export interface Track {
  id: string;
  name: string;
  createdAt: number;
  updatedAt: number;
}

/** One cell of the Tracker matrix. A library issue carries `bookId` (toggle read via
 * `markBook`); an overlay issue carries `id` (toggle via `markTrackIssue`) and no book —
 * its read state is independent of owning a file. */
export interface TrackerIssue {
  id?: string;
  number: string;
  sort: number;
  bookId?: string;
  state: 'read' | 'reading' | 'unread';
  page?: number;
  pages?: number;
  source: 'library' | 'manual';
}

/** One row of the Tracker: a library series (`link: 'library'`, `seriesId` set) or a
 * standalone track (`link: 'manual'`). */
export interface TrackerTrack {
  id: string;
  seriesId?: string;
  libraryId?: string;
  name: string;
  link: 'library' | 'manual';
  issues: TrackerIssue[];
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
/** Provider credential status for the settings screen (secrets are never returned). */
export interface ProviderSettings {
  comicvine: { configured: boolean };
  metron: { configured: boolean; username: string };
  /** Write matched metadata back into each book's .cbz as a ComicInfo.xml. */
  writeSidecar: boolean;
}

/** Provider credential update. Omitted fields are left unchanged; "" clears a field. */
export interface ProviderSettingsUpdate {
  comicVineApiKey?: string;
  metronUsername?: string;
  metronPassword?: string;
  writeSidecar?: boolean;
}

export interface SeriesMatchCandidate {
  providerId: string;
  /** Source provider name (e.g. "comicvine", "metron"). */
  provider: string;
  name: string;
  year: number;
  publisher: string;
  issueCount: number;
  coverUrl: string;
  /** Matcher confidence, 0..1. */
  score: number;
}
