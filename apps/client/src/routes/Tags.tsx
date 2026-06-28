import { useState } from 'react';
import { useNavigate } from '@tanstack/react-router';
import { Button, Input, Icon, EmptyState } from '@comichub/ui';
import { useTags, useCreateTag } from '../lib/queries.js';
import { Page, LoadingState, ErrorState } from '../components/Page.js';

/** Index of tags: create + browse; clicking a tag opens its books. */
export function Tags() {
  const navigate = useNavigate();
  const tags = useTags();
  const create = useCreateTag();
  const [name, setName] = useState('');

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    const n = name.trim();
    if (!n || create.isPending) return;
    create.mutate({ name: n });
    setName('');
  };

  return (
    <Page eyebrow="Library" title="Tags">
      <form onSubmit={submit} style={{ display: 'flex', gap: 10, maxWidth: 520, marginBottom: 24 }}>
        <div style={{ flex: 1, minWidth: 0 }}>
          <Input
            icon="plus"
            placeholder="New tag name…"
            value={name}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) => setName(e.target.value)}
          />
        </div>
        <Button type="submit" variant="secondary" disabled={!name.trim() || create.isPending}>
          {create.isPending ? 'Creating…' : 'Create'}
        </Button>
      </form>

      {tags.isLoading ? (
        <LoadingState />
      ) : tags.isError ? (
        <ErrorState
          message={tags.error instanceof Error ? tags.error.message : 'Could not load tags.'}
          onRetry={() => tags.refetch()}
        />
      ) : !tags.data || tags.data.length === 0 ? (
        <EmptyState title="No tags yet">
          Create tags here, then apply them to issues from a book’s “Edit tags”.
        </EmptyState>
      ) : (
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 10 }}>
          {tags.data.map((t) => (
            <button
              key={t.id}
              type="button"
              onClick={() => navigate({ to: '/tags/$id', params: { id: t.id } })}
              style={{
                display: 'inline-flex',
                alignItems: 'center',
                gap: 8,
                padding: '8px 14px',
                borderRadius: 999,
                cursor: 'pointer',
                background: 'var(--surface-raised)',
                border: '1px solid var(--border-hairline)',
                color: 'var(--text-primary)',
                fontSize: 'var(--text-small)',
              }}
            >
              <span
                aria-hidden
                style={{
                  width: 8,
                  height: 8,
                  borderRadius: 999,
                  background: t.color || 'var(--text-tertiary)',
                }}
              />
              {t.name}
              <span className="ch-mono" style={{ color: 'var(--text-tertiary)' }}>
                {t.bookCount}
              </span>
              <Icon name="chevron-right" size={14} color="var(--text-tertiary)" />
            </button>
          ))}
        </div>
      )}
    </Page>
  );
}
