import { useRef, useState } from 'react';
import { useNavigate } from '@tanstack/react-router';
import { useQueries } from '@tanstack/react-query';
import { Button, Input, EmptyState } from '@comichub/ui';
import { useClient } from '../lib/client.js';
import { qk, useCollections, useCreateCollection } from '../lib/queries.js';
import { LoadingState, ErrorState } from '../components/Page.js';
import {
  ListIndexHeader,
  ListCardGrid,
  CreateTile,
  cardCoversAndPct,
  type ListCardModel,
} from '../components/lists.js';

/**
 * Collections index (curated, shared shelves) — the longbox card grid shared with reading
 * lists: fanned covers, per-collection read progress, and a dashed create tile.
 */
export function Collections() {
  const navigate = useNavigate();
  const client = useClient();
  const q = useCollections();
  const create = useCreateCollection();
  const [name, setName] = useState('');
  const nameWrapRef = useRef<HTMLDivElement>(null);
  const focusName = () => nameWrapRef.current?.querySelector('input')?.focus();

  const collections = q.data ?? [];

  const details = useQueries({
    queries: collections.map((c) => ({
      queryKey: qk.collection(c.id),
      queryFn: () => client.collection(c.id),
    })),
  });

  const submit = () => {
    const n = name.trim();
    if (!n || create.isPending) return;
    create.mutate(n);
    setName('');
  };

  const cards: ListCardModel[] = collections.map((c, i) => {
    const books = details[i]?.data?.books ?? [];
    const { readPct, covers } = cardCoversAndPct(books, (id) => client.coverUrl(id, 200));
    return {
      id: c.id,
      name: c.name,
      bookCount: c.bookCount,
      readPct,
      covers,
      updatedAt: c.updatedAt,
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
          message={q.error instanceof Error ? q.error.message : 'Could not load collections.'}
          onRetry={() => q.refetch()}
        />
      </div>
    );
  }

  return (
    <div style={{ padding: 'var(--pad-screen)', maxWidth: 'var(--content-max)', margin: '0 auto' }}>
      <ListIndexHeader
        eyebrow="Lists"
        title="Collections"
        subtitle={`${collections.length} collection${collections.length === 1 ? '' : 's'} · curated shelves you can share`}
        right={
          <>
            <div ref={nameWrapRef} style={{ width: 220 }}>
              <Input
                icon="plus"
                placeholder="New collection name…"
                aria-label="New collection name"
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

      {collections.length === 0 ? (
        <EmptyState title="Nothing here yet">
          Create a collection to group issues into a curated shelf.
        </EmptyState>
      ) : (
        <ListCardGrid
          cards={cards}
          onOpen={(id) => navigate({ to: '/collections/$id', params: { id } })}
          createTile={
            <CreateTile
              label="New collection"
              hint="Name it, add issues later"
              onClick={focusName}
            />
          }
        />
      )}
    </div>
  );
}
