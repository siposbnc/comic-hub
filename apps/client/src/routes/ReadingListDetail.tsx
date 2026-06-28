import { useState } from 'react';
import { getRouteApi, useNavigate } from '@tanstack/react-router';
import {
  useReadingList,
  useDeleteReadingList,
  useRemoveFromReadingList,
  useAddToReadingList,
} from '../lib/queries.js';
import { ListDetailScreen } from '../components/lists.js';
import { AddIssuesDialog } from '../components/AddIssuesDialog.js';

const route = getRouteApi('/reading-lists/$id');

/** One reading list: its books, with add/remove issues and a delete-list action. */
export function ReadingListDetail() {
  const { id } = route.useParams();
  const navigate = useNavigate();
  const q = useReadingList(id);
  const del = useDeleteReadingList();
  const removeItem = useRemoveFromReadingList();
  const add = useAddToReadingList();
  const [adding, setAdding] = useState(false);

  const onDelete = async () => {
    if (!window.confirm('Delete this reading list?')) return;
    await del.mutateAsync(id);
    navigate({ to: '/reading-lists' });
  };

  return (
    <>
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
        onAddIssues={() => setAdding(true)}
        emptyText="Add issues with the “Add issues” button, or from a book's “Add to…” menu."
      />
      {adding && (
        <AddIssuesDialog
          title="Add issues to reading list"
          existingIds={new Set((q.data?.books ?? []).map((b) => b.id))}
          onAdd={(bookIds) => add.mutateAsync({ id, bookIds })}
          onClose={() => setAdding(false)}
        />
      )}
    </>
  );
}
