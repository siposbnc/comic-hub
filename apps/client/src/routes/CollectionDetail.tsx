import { getRouteApi, useNavigate } from '@tanstack/react-router';
import { useCollection, useDeleteCollection, useRemoveFromCollection } from '../lib/queries.js';
import { ListDetailScreen } from '../components/lists.js';

const route = getRouteApi('/collections/$id');

/** One collection: its books, with per-issue remove and a delete-collection action. */
export function CollectionDetail() {
  const { id } = route.useParams();
  const navigate = useNavigate();
  const q = useCollection(id);
  const del = useDeleteCollection();
  const removeItem = useRemoveFromCollection();

  const onDelete = async () => {
    if (!window.confirm('Delete this collection? Your issues are not deleted.')) return;
    await del.mutateAsync(id);
    navigate({ to: '/collections' });
  };

  return (
    <ListDetailScreen
      eyebrow="Collection"
      title={q.data?.collection.name ?? 'Collection'}
      bookCount={q.data?.collection.bookCount ?? 0}
      books={q.data?.books}
      isLoading={q.isLoading}
      isError={q.isError}
      errorMessage={q.error instanceof Error ? q.error.message : 'Could not load this collection.'}
      onRetry={() => q.refetch()}
      onBack={() => navigate({ to: '/collections' })}
      onDelete={onDelete}
      onRemoveBook={(bookId) => removeItem.mutate({ id, bookId })}
      emptyText="Add issues from a book's “Add to…” menu."
    />
  );
}
