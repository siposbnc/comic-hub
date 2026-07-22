import { useEffect, useMemo, useState } from 'react';
import { Button, Badge, Input, Icon, IconButton } from '@comichub/ui';
import type { Collection } from '@comichub/api-client';
import { useClient } from '../lib/client.js';
import { useCollections } from '../lib/queries.js';
import { Cover, Hint } from './AddIssuesDialog.js';

/**
 * Pick one or more collections to reference into a reading list. Each becomes a single
 * ordered group that expands, live, into the collection's books. Collections already in the
 * list show as added and aren't selectable. Matches the AddIssuesDialog shell.
 */
export function AddCollectionDialog({
  subtitle,
  existingIds,
  onAdd,
  onClose,
}: {
  subtitle?: string;
  /** Collection ids already referenced by this list. */
  existingIds: Set<string>;
  onAdd: (collectionIds: string[]) => Promise<void>;
  onClose: () => void;
}) {
  const client = useClient();
  const collections = useCollections();
  const [query, setQuery] = useState('');
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !busy) onClose();
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [busy, onClose]);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    const all = collections.data ?? [];
    return q ? all.filter((c) => c.name.toLowerCase().includes(q)) : all;
  }, [collections.data, query]);

  const toggle = (c: Collection) => {
    if (existingIds.has(c.id)) return;
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(c.id)) next.delete(c.id);
      else next.add(c.id);
      return next;
    });
  };

  const submit = async () => {
    if (selected.size === 0 || busy) return;
    setBusy(true);
    try {
      await onAdd([...selected]);
      onClose();
    } catch {
      setBusy(false); // surfaced by the caller's toast; keep the dialog open
    }
  };

  return (
    <div
      role="presentation"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget && !busy) onClose();
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
        aria-label="Add collection"
        style={{
          width: 'min(560px, 100%)',
          height: 'min(680px, 90vh)',
          display: 'flex',
          flexDirection: 'column',
          background: 'var(--surface-raised)',
          border: '1px solid var(--border-strong)',
          borderRadius: 'var(--radius-lg)',
          boxShadow: 'var(--shadow-popover)',
          padding: 20,
          gap: 12,
        }}
      >
        <div style={{ display: 'flex', alignItems: 'flex-start', gap: 10 }}>
          <div style={{ flex: 1, minWidth: 0 }}>
            <h2
              style={{
                margin: 0,
                fontFamily: 'var(--font-display)',
                fontSize: 'var(--text-heading)',
                fontWeight: 700,
                color: 'var(--text-primary)',
              }}
            >
              Add collection
            </h2>
            {subtitle && (
              <div
                className="ch-mono"
                style={{ fontSize: '0.66rem', color: 'var(--text-tertiary)', marginTop: 3 }}
              >
                {subtitle}
              </div>
            )}
          </div>
          <IconButton icon="x" label="Cancel" variant="ghost" size="sm" onClick={onClose} />
        </div>

        <p
          style={{
            margin: 0,
            fontSize: '0.82rem',
            color: 'var(--text-secondary)',
            lineHeight: 1.5,
          }}
        >
          A collection joins the list as one group, in its own order. Editing the collection later
          updates it here too.
        </p>

        <Input
          icon="search"
          placeholder="Filter collections…"
          aria-label="Filter collections"
          value={query}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => setQuery(e.target.value)}
        />

        <div
          style={{ flex: 1, minHeight: 0, overflowY: 'auto', margin: '0 -4px', padding: '0 4px' }}
        >
          {collections.isLoading ? (
            <Hint>Loading collections…</Hint>
          ) : filtered.length === 0 ? (
            <Hint>
              {(collections.data ?? []).length === 0
                ? 'You haven’t made any collections yet.'
                : `No collections match “${query.trim()}”.`}
            </Hint>
          ) : (
            filtered.map((c) => {
              const added = existingIds.has(c.id);
              const checked = selected.has(c.id);
              return (
                <button
                  key={c.id}
                  type="button"
                  disabled={added}
                  onClick={() => toggle(c)}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 10,
                    width: '100%',
                    padding: '6px 8px',
                    textAlign: 'left',
                    background: 'transparent',
                    border: 'none',
                    borderRadius: 'var(--radius-sm)',
                    color: 'var(--text-primary)',
                    opacity: added ? 0.55 : 1,
                    cursor: added ? 'default' : 'pointer',
                  }}
                >
                  <Cover client={client} bookId={c.coverBookId} />
                  <span style={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}>
                    <span
                      style={{
                        fontSize: 'var(--text-small)',
                        whiteSpace: 'nowrap',
                        overflow: 'hidden',
                        textOverflow: 'ellipsis',
                      }}
                    >
                      {c.name}
                    </span>
                    <span
                      className="ch-mono"
                      style={{ fontSize: 'var(--text-label)', color: 'var(--text-tertiary)' }}
                    >
                      {c.bookCount} issue{c.bookCount === 1 ? '' : 's'}
                    </span>
                  </span>
                  {added ? (
                    <span style={{ fontSize: 'var(--text-label)', color: 'var(--text-tertiary)' }}>
                      Added
                    </span>
                  ) : (
                    <span
                      aria-hidden
                      style={{
                        width: 20,
                        height: 20,
                        borderRadius: 6,
                        border: `1px solid ${checked ? 'var(--accent)' : 'var(--border-strong)'}`,
                        background: checked ? 'var(--accent)' : 'transparent',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                      }}
                    >
                      {checked && <Icon name="check" size={13} color="var(--text-on-accent)" />}
                    </span>
                  )}
                </button>
              );
            })
          )}
        </div>

        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            gap: 10,
            borderTop: '1px solid var(--border-hairline)',
            paddingTop: 12,
          }}
        >
          <Badge tone="neutral" mono>
            {selected.size} selected
          </Badge>
          <div style={{ display: 'flex', gap: 10 }}>
            <Button variant="ghost" onClick={onClose} disabled={busy}>
              Cancel
            </Button>
            <Button
              variant="primary"
              icon="plus"
              onClick={submit}
              disabled={selected.size === 0 || busy}
            >
              {busy ? 'Adding…' : 'Add to list'}
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
