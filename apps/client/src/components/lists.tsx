import { useEffect } from 'react';
import { Button, Icon, IconButton, EmptyState } from '@comichub/ui';
import type { BookCard } from '@comichub/api-client';
import { LoadingState, ErrorState } from './Page.js';
import { CoverGrid } from './CoverGrid.js';
import { BookCover } from './cards.js';
import { useSeriesNames } from '../lib/queries.js';
import { relativeTime } from '../lib/format.js';

/**
 * A card in the longbox-style list index (collections, reading lists, smart lists): a fanned
 * cover collage, name, issue count + read %, a progress bar, and "updated" time. Optional
 * `topLeft`/`topRight` pills overlay the covers (e.g. reading-list "In queue" / "N missing").
 */
export interface ListCardModel {
  id: string;
  name: string;
  bookCount: number;
  readPct: number;
  covers: string[];
  updatedAt: number;
  topLeft?: React.ReactNode;
  topRight?: React.ReactNode;
}

/** Read %, and up to four cover urls, derived from a set of books. Shared by every index. */
export function cardCoversAndPct(
  books: BookCard[],
  coverUrl: (bookId: string) => string,
): { readPct: number; covers: string[] } {
  const read = books.filter((b) => b.progress?.status === 'read').length;
  const readPct = books.length ? Math.round((read / books.length) * 100) : 0;
  return { readPct, covers: books.slice(0, 4).map((b) => coverUrl(b.id)) };
}

/** The responsive card grid every list index shares. `createTile` closes the grid. */
export function ListCardGrid({
  cards,
  onOpen,
  createTile,
}: {
  cards: ListCardModel[];
  onOpen: (id: string) => void;
  createTile?: React.ReactNode;
}) {
  return (
    <div
      style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fill, minmax(258px, 1fr))',
        gap: 20,
      }}
    >
      {cards.map((c) => (
        <LongboxCard key={c.id} card={c} onOpen={() => onOpen(c.id)} />
      ))}
      {createTile}
    </div>
  );
}

function LongboxCard({ card, onOpen }: { card: ListCardModel; onOpen: () => void }) {
  return (
    <button
      type="button"
      onClick={onOpen}
      style={{
        display: 'block',
        textAlign: 'left',
        padding: 0,
        cursor: 'pointer',
        background: 'var(--surface-raised)',
        border: '1px solid var(--border-hairline)',
        borderRadius: 'var(--radius-lg)',
        overflow: 'hidden',
        transition: 'border-color 120ms',
      }}
      onMouseEnter={(e) => (e.currentTarget.style.borderColor = 'var(--border-strong)')}
      onMouseLeave={(e) => (e.currentTarget.style.borderColor = 'var(--border-hairline)')}
    >
      <div style={{ position: 'relative' }}>
        <CoverFan covers={card.covers} />
        {card.topLeft && (
          <div style={{ position: 'absolute', top: 10, left: 10 }}>{card.topLeft}</div>
        )}
        {card.topRight && (
          <div style={{ position: 'absolute', top: 10, right: 10 }}>{card.topRight}</div>
        )}
      </div>
      <div style={{ padding: '14px 16px 16px' }}>
        <div
          style={{
            fontFamily: 'var(--font-display)',
            fontWeight: 700,
            fontSize: '1.02rem',
            letterSpacing: '-0.01em',
            color: 'var(--paper-100)',
            whiteSpace: 'nowrap',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
          }}
        >
          {card.name}
        </div>
        <div
          className="ch-mono"
          style={{
            fontSize: '0.66rem',
            color: 'var(--text-tertiary)',
            marginTop: 6,
            display: 'flex',
            alignItems: 'center',
            gap: 8,
          }}
        >
          <span>
            {card.bookCount} issue{card.bookCount === 1 ? '' : 's'}
          </span>
          <span
            style={{ width: 3, height: 3, borderRadius: '50%', background: 'var(--paper-600)' }}
          />
          <span>{card.readPct}% read</span>
        </div>
        <div className="ch-progress" style={{ borderRadius: 999, height: 5, marginTop: 12 }}>
          <span style={{ width: `${card.readPct}%`, borderRadius: 999 }} />
        </div>
        <div
          className="ch-mono"
          style={{ fontSize: '0.6rem', color: 'var(--paper-600)', marginTop: 10 }}
        >
          Updated {relativeTime(card.updatedAt)}
        </div>
      </div>
    </button>
  );
}

/** A pill overlaid on a card's cover fan (reading-list "In queue" / "N missing"). */
export function CardPill({
  tone,
  icon,
  children,
}: {
  tone: 'accent' | 'warning';
  icon: React.ComponentProps<typeof Icon>['name'];
  children: React.ReactNode;
}) {
  const accent = tone === 'accent';
  return (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 5,
        height: 22,
        padding: '0 9px',
        borderRadius: 'var(--radius-pill)',
        background: accent ? 'var(--accent)' : 'color-mix(in oklab, var(--warning) 92%, #000)',
        color: accent ? 'var(--text-on-accent)' : 'var(--ink-900)',
        fontFamily: 'var(--font-mono)',
        fontSize: '0.58rem',
        letterSpacing: '0.05em',
        textTransform: 'uppercase',
        boxShadow: '0 2px 8px rgba(0,0,0,.4)',
      }}
    >
      <Icon name={icon} size={11} color={accent ? 'var(--text-on-accent)' : 'var(--ink-900)'} />
      {children}
    </span>
  );
}

/** Up to four covers fanned like pulls from a longbox (per the design preview). */
export function CoverFan({ covers }: { covers: string[] }) {
  const arr = covers.slice(0, 4);
  const mid = (arr.length - 1) / 2;
  return (
    <div
      style={{
        position: 'relative',
        height: 158,
        background: 'linear-gradient(180deg, var(--ink-900), var(--ink-800))',
        overflow: 'hidden',
      }}
    >
      {arr.length === 0 && (
        <div
          style={{
            position: 'absolute',
            inset: 0,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}
        >
          <Icon name="book-open" size={22} color="var(--paper-600)" />
        </div>
      )}
      {arr.map((c, i) => {
        const off = i - mid;
        return (
          <img
            key={i}
            src={c}
            alt=""
            style={{
              position: 'absolute',
              left: '50%',
              top: 20,
              width: 86,
              height: 129,
              objectFit: 'cover',
              transform: `translateX(calc(-50% + ${off * 40}px)) rotate(${off * 6}deg)`,
              transformOrigin: 'bottom center',
              boxShadow: '0 8px 22px rgba(0,0,0,.55)',
              outline: '1px solid rgba(0,0,0,.45)',
              zIndex: i,
            }}
          />
        );
      })}
    </div>
  );
}

/** The dashed "create" tile that closes a list card grid. */
export function CreateTile({
  label,
  hint,
  onClick,
}: {
  label: string;
  hint: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        gap: 10,
        minHeight: 260,
        cursor: 'pointer',
        background: 'transparent',
        border: '1px dashed var(--border-strong)',
        borderRadius: 'var(--radius-lg)',
        color: 'var(--text-tertiary)',
      }}
      onMouseEnter={(e) => {
        e.currentTarget.style.borderColor = 'var(--accent)';
        e.currentTarget.style.color = 'var(--accent)';
      }}
      onMouseLeave={(e) => {
        e.currentTarget.style.borderColor = 'var(--border-strong)';
        e.currentTarget.style.color = 'var(--text-tertiary)';
      }}
    >
      <span
        style={{
          width: 44,
          height: 44,
          borderRadius: '50%',
          border: '1px solid currentColor',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
        }}
      >
        <Icon name="plus" size={20} color="currentColor" />
      </span>
      <span style={{ fontFamily: 'var(--font-body)', fontWeight: 600, fontSize: '0.9rem' }}>
        {label}
      </span>
      <span className="ch-mono" style={{ fontSize: '0.62rem', color: 'var(--paper-600)' }}>
        {hint}
      </span>
    </button>
  );
}

/** The shared index header: accent eyebrow, display title, a subtitle, and a right-side slot
 *  for the create control (a name field, or a smart-list builder toggle). */
export function ListIndexHeader({
  eyebrow,
  title,
  subtitle,
  right,
}: {
  eyebrow: string;
  title: string;
  subtitle: string;
  right?: React.ReactNode;
}) {
  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'flex-end',
        gap: 24,
        flexWrap: 'wrap',
        marginBottom: 24,
      }}
    >
      <div style={{ flex: 1, minWidth: 280 }}>
        <div
          className="ch-mono"
          style={{
            fontSize: '0.66rem',
            fontWeight: 600,
            letterSpacing: '0.16em',
            textTransform: 'uppercase',
            color: 'var(--accent)',
          }}
        >
          {eyebrow}
        </div>
        <h1
          style={{
            margin: '8px 0 0',
            fontFamily: 'var(--font-display)',
            fontWeight: 800,
            fontSize: 'var(--text-display-l)',
            letterSpacing: '-0.01em',
            color: 'var(--text-primary)',
          }}
        >
          {title}
        </h1>
        <p className="ch-label" style={{ margin: '8px 0 0', color: 'var(--text-tertiary)' }}>
          {subtitle}
        </p>
      </div>
      {right && (
        <div style={{ flex: 'none', display: 'flex', gap: 8, alignItems: 'center' }}>{right}</div>
      )}
    </div>
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
  onAddIssues,
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
  /** When provided, an "Add issues" button opens a search-and-multi-select picker. */
  onAddIssues?: () => void;
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
          <div style={{ flex: 'none', display: 'flex', gap: 10 }}>
            {onAddIssues && (
              <Button variant="secondary" icon="plus" onClick={onAddIssues}>
                Add issues
              </Button>
            )}
            <Button variant="ghost" icon="trash" onClick={onDelete}>
              Delete
            </Button>
          </div>
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
