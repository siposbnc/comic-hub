import { useEffect, useState } from 'react';
import { Button, Input, Icon } from '@comichub/ui';
import type { Tag } from '@comichub/api-client';
import { useTags, useCreateTag, useAssignTags, useUnassignTag } from '../lib/queries.js';

/** Read-only tag pills (used on the book detail). */
export function TagChips({ tags }: { tags: Tag[] }) {
  if (tags.length === 0) return null;
  return (
    <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
      {tags.map((t) => (
        <TagPill key={t.id} tag={t} />
      ))}
    </div>
  );
}

function TagPill({ tag, active = true }: { tag: Tag; active?: boolean }) {
  return (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 6,
        padding: '3px 10px',
        borderRadius: 999,
        fontSize: 'var(--text-label)',
        background: active ? 'var(--accent-soft)' : 'var(--surface-card)',
        color: active ? 'var(--accent)' : 'var(--text-secondary)',
        border: `1px solid ${active ? 'var(--accent)' : 'var(--border-hairline)'}`,
      }}
    >
      <span
        aria-hidden
        style={{
          width: 7,
          height: 7,
          borderRadius: 999,
          background: tag.color || 'var(--text-tertiary)',
        }}
      />
      {tag.name}
    </span>
  );
}

/**
 * Modal to manage a book's tags: toggle any tag on/off and create new ones inline.
 * Mirrors the AddLibraryDialog overlay pattern.
 */
export function TagEditor({
  bookId,
  assignedIds,
  onClose,
}: {
  bookId: string;
  assignedIds: Set<string>;
  onClose: () => void;
}) {
  const tags = useTags();
  const create = useCreateTag();
  const assign = useAssignTags();
  const unassign = useUnassignTag();
  const [name, setName] = useState('');

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [onClose]);

  const toggle = (tag: Tag) => {
    if (assignedIds.has(tag.id)) unassign.mutate({ bookId, tagId: tag.id });
    else assign.mutate({ bookId, tagIds: [tag.id] });
  };

  const submitNew = async (e: React.FormEvent) => {
    e.preventDefault();
    const n = name.trim();
    if (!n || create.isPending) return;
    try {
      const tag = await create.mutateAsync({ name: n });
      await assign.mutateAsync({ bookId, tagIds: [tag.id] });
      setName('');
    } catch {
      // surfaced by the create/assign error states; keep the dialog open
    }
  };

  return (
    <div
      role="presentation"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 1100,
        display: 'grid',
        placeItems: 'center',
        background: 'color-mix(in oklab, var(--ink-900) 70%, transparent)',
        padding: 24,
      }}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-label="Edit tags"
        style={{
          width: 'min(440px, 100%)',
          maxHeight: '72vh',
          overflowY: 'auto',
          background: 'var(--surface-raised)',
          border: '1px solid var(--border-strong)',
          borderRadius: 'var(--radius-lg)',
          boxShadow: 'var(--shadow-popover)',
          padding: 20,
          display: 'flex',
          flexDirection: 'column',
          gap: 14,
        }}
      >
        <h2
          style={{
            margin: 0,
            fontFamily: 'var(--font-display)',
            fontSize: 'var(--text-heading)',
            fontWeight: 700,
            color: 'var(--text-primary)',
          }}
        >
          Edit tags
        </h2>

        <form onSubmit={submitNew} style={{ display: 'flex', gap: 8 }}>
          <div style={{ flex: 1, minWidth: 0 }}>
            <Input
              icon="plus"
              placeholder="New tag…"
              value={name}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setName(e.target.value)}
            />
          </div>
          <Button type="submit" variant="secondary" disabled={!name.trim() || create.isPending}>
            Add
          </Button>
        </form>

        {tags.data && tags.data.length > 0 ? (
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
            {tags.data.map((t) => {
              const active = assignedIds.has(t.id);
              return (
                <button
                  key={t.id}
                  type="button"
                  onClick={() => toggle(t)}
                  style={{
                    display: 'inline-flex',
                    alignItems: 'center',
                    gap: 6,
                    padding: '5px 10px',
                    borderRadius: 999,
                    cursor: 'pointer',
                    fontSize: 'var(--text-small)',
                    background: active ? 'var(--accent-soft)' : 'var(--surface-card)',
                    color: active ? 'var(--accent)' : 'var(--text-secondary)',
                    border: `1px solid ${active ? 'var(--accent)' : 'var(--border-hairline)'}`,
                  }}
                >
                  <Icon name={active ? 'check' : 'plus'} size={13} />
                  {t.name}
                </button>
              );
            })}
          </div>
        ) : (
          <p style={{ margin: 0, color: 'var(--text-tertiary)', fontSize: 'var(--text-small)' }}>
            No tags yet — create one above.
          </p>
        )}

        <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
          <Button type="button" variant="ghost" onClick={onClose}>
            Done
          </Button>
        </div>
      </div>
    </div>
  );
}
