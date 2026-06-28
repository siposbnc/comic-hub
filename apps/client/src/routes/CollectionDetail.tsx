import { useState } from 'react';
import { getRouteApi, useNavigate } from '@tanstack/react-router';
import {
  useCollection,
  useDeleteCollection,
  useRemoveFromCollection,
  useAddToCollection,
} from '../lib/queries.js';
import { ListDetailScreen } from '../components/lists.js';
import { AddIssuesDialog } from '../components/AddIssuesDialog.js';

const route = getRouteApi('/collections/$id');

/** One collection: its books, with add/remove issues and a delete-collection action. */
export function CollectionDetail() {
  const { id } = route.useParams();
  const navigate = useNavigate();
  const q = useCollection(id);
  const del = useDeleteCollection();
  const removeItem = useRemoveFromCollection();
  const add = useAddToCollection();
  const [adding, setAdding] = useState(false);

  const onDelete = async () => {
    if (!window.confirm('Delete this collection? Your issues are not deleted.')) return;
    await del.mutateAsync(id);
    navigate({ to: '/collections' });
  };

  return (
    <>
      <ListDetailScreen
        eyebrow="Collection"
        title={q.data?.collection.name ?? 'Collection'}
        bookCount={q.data?.collection.bookCount ?? 0}
        books={q.data?.books}
        isLoading={q.isLoading}
        isError={q.isError}
        errorMessage={
          q.error instanceof Error ? q.error.message : 'Could not load this collection.'
        }
        onRetry={() => q.refetch()}
        onBack={() => navigate({ to: '/collections' })}
        onDelete={onDelete}
        onRemoveBook={(bookId) => removeItem.mutate({ id, bookId })}
        onAddIssues={() => setAdding(true)}
        emptyText="Add issues with the “Add issues” button, or from a book's “Add to…” menu."
      />
      {adding && (
        <AddIssuesDialog
          title="Add issues to collection"
          existingIds={new Set((q.data?.books ?? []).map((b) => b.id))}
          onAdd={(bookIds) => add.mutateAsync({ id, bookIds })}
          onClose={() => setAdding(false)}
        />
      )}
    </>
  );
}
