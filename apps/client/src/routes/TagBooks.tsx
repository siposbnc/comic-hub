import { getRouteApi, useNavigate } from '@tanstack/react-router';
import { useTags, useTagBooks, useDeleteTag, useUnassignTag } from '../lib/queries.js';
import { ListDetailScreen } from '../components/lists.js';

const route = getRouteApi('/tags/$id');

/** All books carrying a tag: per-issue unassign + a delete-tag action. */
export function TagBooks() {
  const { id } = route.useParams();
  const navigate = useNavigate();
  const tags = useTags();
  const books = useTagBooks(id);
  const del = useDeleteTag();
  const unassign = useUnassignTag();

  const tag = tags.data?.find((t) => t.id === id);

  const onDelete = async () => {
    if (!window.confirm('Delete this tag? It will be removed from all issues.')) return;
    await del.mutateAsync(id);
    navigate({ to: '/tags' });
  };

  return (
    <ListDetailScreen
      eyebrow="Tag"
      title={tag?.name ?? 'Tag'}
      bookCount={books.data?.length ?? 0}
      books={books.data}
      isLoading={books.isLoading}
      isError={books.isError}
      errorMessage={
        books.error instanceof Error ? books.error.message : 'Could not load tagged books.'
      }
      onRetry={() => books.refetch()}
      onBack={() => navigate({ to: '/tags' })}
      onDelete={onDelete}
      onRemoveBook={(bookId) => unassign.mutate({ bookId, tagId: id })}
      emptyText="No issues carry this tag yet."
    />
  );
}
