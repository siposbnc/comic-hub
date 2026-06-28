import { useNavigate } from '@tanstack/react-router';
import { useCollections, useCreateCollection } from '../lib/queries.js';
import { ListIndexScreen } from '../components/lists.js';

/** Index of collections (curated, shared shelves): create + browse. */
export function Collections() {
  const navigate = useNavigate();
  const q = useCollections();
  const create = useCreateCollection();
  return (
    <ListIndexScreen
      eyebrow="Library"
      title="Collections"
      items={q.data}
      isLoading={q.isLoading}
      isError={q.isError}
      errorMessage={q.error instanceof Error ? q.error.message : 'Could not load collections.'}
      onRetry={() => q.refetch()}
      onOpen={(id) => navigate({ to: '/collections/$id', params: { id } })}
      onCreate={(name) => create.mutate(name)}
      creating={create.isPending}
      createPlaceholder="New collection name…"
      emptyText="Create a collection to group issues into a curated shelf."
    />
  );
}
