import { useRef, useState } from 'react';
import { getRouteApi, useNavigate } from '@tanstack/react-router';
import { Button, Badge, Icon, IconButton, EmptyState } from '@comichub/ui';
import type {
  BookCard,
  ReadingListDetail as ReadingListDetailData,
  ReadingListEntry,
} from '@comichub/api-client';
import { useClient } from '../lib/client.js';
import {
  useReadingList,
  useDeleteReadingList,
  useRemoveFromReadingList,
  useAddToReadingList,
  useAddManualToReadingList,
  useRelinkReadingListItem,
  useReorderReadingList,
  useSetActiveReadingList,
  useSeriesNames,
} from '../lib/queries.js';
import { useReadLaunch } from '../lib/launch.js';
import { LoadingState, ErrorState } from '../components/Page.js';
import { AddIssuesDialog, MissingPill } from '../components/AddIssuesDialog.js';
import { LinkIssueDialog } from '../components/LinkIssueDialog.js';
import { issueLabel, relativeTime, resumePage } from '../lib/format.js';

const route = getRouteApi('/reading-lists/$id');

type RState = 'read' | 'reading' | 'unread';

/** Per-state spine-tab colors + the row action verb/variant (from the design handoff). */
const RL_STATE: Record<
  RState,
  {
    tab: string;
    tabText: string;
    label: string;
    action: string;
    variant: 'ghost' | 'primary' | 'secondary';
  }
> = {
  read: {
    tab: 'var(--ink-800)',
    tabText: 'var(--paper-400)',
    label: 'Read',
    action: 'Reread',
    variant: 'ghost',
  },
  reading: {
    tab: 'var(--accent)',
    tabText: 'var(--text-on-accent)',
    label: 'In progress',
    action: 'Resume',
    variant: 'primary',
  },
  unread: {
    tab: 'var(--unread)',
    tabText: 'var(--text-on-accent)',
    label: 'Unread',
    action: 'Read',
    variant: 'secondary',
  },
};

function stateOf(b: BookCard): RState {
  const s = b.progress?.status;
  return s === 'read' ? 'read' : s === 'in_progress' ? 'reading' : 'unread';
}

/** A reading list as an ordered, drag-to-reorder queue (matches the design handoff). */
export function ReadingListDetail() {
  const { id } = route.useParams();
  const q = useReadingList(id);

  if (q.isLoading) {
    return (
      <div style={{ padding: 'var(--pad-screen)' }}>
        <LoadingState />
      </div>
    );
  }
  if (q.isError || !q.data) {
    return (
      <div style={{ padding: 'var(--pad-screen)' }}>
        <ErrorState
          message={q.error instanceof Error ? q.error.message : 'Could not load this reading list.'}
          onRetry={() => q.refetch()}
        />
      </div>
    );
  }

  return <QueueView data={q.data} listId={id} />;
}

function QueueView({ data, listId }: { data: ReadingListDetailData; listId: string }) {
  const navigate = useNavigate();
  const client = useClient();
  const launch = useReadLaunch();
  const seriesNames = useSeriesNames();
  const del = useDeleteReadingList();
  const removeItem = useRemoveFromReadingList();
  const reorder = useReorderReadingList();
  const setActive = useSetActiveReadingList();
  const add = useAddToReadingList();
  const addManual = useAddManualToReadingList();
  const relink = useRelinkReadingListItem();
  const [adding, setAdding] = useState(false);
  const [linking, setLinking] = useState<ReadingListEntry | null>(null);

  const { readingList, items } = data;
  // Linked entries carry a BookCard; stale placeholders hold their slot but can't be read.
  const books = items.flatMap((it) => (it.book ? [it.book] : []));
  const staleCount = items.length - books.length;

  // Smart resume target + next unread for the header CTAs.
  const nextItem =
    books.find((b) => stateOf(b) === 'reading') ?? books.find((b) => stateOf(b) === 'unread');
  const firstUnread = books.find((b) => stateOf(b) === 'unread');
  const hasReading = books.some((b) => stateOf(b) === 'reading');

  // Overall progress (by pages): read = all pages, reading = current page, unread = 0.
  const readCount = books.filter((b) => stateOf(b) === 'read').length;
  const readingCount = books.filter((b) => stateOf(b) === 'reading').length;
  const toGo = books.length - readCount - readingCount;
  const totalPages = books.reduce((a, b) => a + (b.pageCount || 0), 0);
  const pagesRead = books.reduce((a, b) => {
    const s = stateOf(b);
    return a + (s === 'read' ? b.pageCount || 0 : s === 'reading' ? b.progress?.page || 0 : 0);
  }, 0);
  const pct = totalPages ? Math.round((pagesRead / totalPages) * 100) : 0;
  const mins = Math.round((totalPages - pagesRead) * 0.4);
  const timeLeft =
    mins >= 60 ? `${Math.floor(mins / 60)}h ${String(mins % 60).padStart(2, '0')}m` : `${mins}m`;

  const openReader = (b: BookCard) =>
    launch(b.id, stateOf(b) === 'reading' ? resumePage(b.progress) : 0);
  const openIssue = (b: BookCard) => navigate({ to: '/book/$id', params: { id: b.id } });

  // Native drag-and-drop. The dragged index + insertion slot live in refs so the drop
  // handler never reads stale state; the slot also drives the insertion indicator.
  const dragRef = useRef<number | null>(null);
  const overRef = useRef<number | null>(null);
  const [drag, setDrag] = useState<number | null>(null);
  const [over, setOver] = useState<number | null>(null);

  const setOverSlot = (slot: number | null) => {
    overRef.current = slot;
    setOver(slot);
  };
  const startDrag = (idx: number) => {
    dragRef.current = idx;
    setDrag(idx);
  };
  const endDrag = () => {
    dragRef.current = null;
    overRef.current = null;
    setDrag(null);
    setOver(null);
  };
  const drop = () => {
    const d = dragRef.current;
    const o = overRef.current;
    endDrag();
    if (d == null || o == null) return;
    // Reorder by entry id so stale placeholders move like any other row.
    const beforeId = o < items.length ? items[o]!.id : undefined; // undefined → move to end
    if (beforeId === items[d]!.id) return; // dropped onto itself
    reorder.mutate({ id: listId, bookId: items[d]!.id, beforeId });
  };

  const onDelete = async () => {
    if (!window.confirm('Delete this reading list?')) return;
    await del.mutateAsync(listId);
    navigate({ to: '/reading-lists' });
  };

  return (
    <div style={{ padding: 'var(--pad-screen)', maxWidth: 'var(--content-max)', margin: '0 auto' }}>
      {adding && (
        <AddIssuesDialog
          title="Add issues"
          subtitle={readingList.name}
          existingIds={new Set(books.map((b) => b.id))}
          onAdd={(bookIds) => add.mutateAsync({ id: listId, bookIds })}
          onAddManual={(manual) => addManual.mutateAsync({ id: listId, manual })}
          onClose={() => setAdding(false)}
        />
      )}
      {linking && (
        <LinkIssueDialog
          entry={linking}
          existingIds={new Set(books.map((b) => b.id))}
          onLink={(bookId) => relink.mutateAsync({ id: listId, itemId: linking.id, bookId })}
          onClose={() => setLinking(null)}
        />
      )}

      <button
        type="button"
        onClick={() => navigate({ to: '/reading-lists' })}
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
        ← Reading lists
      </button>

      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'flex-end', gap: 24, flexWrap: 'wrap' }}>
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
            Reading list
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 10, margin: '8px 0 0' }}>
            <h1
              style={{
                margin: 0,
                fontFamily: 'var(--font-display)',
                fontWeight: 800,
                fontSize: 'var(--text-display-l)',
                letterSpacing: '-0.01em',
                color: 'var(--text-primary)',
              }}
            >
              {readingList.name}
            </h1>
            {readingList.active && (
              <Badge tone="accent" mono dot>
                Active
              </Badge>
            )}
          </div>
        </div>
        <div style={{ flex: 'none', display: 'flex', gap: 10, flexWrap: 'wrap' }}>
          {nextItem && (
            <Button icon="book-open" onClick={() => openReader(nextItem)}>
              {hasReading
                ? `Resume · ${issueLabel(nextItem.number) ?? ''}`
                : `Start · ${issueLabel(nextItem.number) ?? ''}`}
            </Button>
          )}
          {firstUnread && (
            <Button variant="secondary" onClick={() => openReader(firstUnread)}>
              Next unread →
            </Button>
          )}
          {!readingList.active && (
            <Button variant="ghost" icon="check" onClick={() => setActive.mutate(listId)}>
              Set active
            </Button>
          )}
          <Button variant="ghost" icon="plus" onClick={() => setAdding(true)}>
            Add issues
          </Button>
          <Button variant="ghost" icon="trash" onClick={onDelete}>
            Delete
          </Button>
        </div>
      </div>

      {/* Stats + overall progress */}
      <div
        style={{
          marginTop: 22,
          padding: '16px 18px',
          background: 'var(--surface-raised)',
          border: '1px solid var(--border-hairline)',
          borderRadius: 8,
        }}
      >
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 16,
            flexWrap: 'wrap',
            marginBottom: 12,
          }}
        >
          <span className="ch-mono" style={{ fontSize: '0.74rem', color: 'var(--text-primary)' }}>
            {items.length} issues
          </span>
          {staleCount > 0 && (
            <span
              className="ch-mono"
              style={{
                display: 'inline-flex',
                alignItems: 'center',
                gap: 6,
                fontSize: '0.74rem',
                color: 'var(--warning)',
              }}
            >
              <Icon name="alert-triangle" size={13} color="var(--warning)" /> {staleCount} missing
            </span>
          )}
          <span style={{ width: 1, height: 14, background: 'var(--border-hairline)' }} />
          <span className="ch-mono" style={{ fontSize: '0.74rem', color: 'var(--paper-400)' }}>
            {readCount} read · {readingCount} reading · {toGo} to go
          </span>
          <span style={{ width: 1, height: 14, background: 'var(--border-hairline)' }} />
          <span
            className="ch-mono"
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: 6,
              fontSize: '0.74rem',
              color: 'var(--paper-400)',
            }}
          >
            <Icon name="clock" size={13} color="var(--paper-400)" /> ~{timeLeft} left
          </span>
          <div style={{ flex: 1 }} />
          <Badge tone="accent" mono>
            {pct}% complete
          </Badge>
        </div>
        <div className="ch-progress" style={{ borderRadius: 999, height: 6 }}>
          <span style={{ width: `${pct}%`, borderRadius: 999 }} />
        </div>
      </div>

      {/* Reading-order toolbar */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, margin: '24px 0 6px' }}>
        <span
          className="ch-mono"
          style={{
            fontSize: '0.62rem',
            fontWeight: 600,
            letterSpacing: '0.16em',
            textTransform: 'uppercase',
            color: 'var(--paper-600)',
          }}
        >
          Reading order
        </span>
        <span style={{ flex: 1, height: 1, background: 'var(--border-hairline)' }} />
        <span
          className="ch-mono"
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: 6,
            fontSize: '0.64rem',
            color: 'var(--paper-600)',
          }}
        >
          <Grip /> Drag to reorder
        </span>
      </div>

      {/* Rows with insertion indicators */}
      {items.length === 0 ? (
        <EmptyState title="This queue is empty">
          Use “Add issues” to build your queue, then drag issues to reorder them.
        </EmptyState>
      ) : (
        <div onDragOver={(e) => e.preventDefault()}>
          {items.map((it, idx) => {
            const dragHandlers = {
              onGripDragStart: (e: React.DragEvent) => {
                e.dataTransfer.effectAllowed = 'move';
                e.dataTransfer.setData('text/plain', String(idx));
                startDrag(idx);
              },
              onGripDragEnd: endDrag,
              onRowDragOver: (e: React.DragEvent<HTMLDivElement>) => {
                e.preventDefault();
                e.dataTransfer.dropEffect = 'move';
                const r = e.currentTarget.getBoundingClientRect();
                setOverSlot(e.clientY < r.top + r.height / 2 ? idx : idx + 1);
              },
              onRowDrop: (e: React.DragEvent<HTMLDivElement>) => {
                e.preventDefault();
                drop();
              },
            };
            return (
              <div key={it.id}>
                <DropLine active={drag != null && over === idx} />
                {it.book ? (
                  <ReadingRow
                    book={it.book}
                    order={String(idx + 1).padStart(2, '0')}
                    cover={client.coverUrl(it.book.id, 80)}
                    seriesName={seriesNames.get(it.book.seriesId)}
                    dragging={drag === idx}
                    onOpenIssue={() => openIssue(it.book!)}
                    onOpenReader={() => openReader(it.book!)}
                    onRemove={() => removeItem.mutate({ id: listId, bookId: it.id })}
                    {...dragHandlers}
                  />
                ) : (
                  <StaleRow
                    entry={it}
                    order={String(idx + 1).padStart(2, '0')}
                    dragging={drag === idx}
                    onLink={() => setLinking(it)}
                    onRemove={() => removeItem.mutate({ id: listId, bookId: it.id })}
                    {...dragHandlers}
                  />
                )}
              </div>
            );
          })}
          <DropLine active={drag != null && over === items.length} />
        </div>
      )}
    </div>
  );
}

function DropLine({ active }: { active: boolean }) {
  return (
    <div
      style={{
        height: 2,
        margin: '1px 8px',
        borderRadius: 2,
        background: active ? 'var(--accent)' : 'transparent',
      }}
    />
  );
}

/** A 2×3 dot grip — the drag affordance from the design handoff. */
function Grip({ active }: { active?: boolean }) {
  return (
    <span
      aria-hidden
      style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 3px)', gap: 3, cursor: 'grab' }}
    >
      {Array.from({ length: 6 }).map((_, i) => (
        <span
          key={i}
          style={{
            width: 3,
            height: 3,
            borderRadius: '50%',
            background: active ? 'var(--paper-100)' : 'var(--paper-600)',
          }}
        />
      ))}
    </span>
  );
}

/**
 * A kept placeholder row (per the design preview): no cover, no progress, no Read action.
 * It holds its slot and drag handle, renders from its snapshot, and offers Link issue +
 * Remove. Quiet warning treatment — intentionally kept, not broken.
 */
function StaleRow({
  entry,
  order,
  dragging,
  onLink,
  onRemove,
  onGripDragStart,
  onGripDragEnd,
  onRowDragOver,
  onRowDrop,
}: {
  entry: ReadingListEntry;
  order: string;
  dragging: boolean;
  onLink: () => void;
  onRemove: () => void;
  onGripDragStart: (e: React.DragEvent) => void;
  onGripDragEnd: () => void;
  onRowDragOver: (e: React.DragEvent<HTMLDivElement>) => void;
  onRowDrop: (e: React.DragEvent<HTMLDivElement>) => void;
}) {
  const [hover, setHover] = useState(false);
  const label =
    [entry.seriesName, issueLabel(entry.number)].filter(Boolean).join(' ') ||
    entry.title ||
    'Unknown issue';
  return (
    <div
      onMouseEnter={() => setHover(true)}
      onMouseLeave={() => setHover(false)}
      onDragOver={onRowDragOver}
      onDrop={onRowDrop}
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 14,
        padding: '10px 12px 10px 8px',
        borderRadius: 6,
        background: hover ? 'var(--ink-700)' : 'transparent',
        opacity: dragging ? 0.35 : 1,
        border: '1px solid',
        borderColor: hover ? 'var(--border-hairline)' : 'transparent',
        transition: 'background 110ms',
      }}
    >
      <span
        draggable
        onDragStart={onGripDragStart}
        onDragEnd={onGripDragEnd}
        title="Drag to reorder"
        style={{ flex: 'none', padding: '6px 2px' }}
      >
        <Grip active={hover} />
      </span>

      <span
        className="ch-mono"
        style={{
          flex: 'none',
          width: 22,
          textAlign: 'right',
          fontSize: '0.78rem',
          fontVariantNumeric: 'tabular-nums',
          color: 'var(--paper-600)',
        }}
      >
        {order}
      </span>

      {/* Dashed "missing" cover slot — quietly kept, not broken */}
      <span
        aria-hidden
        style={{
          flex: 'none',
          width: 46,
          height: 69,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          border: '1px dashed var(--border-strong)',
          borderRadius: 3,
          background:
            'repeating-linear-gradient(135deg, transparent, transparent 5px, color-mix(in oklab, var(--ink-600) 55%, transparent) 5px, color-mix(in oklab, var(--ink-600) 55%, transparent) 6px)',
        }}
      >
        <Icon name="book-open" size={16} color="var(--paper-600)" />
      </span>

      {/* Snapshot title block */}
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
          <span
            style={{
              fontFamily: 'var(--font-body)',
              fontWeight: 600,
              fontSize: '0.95rem',
              color: 'var(--paper-400)',
              whiteSpace: 'nowrap',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              maxWidth: 260,
            }}
          >
            {label}
          </span>
          <MissingPill />
        </div>
        <div
          className="ch-mono"
          style={{
            fontSize: '0.68rem',
            color: 'var(--paper-600)',
            marginTop: 4,
            whiteSpace: 'nowrap',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
          }}
        >
          {entry.title && entry.seriesName ? `${entry.title} · ` : ''}
          Kept from snapshot · added {relativeTime(entry.addedAt)}
        </div>
      </div>

      {/* Actions — link to a real issue, or remove. No Read. */}
      <div
        style={{
          flex: 'none',
          display: 'flex',
          alignItems: 'center',
          gap: 4,
          opacity: hover ? 1 : 0.85,
        }}
      >
        <Button size="sm" variant="secondary" icon="link" onClick={onLink}>
          Link issue
        </Button>
        <IconButton
          icon="trash"
          label="Remove from list"
          variant="ghost"
          size="sm"
          onClick={onRemove}
        />
      </div>
    </div>
  );
}

function ReadingRow({
  book,
  order,
  cover,
  seriesName,
  dragging,
  onOpenIssue,
  onOpenReader,
  onRemove,
  onGripDragStart,
  onGripDragEnd,
  onRowDragOver,
  onRowDrop,
}: {
  book: BookCard;
  order: string;
  cover: string;
  seriesName?: string;
  dragging: boolean;
  onOpenIssue: () => void;
  onOpenReader: () => void;
  onRemove: () => void;
  onGripDragStart: (e: React.DragEvent) => void;
  onGripDragEnd: () => void;
  onRowDragOver: (e: React.DragEvent<HTMLDivElement>) => void;
  onRowDrop: (e: React.DragEvent<HTMLDivElement>) => void;
}) {
  const [hover, setHover] = useState(false);
  const st = stateOf(book);
  const meta = RL_STATE[st];
  const pages = book.pageCount || 0;
  const page = book.progress?.page || 0;
  const pct = pages ? Math.round((page / pages) * 100) : 0;
  const title = seriesName || book.title || issueLabel(book.number) || 'Untitled';

  return (
    <div
      onMouseEnter={() => setHover(true)}
      onMouseLeave={() => setHover(false)}
      onDragOver={onRowDragOver}
      onDrop={onRowDrop}
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 14,
        padding: '10px 12px 10px 8px',
        borderRadius: 6,
        background: hover ? 'var(--ink-700)' : 'transparent',
        opacity: dragging ? 0.35 : 1,
        border: '1px solid',
        borderColor: hover ? 'var(--border-hairline)' : 'transparent',
        transition: 'background 110ms',
      }}
    >
      <span
        draggable
        onDragStart={onGripDragStart}
        onDragEnd={onGripDragEnd}
        title="Drag to reorder"
        style={{ flex: 'none', padding: '6px 2px' }}
      >
        <Grip active={hover} />
      </span>

      <span
        className="ch-mono"
        style={{
          flex: 'none',
          width: 22,
          textAlign: 'right',
          fontSize: '0.78rem',
          fontVariantNumeric: 'tabular-nums',
          color: 'var(--paper-600)',
        }}
      >
        {order}
      </span>

      {/* Cover with registration ticks + state spine tab */}
      <div
        className="ch-reg"
        onClick={onOpenIssue}
        style={{ position: 'relative', flex: 'none', cursor: 'pointer' }}
      >
        <img
          src={cover}
          alt=""
          draggable={false}
          style={{
            width: 46,
            height: 69,
            objectFit: 'cover',
            display: 'block',
            boxShadow: '0 2px 8px rgba(0,0,0,.5)',
          }}
        />
        <span
          className="ch-mono"
          style={{
            position: 'absolute',
            left: 0,
            bottom: 0,
            fontSize: '0.52rem',
            fontWeight: 600,
            letterSpacing: '0.02em',
            padding: '1px 7px 1px 4px',
            background: meta.tab,
            color: meta.tabText,
            clipPath: 'polygon(0 0, 100% 0, calc(100% - 6px) 100%, 0 100%)',
          }}
        >
          {issueLabel(book.number) ?? ''}
        </span>
      </div>

      {/* Title block */}
      <div style={{ flex: 1, minWidth: 0, cursor: 'pointer' }} onClick={onOpenIssue}>
        <div style={{ display: 'flex', alignItems: 'baseline', gap: 8 }}>
          <span
            style={{
              fontFamily: 'var(--font-body)',
              fontWeight: 600,
              fontSize: '0.95rem',
              color: 'var(--paper-100)',
              whiteSpace: 'nowrap',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
            }}
          >
            {title}
          </span>
          {issueLabel(book.number) && (
            <span
              className="ch-mono"
              style={{ flex: 'none', fontSize: '0.8rem', color: 'var(--accent)' }}
            >
              {issueLabel(book.number)}
            </span>
          )}
        </div>
        <div
          className="ch-mono"
          style={{ fontSize: '0.68rem', color: 'var(--paper-600)', marginTop: 3 }}
        >
          {[pages ? `${pages} pp` : null, book.format?.toUpperCase()].filter(Boolean).join(' · ')}
        </div>
      </div>

      {/* State / progress */}
      <div style={{ flex: 'none', width: 150 }}>
        {st === 'reading' ? (
          <div>
            <div
              className="ch-mono"
              style={{
                fontSize: '0.66rem',
                color: 'var(--text-secondary)',
                marginBottom: 5,
                display: 'flex',
                justifyContent: 'space-between',
              }}
            >
              <span>
                p.{page}/{pages}
              </span>
              <span style={{ color: 'var(--accent)' }}>{pct}%</span>
            </div>
            <div className="ch-progress" style={{ borderRadius: 999 }}>
              <span style={{ width: `${pct}%`, borderRadius: 999 }} />
            </div>
          </div>
        ) : (
          <div style={{ display: 'flex', alignItems: 'center', gap: 7 }}>
            {st === 'read' ? (
              <Icon name="check" size={15} color="var(--paper-400)" />
            ) : (
              <span
                style={{ width: 7, height: 7, borderRadius: '50%', background: 'var(--unread)' }}
              />
            )}
            <span
              className="ch-mono"
              style={{
                fontSize: '0.68rem',
                letterSpacing: '0.06em',
                textTransform: 'uppercase',
                color: st === 'unread' ? 'var(--unread)' : 'var(--paper-400)',
              }}
            >
              {meta.label}
            </span>
          </div>
        )}
      </div>

      {/* Actions */}
      <div
        style={{
          flex: 'none',
          display: 'flex',
          alignItems: 'center',
          gap: 4,
          opacity: hover ? 1 : 0.85,
        }}
      >
        <Button size="sm" variant={meta.variant} icon="book-open" onClick={onOpenReader}>
          {meta.action}
        </Button>
        <IconButton
          icon="chevron-right"
          label="Issue details"
          variant="ghost"
          size="sm"
          onClick={onOpenIssue}
        />
        <IconButton icon="x" label="Remove" variant="ghost" size="sm" onClick={onRemove} />
      </div>
    </div>
  );
}
