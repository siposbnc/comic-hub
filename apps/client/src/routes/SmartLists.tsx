import { useNavigate } from '@tanstack/react-router';
import { useSmartLists, useCreateSmartList, useTags } from '../lib/queries.js';
import { Page, LoadingState, ErrorState } from '../components/Page.js';
import { ListRow } from '../components/lists.js';
import { SmartListBuilder } from '../components/smartlist.js';
import { EmptyState } from '@comichub/ui';

/** Index of smart lists: a rule builder + the existing rule-based lists. */
export function SmartLists() {
  const navigate = useNavigate();
  const lists = useSmartLists();
  const tags = useTags();
  const create = useCreateSmartList();

  return (
    <Page eyebrow="Library" title="Smart Lists">
      <SmartListBuilder
        tags={tags.data ?? []}
        creating={create.isPending}
        onCreate={(name, rules) => create.mutate({ name, rules })}
      />

      {lists.isLoading ? (
        <LoadingState />
      ) : lists.isError ? (
        <ErrorState
          message={
            lists.error instanceof Error ? lists.error.message : 'Could not load smart lists.'
          }
          onRetry={() => lists.refetch()}
        />
      ) : !lists.data || lists.data.length === 0 ? (
        <EmptyState title="No smart lists yet">
          Build one above — e.g. “Read status is unread” for everything left to read.
        </EmptyState>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8, maxWidth: 720 }}>
          {lists.data.map((l) => (
            <ListRow
              key={l.id}
              item={l}
              onOpen={() => navigate({ to: '/smart-lists/$id', params: { id: l.id } })}
            />
          ))}
        </div>
      )}
    </Page>
  );
}
