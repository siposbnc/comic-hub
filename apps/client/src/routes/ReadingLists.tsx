import { useRef, useState } from 'react';
import { useNavigate } from '@tanstack/react-router';
import { useQueries } from '@tanstack/react-query';
import { Button, Input, EmptyState } from '@comichub/ui';
import { useClient } from '../lib/client.js';
import { qk, useReadingLists, useCreateReadingList } from '../lib/queries.js';
import { LoadingState, ErrorState } from '../components/Page.js';
import {
  ListIndexHeader,
  ListCardGrid,
  CreateTile,
  CardPill,
  cardCoversAndPct,
  type ListCardModel,
} from '../components/lists.js';

/**
 * Reading lists index (per the design preview): a card grid with a fanned cover collage,
 * "In queue" / "N missing" pills, per-list read progress, and a dashed create tile.
 */
export function ReadingLists() {
  const navigate = useNavigate();
  const client = useClient();
  const q = useReadingLists();
  const create = useCreateReadingList();
  const [name, setName] = useState('');
  // The DS Input doesn't forward refs; focus the real <input> through a wrapper node.
  const nameWrapRef = useRef<HTMLDivElement>(null);
  const focusName = () => nameWrapRef.current?.querySelector('input')?.focus();

  const lists = q.data ?? [];

  // Each card's covers / read % / missing count come from the list's detail (the handoff
  // sanctions a per-list detail fetch; the detail view shares the same cache entry).
  const details = useQueries({
    queries: lists.map((l) => ({
      queryKey: qk.readingList(l.id),
      queryFn: () => client.readingList(l.id),
    })),
  });

  const submit = () => {
    const n = name.trim();
    if (!n || create.isPending) return;
    create.mutate(n);
    setName('');
  };

  const cards: ListCardModel[] = lists.map((l, i) => {
    const d = details[i]?.data;
    const books = d?.books ?? [];
    const { readPct, covers } = cardCoversAndPct(books, (id) => client.coverUrl(id, 200));
    const missing = d ? d.items.length - books.length : 0;
    return {
      id: l.id,
      name: l.name,
      bookCount: l.bookCount,
      readPct,
      covers,
      updatedAt: l.updatedAt,
      topLeft: l.active ? (
        <CardPill tone="accent" icon="book-open">
          In queue
        </CardPill>
      ) : undefined,
      topRight:
        missing > 0 ? (
          <CardPill tone="warning" icon="alert-triangle">
            {missing} missing
          </CardPill>
        ) : undefined,
    };
  });

  if (q.isLoading) {
    return (
      <div style={{ padding: 'var(--pad-screen)' }}>
        <LoadingState />
      </div>
    );
  }
  if (q.isError) {
    return (
      <div style={{ padding: 'var(--pad-screen)' }}>
        <ErrorState
          message={q.error instanceof Error ? q.error.message : 'Could not load reading lists.'}
          onRetry={() => q.refetch()}
        />
      </div>
    );
  }

  return (
    <div style={{ padding: 'var(--pad-screen)', maxWidth: 'var(--content-max)', margin: '0 auto' }}>
      <ListIndexHeader
        eyebrow="Lists"
        title="Reading lists"
        subtitle={`${lists.length} list${lists.length === 1 ? '' : 's'} · curated cross-series reading orders`}
        right={
          <>
            <div ref={nameWrapRef} style={{ width: 220 }}>
              <Input
                icon="plus"
                placeholder="New list name…"
                aria-label="New reading list name"
                value={name}
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => setName(e.target.value)}
                onKeyDown={(e: React.KeyboardEvent) => {
                  if (e.key === 'Enter') submit();
                }}
              />
            </div>
            <Button icon="plus" onClick={submit} disabled={!name.trim() || create.isPending}>
              Create
            </Button>
          </>
        }
      />

      {lists.length === 0 ? (
        <EmptyState title="Nothing here yet">
          Make a reading list to queue up issues to read next.
        </EmptyState>
      ) : (
        <ListCardGrid
          cards={cards}
          onOpen={(id) => navigate({ to: '/reading-lists/$id', params: { id } })}
          createTile={
            <CreateTile
              label="New reading list"
              hint="Name it, add issues later"
              onClick={focusName}
            />
          }
        />
      )}
    </div>
  );
}
