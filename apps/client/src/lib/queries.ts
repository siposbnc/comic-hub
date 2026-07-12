import {
  QueryClient,
  useQuery,
  useQueries,
  useMutation,
  useQueryClient,
} from '@tanstack/react-query';
import type { ReadStatus, SmartRules } from '@comichub/api-client';
import { useClient } from './client.js';

/**
 * Stable query-key factory. WS event handlers invalidate by these prefixes, so keeping
 * them in one place keeps push-invalidation and reads in lockstep.
 */
export const qk = {
  libraries: ['libraries'] as const,
  library: (id: string) => ['library', id] as const,
  series: (libraryId: string) => ['series', libraryId] as const,
  seriesDetail: (id: string) => ['seriesDetail', id] as const,
  book: (id: string) => ['book', id] as const,
  discover: (libraryId?: string) => ['discover', libraryId ?? 'all'] as const,
  continueReading: ['continue'] as const,
  job: (id: string) => ['job', id] as const,
  collections: ['collections'] as const,
  collection: (id: string) => ['collection', id] as const,
  readingLists: ['readingLists'] as const,
  readingList: (id: string) => ['readingList', id] as const,
  tags: ['tags'] as const,
  smartLists: ['smartLists'] as const,
  smartList: (id: string) => ['smartList', id] as const,
  tracker: ['tracker'] as const,
  presence: ['presence'] as const,
};

/** App-wide QueryClient. Covers are immutable + content-addressed, so list data can be
 *  cached generously; WS push handles freshness for the parts that change. */
export function createQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 30_000,
        gcTime: 5 * 60_000,
        retry: 1,
        refetchOnWindowFocus: false,
      },
    },
  });
}

export function useLibraries() {
  const client = useClient();
  return useQuery({ queryKey: qk.libraries, queryFn: () => client.listLibraries() });
}

export function useLibrary(id: string) {
  const client = useClient();
  return useQuery({ queryKey: qk.library(id), queryFn: () => client.getLibrary(id) });
}

export function useLibraryHealth(id: string, enabled = true) {
  const client = useClient();
  return useQuery({
    queryKey: ['libraryHealth', id],
    queryFn: () => client.libraryHealth(id),
    enabled,
  });
}

export function useSeriesList(libraryId: string) {
  const client = useClient();
  return useQuery({
    queryKey: qk.series(libraryId),
    queryFn: () => client.listSeries(libraryId),
    enabled: Boolean(libraryId),
  });
}

export function useSeriesDetail(id: string) {
  const client = useClient();
  return useQuery({ queryKey: qk.seriesDetail(id), queryFn: () => client.seriesDetail(id) });
}

export function useStoryArc(seriesId: string, arcId: string) {
  const client = useClient();
  return useQuery({
    queryKey: ['storyArc', seriesId, arcId] as const,
    queryFn: () => client.storyArc(seriesId, arcId),
  });
}

export function useVolume(seriesId: string, volume: string) {
  const client = useClient();
  return useQuery({
    queryKey: ['volume', seriesId, volume] as const,
    queryFn: () => client.volume(seriesId, volume),
  });
}

export function useBookDetail(id: string) {
  const client = useClient();
  return useQuery({ queryKey: qk.book(id), queryFn: () => client.bookDetail(id) });
}

export function useDiscover(libraryId?: string) {
  const client = useClient();
  return useQuery({
    queryKey: qk.discover(libraryId),
    queryFn: () => client.discover(libraryId),
  });
}

export function useContinueReading() {
  const client = useClient();
  return useQuery({ queryKey: qk.continueReading, queryFn: () => client.continueReading() });
}

/**
 * Household "now reading" presence (auth mode only — pass enabled=false elsewhere).
 * The snapshot seeds the cache; WS presence events keep it live (see events.ts), so
 * the query itself never goes stale.
 */
export function usePresence(enabled: boolean) {
  const client = useClient();
  return useQuery({
    queryKey: qk.presence,
    queryFn: () => client.presence(),
    enabled,
    staleTime: Infinity,
  });
}

/**
 * A seriesId → name lookup spanning every library, used to label cross-library cards
 * (Home rails) where the wire BookCard carries only `seriesId`.
 */
export function useSeriesNames(): Map<string, string> {
  const client = useClient();
  const libraries = useLibraries();
  const ids = libraries.data?.map((l) => l.id) ?? [];

  const results = useQueries({
    queries: ids.map((libraryId) => ({
      queryKey: qk.series(libraryId),
      queryFn: () => client.listSeries(libraryId),
      staleTime: 60_000,
    })),
  });

  const map = new Map<string, string>();
  for (const r of results) {
    for (const s of r.data ?? []) map.set(s.id, s.name);
  }
  return map;
}

/**
 * A libraryId → series-count map across every library, for the sidebar nav counts.
 * Reuses the `qk.series` query keys, so it shares cache with the library screens
 * rather than issuing extra requests.
 */
export function useLibrarySeriesCounts(): Map<string, number> {
  const client = useClient();
  const libraries = useLibraries();
  const ids = libraries.data?.map((l) => l.id) ?? [];

  const results = useQueries({
    queries: ids.map((libraryId) => ({
      queryKey: qk.series(libraryId),
      queryFn: () => client.listSeries(libraryId),
      staleTime: 60_000,
    })),
  });

  const map = new Map<string, number>();
  ids.forEach((id, i) => {
    const data = results[i]?.data;
    if (data) map.set(id, data.length);
  });
  return map;
}

// ── Collections (shared, ordered shelves) ───────────────────────────────────────────

export function useCollections() {
  const client = useClient();
  return useQuery({ queryKey: qk.collections, queryFn: () => client.listCollections() });
}

export function useCollection(id: string) {
  const client = useClient();
  return useQuery({ queryKey: qk.collection(id), queryFn: () => client.collection(id) });
}

export function useCreateCollection() {
  const client = useClient();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (name: string) => client.createCollection({ name }),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.collections }),
  });
}

export function useDeleteCollection() {
  const client = useClient();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => client.deleteCollection(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.collections }),
  });
}

export function useAddToCollection() {
  const client = useClient();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, bookIds }: { id: string; bookIds: string[] }) =>
      client.addToCollection(id, bookIds),
    onSuccess: (_data, { id }) => {
      qc.invalidateQueries({ queryKey: qk.collection(id) });
      qc.invalidateQueries({ queryKey: qk.collections });
    },
  });
}

export function useRemoveFromCollection() {
  const client = useClient();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, bookId }: { id: string; bookId: string }) =>
      client.removeFromCollection(id, bookId),
    onSuccess: (_data, { id }) => {
      qc.invalidateQueries({ queryKey: qk.collection(id) });
      qc.invalidateQueries({ queryKey: qk.collections });
    },
  });
}

// ── Reading lists (per-user, ordered) ───────────────────────────────────────────────

export function useReadingLists() {
  const client = useClient();
  return useQuery({ queryKey: qk.readingLists, queryFn: () => client.listReadingLists() });
}

export function useReadingList(id: string) {
  const client = useClient();
  return useQuery({ queryKey: qk.readingList(id), queryFn: () => client.readingList(id) });
}

export function useCreateReadingList() {
  const client = useClient();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (name: string) => client.createReadingList(name),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.readingLists }),
  });
}

export function useDeleteReadingList() {
  const client = useClient();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => client.deleteReadingList(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.readingLists }),
  });
}

export function useAddToReadingList() {
  const client = useClient();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, bookIds }: { id: string; bookIds: string[] }) =>
      client.addToReadingList(id, bookIds),
    onSuccess: (_data, { id }) => {
      qc.invalidateQueries({ queryKey: qk.readingList(id) });
      qc.invalidateQueries({ queryKey: qk.readingLists });
      qc.invalidateQueries({ queryKey: ['discover'] });
    },
  });
}

export function useRemoveFromReadingList() {
  const client = useClient();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, bookId }: { id: string; bookId: string }) =>
      client.removeFromReadingList(id, bookId),
    onSuccess: (_data, { id }) => {
      qc.invalidateQueries({ queryKey: qk.readingList(id) });
      qc.invalidateQueries({ queryKey: qk.readingLists });
      qc.invalidateQueries({ queryKey: ['discover'] });
    },
  });
}

export function useReorderReadingList() {
  const client = useClient();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, bookId, beforeId }: { id: string; bookId: string; beforeId?: string }) =>
      client.reorderReadingList(id, bookId, beforeId),
    onSuccess: (_data, { id }) => {
      qc.invalidateQueries({ queryKey: qk.readingList(id) });
      qc.invalidateQueries({ queryKey: ['discover'] });
    },
  });
}

export function useAddManualToReadingList() {
  const client = useClient();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      manual,
    }: {
      id: string;
      manual: { seriesName?: string; number?: string; title?: string }[];
    }) => client.addManualToReadingList(id, manual),
    onSuccess: (_data, { id }) => {
      qc.invalidateQueries({ queryKey: qk.readingList(id) });
      qc.invalidateQueries({ queryKey: qk.readingLists });
    },
  });
}

export function useRelinkReadingListItem() {
  const client = useClient();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, itemId, bookId }: { id: string; itemId: string; bookId: string }) =>
      client.relinkReadingListItem(id, itemId, bookId),
    onSuccess: (_data, { id }) => {
      qc.invalidateQueries({ queryKey: qk.readingList(id) });
      qc.invalidateQueries({ queryKey: qk.readingLists });
      qc.invalidateQueries({ queryKey: ['discover'] });
    },
  });
}

export function useRescanSeries() {
  const client = useClient();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => client.rescanSeries(id),
    onSuccess: () => {
      // The series is gone until the scan re-creates it; drop everything derived.
      qc.invalidateQueries();
    },
  });
}

export function useSetActiveReadingList() {
  const client = useClient();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => client.setActiveReadingList(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: qk.readingLists });
      qc.invalidateQueries({ queryKey: ['readingList'] });
      qc.invalidateQueries({ queryKey: ['discover'] });
    },
  });
}

// ── Tags ─────────────────────────────────────────────────────────────────────────────

export function useTags() {
  const client = useClient();
  return useQuery({ queryKey: qk.tags, queryFn: () => client.listTags() });
}

export function useTagBooks(id: string) {
  const client = useClient();
  return useQuery({ queryKey: ['tagBooks', id], queryFn: () => client.tagBooks(id) });
}

export function useCreateTag() {
  const client = useClient();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { name: string; color?: string }) => client.createTag(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.tags }),
  });
}

export function useDeleteTag() {
  const client = useClient();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => client.deleteTag(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: qk.tags });
      qc.invalidateQueries({ queryKey: ['book'] });
    },
  });
}

/** Assign/unassign a tag, then refresh the tag list and the affected book. */
export function useAssignTags() {
  const client = useClient();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ bookId, tagIds }: { bookId: string; tagIds: string[] }) =>
      client.assignTags(bookId, tagIds),
    onSuccess: (_data, { bookId }) => {
      qc.invalidateQueries({ queryKey: qk.tags });
      qc.invalidateQueries({ queryKey: qk.book(bookId) });
    },
  });
}

export function useUnassignTag() {
  const client = useClient();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ bookId, tagId }: { bookId: string; tagId: string }) =>
      client.unassignTag(bookId, tagId),
    onSuccess: (_data, { bookId, tagId }) => {
      qc.invalidateQueries({ queryKey: qk.tags });
      qc.invalidateQueries({ queryKey: qk.book(bookId) });
      qc.invalidateQueries({ queryKey: ['tagBooks', tagId] });
    },
  });
}

// ── Smart lists (rule-based) ─────────────────────────────────────────────────────────

export function useSmartLists() {
  const client = useClient();
  return useQuery({ queryKey: qk.smartLists, queryFn: () => client.listSmartLists() });
}

export function useSmartList(id: string) {
  const client = useClient();
  return useQuery({ queryKey: qk.smartList(id), queryFn: () => client.smartListResults(id) });
}

export function useCreateSmartList() {
  const client = useClient();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { name: string; rules: SmartRules }) => client.createSmartList(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.smartLists }),
  });
}

export function useDeleteSmartList() {
  const client = useClient();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => client.deleteSmartList(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.smartLists }),
  });
}

// ── Tracker (per-user reading matrix) ────────────────────────────────────────────────

/** Surfaces a tracker mutation touches: the matrix itself plus every read-state view. */
function invalidateTracker(qc: ReturnType<typeof useQueryClient>) {
  qc.invalidateQueries({ queryKey: qk.tracker });
  qc.invalidateQueries({ queryKey: ['seriesDetail'] });
  qc.invalidateQueries({ queryKey: ['series'] });
  qc.invalidateQueries({ queryKey: ['discover'] });
  qc.invalidateQueries({ queryKey: qk.continueReading });
}

export function useTracker() {
  const client = useClient();
  return useQuery({ queryKey: qk.tracker, queryFn: () => client.tracker() });
}

export function useCreateTrack() {
  const client = useClient();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (name: string) => client.createTrack(name),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.tracker }),
  });
}

export function useRenameTrack() {
  const client = useClient();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) => client.renameTrack(id, name),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.tracker }),
  });
}

export function useDeleteTrack() {
  const client = useClient();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => client.deleteTrack(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.tracker }),
  });
}

export function useAddTrackIssues() {
  const client = useClient();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { trackId?: string; seriesId?: string; numbers: string[] }) =>
      client.addTrackIssues(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.tracker }),
  });
}

export function useRemoveTrackIssue() {
  const client = useClient();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => client.removeTrackIssue(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.tracker }),
  });
}

/**
 * Toggle one tracker cell. A library issue (bookId) routes through the progress API; an
 * overlay issue (issueId) flips its own read flag. Either way every read-state view refreshes.
 */
export function useToggleTrackerIssue() {
  const client = useClient();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (v: { bookId?: string; issueId?: string; read: boolean }) => {
      if (v.bookId) await client.markBook(v.bookId, v.read ? 'read' : 'unread');
      else if (v.issueId) await client.markTrackIssue(v.issueId, v.read);
    },
    onSuccess: () => invalidateTracker(qc),
  });
}

/** Mark a run of cells at once (shift-click): library issues via markBook, overlay via mark. */
export function useRangeMarkTracker() {
  const client = useClient();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (v: { bookIds: string[]; issueIds: string[]; read: boolean }) => {
      const status = v.read ? 'read' : 'unread';
      await Promise.all([
        ...v.bookIds.map((id) => client.markBook(id, status)),
        ...v.issueIds.map((id) => client.markTrackIssue(id, v.read)),
      ]);
    },
    onSuccess: () => invalidateTracker(qc),
  });
}

/** Toggle a book's read state, then refresh the surfaces that show it. */
export function useMarkBook() {
  const client = useClient();
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ bookId, status }: { bookId: string; status: 'read' | 'unread' }) =>
      client.markBook(bookId, status),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['seriesDetail'] });
      qc.invalidateQueries({ queryKey: ['book'] });
      qc.invalidateQueries({ queryKey: ['series'] });
      qc.invalidateQueries({ queryKey: ['discover'] });
      qc.invalidateQueries({ queryKey: qk.continueReading });
    },
  });
}

export type { ReadStatus };
