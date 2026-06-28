import { useRef, useState } from 'react';
import { getRouteApi, useNavigate } from '@tanstack/react-router';
import { Button, Badge, Icon, IconButton, EmptyState } from '@comichub/ui';
import type { BookCard, ReadingListDetail as ReadingListDetailData } from '@comichub/api-client';
import { useClient } from '../lib/client.js';
import {
  useReadingList,
  useDeleteReadingList,
  useRemoveFromReadingList,
  useAddToReadingList,
  useReorderReadingList,
  useSetActiveReadingList,
  useSeriesNames,
} from '../lib/queries.js';
import { useReadLaunch } from '../lib/launch.js';
import { LoadingState, ErrorState } from '../components/Page.js';
import { AddIssuesDialog } from '../components/AddIssuesDialog.js';
import { issueLabel, resumePage } from '../lib/format.js';

const route = getRouteApi('/reading-lists/$id');

/** A reading list as an ordered, reorderable queue that can be set as the active list. */
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
  const [adding, setAdding] = useState(false);

  const { readingList, books } = data;
  const next = books.find((b) => b.progress?.status !== 'read') ?? books[0];

  // Native drag-and-drop reordering: drag a row's handle onto another row; dropping in its
  // top/bottom half inserts above/below it. The dragged id lives in a ref so the drop
  // handler never reads stale state; the visual state drives row highlighting.
  const dragRef = useRef<string | null>(null);
  const [dragId, setDragId] = useState<string | null>(null);
  const [over, setOver] = useState<{ id: string; pos: 'before' | 'after' } | null>(null);

  const startDrag = (id: string) => {
    dragRef.current = id;
    setDragId(id);
  };
  const endDrag = () => {
    dragRef.current = null;
    setDragId(null);
    setOver(null);
  };
  const hover = (id: string, pos: 'before' | 'after') => {
    if (dragRef.current && dragRef.current !== id) setOver({ id, pos });
  };
  const dropOn = (id: string, pos: 'before' | 'after') => {
    const dragged = dragRef.current;
    endDrag();
    if (!dragged) return;
    const beforeId = beforeIdForDrop(books, dragged, { id, pos });
    if (beforeId !== dragged) reorder.mutate({ id: listId, bookId: dragged, beforeId });
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
          title="Add issues to reading list"
          existingIds={new Set(books.map((b) => b.id))}
          onAdd={(bookIds) => add.mutateAsync({ id: listId, bookIds })}
          onClose={() => setAdding(false)}
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

      <div
        style={{
          display: 'flex',
          alignItems: 'flex-end',
          justifyContent: 'space-between',
          gap: 16,
          marginBottom: 22,
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
            Reading list · {readingList.bookCount}{' '}
            {readingList.bookCount === 1 ? 'issue' : 'issues'}
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            <h1
              style={{
                margin: 0,
                fontFamily: 'var(--font-display)',
                fontSize: 'var(--text-display-l)',
                lineHeight: 'var(--leading-display-l)',
                fontWeight: 800,
                letterSpacing: 'var(--tracking-tight)',
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
        <div style={{ flex: 'none', display: 'flex', gap: 10 }}>
          {next && (
            <Button
              variant="primary"
              icon="book-open"
              onClick={() => launch(next.id, resumePage(next.progress))}
            >
              {next.progress?.status === 'in_progress' ? 'Continue' : 'Read next'}
            </Button>
          )}
          {!readingList.active && (
            <Button variant="secondary" icon="check" onClick={() => setActive.mutate(listId)}>
              Set active
            </Button>
          )}
          <Button variant="secondary" icon="plus" onClick={() => setAdding(true)}>
            Add issues
          </Button>
          <Button variant="ghost" icon="trash" onClick={onDelete}>
            Delete
          </Button>
        </div>
      </div>

      {books.length === 0 ? (
        <EmptyState title="This queue is empty">
          Use “Add issues” to build your queue, then drag issues to reorder them.
        </EmptyState>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
          {books.map((book, i) => (
            <QueueRow
              key={book.id}
              index={i}
              book={book}
              seriesName={seriesNames.get(book.seriesId)}
              cover={client.coverUrl(book.id, 80)}
              dragging={dragId === book.id}
              over={over?.id === book.id ? over.pos : null}
              onOpen={() => launch(book.id, resumePage(book.progress))}
              onDetails={() => navigate({ to: '/book/$id', params: { id: book.id } })}
              onRemove={() => removeItem.mutate({ id: listId, bookId: book.id })}
              onDragStart={() => startDrag(book.id)}
              onDragEnd={endDrag}
              onDragOver={(pos) => hover(book.id, pos)}
              onDrop={(pos) => dropOn(book.id, pos)}
            />
          ))}
        </div>
      )}
    </div>
  );
}

function QueueRow({
  index,
  book,
  seriesName,
  cover,
  dragging,
  over,
  onOpen,
  onDetails,
  onRemove,
  onDragStart,
  onDragEnd,
  onDragOver,
  onDrop,
}: {
  index: number;
  book: BookCard;
  seriesName?: string;
  cover: string;
  dragging: boolean;
  over: 'before' | 'after' | null;
  onOpen: () => void;
  onDetails: () => void;
  onRemove: () => void;
  onDragStart: () => void;
  onDragEnd: () => void;
  onDragOver: (pos: 'before' | 'after') => void;
  onDrop: (pos: 'before' | 'after') => void;
}) {
  const isRead = book.progress?.status === 'read';
  const meta = [issueLabel(book.number), seriesName].filter(Boolean).join(' · ');
  const accent = '2px solid var(--accent)';
  const posOf = (e: React.DragEvent<HTMLDivElement>): 'before' | 'after' => {
    const r = e.currentTarget.getBoundingClientRect();
    return e.clientY < r.top + r.height / 2 ? 'before' : 'after';
  };
  return (
    <div
      onDragOver={(e) => {
        e.preventDefault();
        e.dataTransfer.dropEffect = 'move';
        onDragOver(posOf(e));
      }}
      onDrop={(e) => {
        e.preventDefault();
        onDrop(posOf(e));
      }}
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 8,
        padding: '4px 8px',
        background: 'var(--surface-raised)',
        border: '1px solid var(--border-hairline)',
        borderTop: over === 'before' ? accent : '1px solid var(--border-hairline)',
        borderBottom: over === 'after' ? accent : '1px solid var(--border-hairline)',
        borderRadius: 'var(--radius-sm)',
        opacity: dragging ? 0.4 : isRead ? 0.6 : 1,
      }}
    >
      <span
        draggable
        onDragStart={(e) => {
          e.dataTransfer.effectAllowed = 'move';
          e.dataTransfer.setData('text/plain', book.id);
          onDragStart();
        }}
        onDragEnd={onDragEnd}
        title="Drag to reorder"
        style={{
          flex: 'none',
          display: 'flex',
          alignItems: 'center',
          gap: 6,
          cursor: 'grab',
          color: 'var(--text-tertiary)',
        }}
      >
        <Icon name="sort" size={14} />
        <span className="ch-mono" style={{ width: 18, textAlign: 'right' }}>
          {index + 1}
        </span>
      </span>
      <button
        type="button"
        onClick={onOpen}
        style={{
          flex: 1,
          minWidth: 0,
          display: 'flex',
          alignItems: 'center',
          gap: 10,
          padding: 0,
          background: 'none',
          border: 'none',
          cursor: 'pointer',
          color: 'var(--text-primary)',
          textAlign: 'left',
        }}
      >
        <img
          src={cover}
          alt=""
          width={24}
          height={36}
          draggable={false}
          style={{
            flex: 'none',
            objectFit: 'cover',
            borderRadius: 'var(--radius-sm)',
            background: 'var(--surface-cover)',
          }}
        />
        <span
          style={{
            flex: 1,
            minWidth: 0,
            fontSize: 'var(--text-small)',
            whiteSpace: 'nowrap',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
          }}
        >
          {book.title || seriesName || issueLabel(book.number) || 'Untitled'}
        </span>
        {meta && (
          <span
            className="ch-mono"
            style={{
              flex: 'none',
              fontSize: 'var(--text-label)',
              color: 'var(--text-tertiary)',
              whiteSpace: 'nowrap',
            }}
          >
            {isRead ? 'Read' : meta}
          </span>
        )}
      </button>
      <IconButton icon="info" label="Issue details" variant="ghost" size="sm" onClick={onDetails} />
      <IconButton icon="x" label="Remove" variant="ghost" size="sm" onClick={onRemove} />
    </div>
  );
}

/** Resolves the "place before" id for a drop: dropping in a row's top half inserts before
 *  it, bottom half after it. Returns undefined to move to the end, or the dragged id itself
 *  when the drop is a no-op. */
function beforeIdForDrop(
  books: BookCard[],
  dragId: string,
  over: { id: string; pos: 'before' | 'after' },
): string | undefined {
  const overIdx = books.findIndex((b) => b.id === over.id);
  if (overIdx < 0) return dragId;
  let target = books[over.pos === 'before' ? overIdx : overIdx + 1];
  if (target?.id === dragId) target = books[overIdx + 2]; // skip the dragged row itself
  return target?.id === dragId ? dragId : target?.id;
}
