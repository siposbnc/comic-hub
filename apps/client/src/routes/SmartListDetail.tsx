import { getRouteApi, useNavigate } from '@tanstack/react-router';
import { useSmartList, useDeleteSmartList } from '../lib/queries.js';
import { ListDetailScreen } from '../components/lists.js';

const route = getRouteApi('/smart-lists/$id');

/** One smart list: its evaluated results (read-only) and a delete action. */
export function SmartListDetail() {
  const { id } = route.useParams();
  const navigate = useNavigate();
  const q = useSmartList(id);
  const del = useDeleteSmartList();

  const onDelete = async () => {
    if (!window.confirm('Delete this smart list? Your issues are not affected.')) return;
    await del.mutateAsync(id);
    navigate({ to: '/smart-lists' });
  };

  return (
    <ListDetailScreen
      eyebrow="Smart list"
      title={q.data?.smartList.name ?? 'Smart list'}
      bookCount={q.data?.smartList.bookCount ?? 0}
      books={q.data?.books}
      isLoading={q.isLoading}
      isError={q.isError}
      errorMessage={q.error instanceof Error ? q.error.message : 'Could not load this smart list.'}
      onRetry={() => q.refetch()}
      onBack={() => navigate({ to: '/smart-lists' })}
      onDelete={onDelete}
      emptyText="No issues match these rules yet."
    />
  );
}
