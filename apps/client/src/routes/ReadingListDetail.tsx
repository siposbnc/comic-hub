import { getRouteApi, useNavigate } from '@tanstack/react-router';
import { useReadingList, useDeleteReadingList, useRemoveFromReadingList } from '../lib/queries.js';
import { ListDetailScreen } from '../components/lists.js';

const route = getRouteApi('/reading-lists/$id');

/** One reading list: its books, with per-issue remove and a delete-list action. */
export function ReadingListDetail() {
  const { id } = route.useParams();
  const navigate = useNavigate();
  const q = useReadingList(id);
  const del = useDeleteReadingList();
  const removeItem = useRemoveFromReadingList();

  const onDelete = async () => {
    if (!window.confirm('Delete this reading list?')) return;
    await del.mutateAsync(id);
    navigate({ to: '/reading-lists' });
  };

  return (
    <ListDetailScreen
      eyebrow="Reading list"
      title={q.data?.readingList.name ?? 'Reading list'}
      bookCount={q.data?.readingList.bookCount ?? 0}
      books={q.data?.books}
      isLoading={q.isLoading}
      isError={q.isError}
      errorMessage={
        q.error instanceof Error ? q.error.message : 'Could not load this reading list.'
      }
      onRetry={() => q.refetch()}
      onBack={() => navigate({ to: '/reading-lists' })}
      onDelete={onDelete}
      onRemoveBook={(bookId) => removeItem.mutate({ id, bookId })}
      emptyText="Add issues from a book's “Add to…” menu."
    />
  );
}
