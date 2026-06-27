import {
  QueryClient,
  useQuery,
  useQueries,
  useMutation,
  useQueryClient,
} from '@tanstack/react-query';
import type { ReadStatus } from '@comichub/api-client';
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
