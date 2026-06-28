import { useNavigate } from '@tanstack/react-router';
import { useReadingLists, useCreateReadingList } from '../lib/queries.js';
import { ListIndexScreen } from '../components/lists.js';

/** Index of the user's personal reading lists: create + browse. */
export function ReadingLists() {
  const navigate = useNavigate();
  const q = useReadingLists();
  const create = useCreateReadingList();
  return (
    <ListIndexScreen
      eyebrow="Yours"
      title="Reading Lists"
      items={q.data}
      isLoading={q.isLoading}
      isError={q.isError}
      errorMessage={q.error instanceof Error ? q.error.message : 'Could not load reading lists.'}
      onRetry={() => q.refetch()}
      onOpen={(id) => navigate({ to: '/reading-lists/$id', params: { id } })}
      onCreate={(name) => create.mutate(name)}
      creating={create.isPending}
      createPlaceholder="New reading list name…"
      emptyText="Make a reading list to queue up issues to read next."
    />
  );
}
