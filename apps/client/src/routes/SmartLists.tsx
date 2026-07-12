import { useNavigate } from '@tanstack/react-router';
import { useQueries } from '@tanstack/react-query';
import { EmptyState } from '@comichub/ui';
import { useClient } from '../lib/client.js';
import { qk, useSmartLists, useCreateSmartList, useTags } from '../lib/queries.js';
import { LoadingState, ErrorState } from '../components/Page.js';
import {
  ListIndexHeader,
  ListCardGrid,
  cardCoversAndPct,
  type ListCardModel,
} from '../components/lists.js';
import { SmartListBuilder } from '../components/smartlist.js';

/**
 * Smart lists index — the same longbox card grid as reading lists (fanned covers + read
 * progress), with the rule builder kept above it since a smart list is defined by its rules.
 */
export function SmartLists() {
  const navigate = useNavigate();
  const client = useClient();
  const lists = useSmartLists();
  const tags = useTags();
  const create = useCreateSmartList();

  const items = lists.data ?? [];

  // Covers + read % per list come from evaluating its rules (shared cache with the detail view).
  const details = useQueries({
    queries: items.map((l) => ({
      queryKey: qk.smartList(l.id),
      queryFn: () => client.smartListResults(l.id),
    })),
  });

  const cards: ListCardModel[] = items.map((l, i) => {
    const books = details[i]?.data?.books ?? [];
    const { readPct, covers } = cardCoversAndPct(books, (id) => client.coverUrl(id, 200));
    return {
      id: l.id,
      name: l.name,
      bookCount: l.bookCount,
      readPct,
      covers,
      updatedAt: l.updatedAt,
    };
  });

  return (
    <div style={{ padding: 'var(--pad-screen)', maxWidth: 'var(--content-max)', margin: '0 auto' }}>
      <ListIndexHeader
        eyebrow="Lists"
        title="Smart lists"
        subtitle={`${items.length} list${items.length === 1 ? '' : 's'} · rule-based, always up to date`}
      />

      <div style={{ marginBottom: 24 }}>
        <SmartListBuilder
          tags={tags.data ?? []}
          creating={create.isPending}
          onCreate={(name, rules) => create.mutate({ name, rules })}
        />
      </div>

      {lists.isLoading ? (
        <LoadingState />
      ) : lists.isError ? (
        <ErrorState
          message={
            lists.error instanceof Error ? lists.error.message : 'Could not load smart lists.'
          }
          onRetry={() => lists.refetch()}
        />
      ) : items.length === 0 ? (
        <EmptyState title="No smart lists yet">
          Build one above — e.g. “Read status is unread” for everything left to read.
        </EmptyState>
      ) : (
        <ListCardGrid
          cards={cards}
          onOpen={(id) => navigate({ to: '/smart-lists/$id', params: { id } })}
        />
      )}
    </div>
  );
}
