import { useEffect, useState } from 'react';
import { Button, Input, Icon, IconButton, EmptyState } from '@comichub/ui';
import type { BookCard } from '@comichub/api-client';
import { Page, LoadingState, ErrorState } from './Page.js';
import { CoverGrid } from './CoverGrid.js';
import { BookCover } from './cards.js';
import { useSeriesNames } from '../lib/queries.js';

/** A row in a lists index: name on the left, item count + chevron on the right. */
export interface IndexItem {
  id: string;
  name: string;
  bookCount: number;
}

/** Generic index screen for collections / reading lists: create bar + clickable rows. */
export function ListIndexScreen({
  eyebrow,
  title,
  items,
  isLoading,
  isError,
  errorMessage,
  onRetry,
  onOpen,
  onCreate,
  creating,
  createPlaceholder,
  emptyText,
}: {
  eyebrow: string;
  title: string;
  items: IndexItem[] | undefined;
  isLoading: boolean;
  isError: boolean;
  errorMessage: string;
  onRetry: () => void;
  onOpen: (id: string) => void;
  onCreate: (name: string) => void;
  creating: boolean;
  createPlaceholder: string;
  emptyText: string;
}) {
  const [name, setName] = useState('');
  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    const n = name.trim();
    if (!n || creating) return;
    onCreate(n);
    setName('');
  };

  return (
    <Page eyebrow={eyebrow} title={title}>
      <form onSubmit={submit} style={{ display: 'flex', gap: 10, maxWidth: 520, marginBottom: 24 }}>
        <div style={{ flex: 1, minWidth: 0 }}>
          <Input
            icon="plus"
            placeholder={createPlaceholder}
            value={name}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) => setName(e.target.value)}
          />
        </div>
        <Button type="submit" variant="secondary" disabled={!name.trim() || creating}>
          {creating ? 'Creating…' : 'Create'}
        </Button>
      </form>

      {isLoading ? (
        <LoadingState />
      ) : isError ? (
        <ErrorState message={errorMessage} onRetry={onRetry} />
      ) : !items || items.length === 0 ? (
        <EmptyState title="Nothing here yet">{emptyText}</EmptyState>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8, maxWidth: 720 }}>
          {items.map((it) => (
            <ListRow key={it.id} item={it} onOpen={() => onOpen(it.id)} />
          ))}
        </div>
      )}
    </Page>
  );
}

/** A clickable index row: name, item count, chevron. Shared by every lists index. */
export function ListRow({ item, onOpen }: { item: IndexItem; onOpen: () => void }) {
  const [hover, setHover] = useState(false);
  return (
    <button
      type="button"
      onClick={onOpen}
      onMouseEnter={() => setHover(true)}
      onMouseLeave={() => setHover(false)}
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 12,
        width: '100%',
        padding: '14px 16px',
        textAlign: 'left',
        cursor: 'pointer',
        background: hover ? 'var(--surface-card)' : 'var(--surface-raised)',
        border: '1px solid var(--border-hairline)',
        borderRadius: 'var(--radius-md)',
        color: 'var(--text-primary)',
      }}
    >
      <span style={{ flex: 1, minWidth: 0, fontWeight: 600 }}>{item.name}</span>
      <span
        className="ch-mono"
        style={{ fontSize: 'var(--text-small)', color: 'var(--text-tertiary)' }}
      >
        {item.bookCount} {item.bookCount === 1 ? 'issue' : 'issues'}
      </span>
      <Icon name="chevron-right" size={16} color="var(--text-tertiary)" />
    </button>
  );
}

/** Generic detail screen: header with delete, then a removable cover grid of its books. */
export function ListDetailScreen({
  eyebrow,
  title,
  bookCount,
  books,
  isLoading,
  isError,
  errorMessage,
  onRetry,
  onBack,
  onDelete,
  onRemoveBook,
  emptyText,
}: {
  eyebrow: string;
  title: string;
  bookCount: number;
  books: BookCard[] | undefined;
  isLoading: boolean;
  isError: boolean;
  errorMessage: string;
  onRetry: () => void;
  onBack: () => void;
  onDelete: () => void;
  /** When provided, each cover gets a corner remove button. Omit for rule-derived lists. */
  onRemoveBook?: (bookId: string) => void;
  emptyText: string;
}) {
  const seriesNames = useSeriesNames();
  const colW = 168;
  const rowHeight = colW * 1.5 + 38;

  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <div
        style={{
          flex: 'none',
          padding: 'var(--pad-screen) var(--pad-screen) 16px',
          maxWidth: 'var(--content-max)',
          margin: '0 auto',
          width: '100%',
        }}
      >
        <button
          type="button"
          onClick={onBack}
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: 6,
            background: 'none',
            border: 'none',
            padding: 0,
            marginBottom: 18,
            cursor: 'pointer',
            color: 'var(--text-secondary)',
            fontSize: 'var(--text-small)',
          }}
        >
          ← Back
        </button>
        <div
          style={{
            display: 'flex',
            alignItems: 'flex-end',
            justifyContent: 'space-between',
            gap: 16,
          }}
        >
          <div style={{ minWidth: 0 }}>
            <div
              className="ch-mono"
              style={{
                fontSize: 'var(--text-label)',
                textTransform: 'uppercase',
                letterSpacing: 'var(--tracking-label)',
                color: 'var(--text-tertiary)',
                marginBottom: 6,
              }}
            >
              {eyebrow} · {bookCount} {bookCount === 1 ? 'issue' : 'issues'}
            </div>
            <h1
              style={{
                margin: 0,
                fontFamily: 'var(--font-display)',
                fontSize: 'var(--text-display-l)',
                lineHeight: 'var(--leading-display-l)',
                fontWeight: 800,
                letterSpacing: 'var(--tracking-tight)',
                color: 'var(--text-primary)',
                whiteSpace: 'nowrap',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
              }}
            >
              {title}
            </h1>
          </div>
          <Button variant="ghost" icon="trash" onClick={onDelete}>
            Delete
          </Button>
        </div>
      </div>

      <div style={{ flex: 1, minHeight: 0 }}>
        {isLoading ? (
          <LoadingState />
        ) : isError ? (
          <ErrorState message={errorMessage} onRetry={onRetry} />
        ) : !books || books.length === 0 ? (
          <div style={{ padding: 'var(--pad-screen)' }}>
            <EmptyState title="No issues yet">{emptyText}</EmptyState>
          </div>
        ) : (
          <CoverGrid
            items={books}
            cardWidth={colW}
            rowHeight={rowHeight}
            gap={18}
            getKey={(b) => b.id}
            renderItem={(b) => (
              <div style={{ position: 'relative' }}>
                <BookCover book={b} seriesName={seriesNames.get(b.seriesId)} />
                {onRemoveBook && (
                  <div style={{ position: 'absolute', top: 6, right: 6 }}>
                    <IconButton
                      icon="x"
                      label="Remove from list"
                      variant="solid"
                      size="sm"
                      onClick={() => onRemoveBook(b.id)}
                    />
                  </div>
                )}
              </div>
            )}
          />
        )}
      </div>
    </div>
  );
}

/**
 * Modal to add a book to one of the user's collections or reading lists. Mirrors the
 * AddLibraryDialog overlay pattern (no DS modal primitive yet).
 */
export function AddToListDialog({
  title,
  options,
  onPick,
  onClose,
  busy,
  emptyHint,
}: {
  title: string;
  options: { id: string; name: string }[];
  onPick: (id: string) => void;
  onClose: () => void;
  busy: boolean;
  emptyHint: string;
}) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !busy) onClose();
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [busy, onClose]);

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
        aria-label={title}
        style={{
          width: 'min(420px, 100%)',
          maxHeight: '70vh',
          overflowY: 'auto',
          background: 'var(--surface-raised)',
          border: '1px solid var(--border-strong)',
          borderRadius: 'var(--radius-lg)',
          boxShadow: 'var(--shadow-popover)',
          padding: 20,
          display: 'flex',
          flexDirection: 'column',
          gap: 12,
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
          {title}
        </h2>
        {options.length === 0 ? (
          <p style={{ margin: 0, color: 'var(--text-tertiary)', fontSize: 'var(--text-small)' }}>
            {emptyHint}
          </p>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
            {options.map((o) => (
              <button
                key={o.id}
                type="button"
                disabled={busy}
                onClick={() => onPick(o.id)}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 10,
                  width: '100%',
                  padding: '10px 12px',
                  textAlign: 'left',
                  background: 'var(--surface-card)',
                  border: '1px solid var(--border-hairline)',
                  borderRadius: 'var(--radius-md)',
                  color: 'var(--text-primary)',
                  cursor: busy ? 'default' : 'pointer',
                  fontSize: 'var(--text-body)',
                }}
              >
                <Icon name="plus" size={15} color="var(--text-tertiary)" />
                <span style={{ flex: 1, minWidth: 0 }}>{o.name}</span>
              </button>
            ))}
          </div>
        )}
        <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: 4 }}>
          <Button type="button" variant="ghost" onClick={onClose} disabled={busy}>
            Close
          </Button>
        </div>
      </div>
    </div>
  );
}
