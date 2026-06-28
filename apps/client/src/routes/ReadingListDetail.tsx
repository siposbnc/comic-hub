import { useState } from 'react';
import { getRouteApi, useNavigate } from '@tanstack/react-router';
import { Button, Badge, IconButton, EmptyState } from '@comichub/ui';
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
import { issueLabel, resumePage, progressFraction } from '../lib/format.js';

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

  const move = (index: number, dir: -1 | 1) => {
    const book = books[index];
    if (!book) return;
    if (dir === -1) {
      const before = books[index - 1];
      if (before) reorder.mutate({ id: listId, bookId: book.id, beforeId: before.id });
    } else {
      const after = books[index + 2]; // place before the item two down (i.e. past the next)
      reorder.mutate({ id: listId, bookId: book.id, beforeId: after?.id });
    }
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
          Use “Add issues” to build your queue, then order it with the up/down arrows.
        </EmptyState>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          {books.map((book, i) => (
            <QueueRow
              key={book.id}
              index={i}
              total={books.length}
              book={book}
              seriesName={seriesNames.get(book.seriesId)}
              cover={client.coverUrl(book.id, 80)}
              onOpen={() => launch(book.id, resumePage(book.progress))}
              onDetails={() => navigate({ to: '/book/$id', params: { id: book.id } })}
              onUp={() => move(i, -1)}
              onDown={() => move(i, 1)}
              onRemove={() => removeItem.mutate({ id: listId, bookId: book.id })}
            />
          ))}
        </div>
      )}
    </div>
  );
}

function QueueRow({
  index,
  total,
  book,
  seriesName,
  cover,
  onOpen,
  onDetails,
  onUp,
  onDown,
  onRemove,
}: {
  index: number;
  total: number;
  book: BookCard;
  seriesName?: string;
  cover: string;
  onOpen: () => void;
  onDetails: () => void;
  onUp: () => void;
  onDown: () => void;
  onRemove: () => void;
}) {
  const isRead = book.progress?.status === 'read';
  const pct = Math.round(progressFraction(book.progress) * 100);
  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 12,
        padding: '8px 12px',
        background: 'var(--surface-raised)',
        border: '1px solid var(--border-hairline)',
        borderRadius: 'var(--radius-md)',
        opacity: isRead ? 0.6 : 1,
      }}
    >
      <span
        className="ch-mono"
        style={{ width: 22, textAlign: 'right', color: 'var(--text-tertiary)' }}
      >
        {index + 1}
      </span>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
        <IconButton
          icon="chevron-up"
          label="Move up"
          variant="ghost"
          size="sm"
          disabled={index === 0}
          onClick={onUp}
        />
        <IconButton
          icon="chevron-down"
          label="Move down"
          variant="ghost"
          size="sm"
          disabled={index === total - 1}
          onClick={onDown}
        />
      </div>
      <button
        type="button"
        onClick={onOpen}
        style={{
          flex: 'none',
          width: 30,
          height: 45,
          padding: 0,
          border: 'none',
          borderRadius: 'var(--radius-sm)',
          overflow: 'hidden',
          background: 'var(--surface-cover)',
          cursor: 'pointer',
        }}
      >
        <img src={cover} alt="" width={30} height={45} style={{ objectFit: 'cover' }} />
      </button>
      <button
        type="button"
        onClick={onOpen}
        style={{
          flex: 1,
          minWidth: 0,
          textAlign: 'left',
          background: 'none',
          border: 'none',
          cursor: 'pointer',
          color: 'var(--text-primary)',
        }}
      >
        <span
          style={{
            display: 'block',
            fontSize: 'var(--text-small)',
            whiteSpace: 'nowrap',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
          }}
        >
          {book.title || seriesName || issueLabel(book.number) || 'Untitled'}
        </span>
        <span
          style={{
            display: 'block',
            fontSize: 'var(--text-label)',
            color: 'var(--text-tertiary)',
          }}
        >
          {[seriesName, issueLabel(book.number), isRead ? 'Read' : pct > 0 ? `${pct}%` : null]
            .filter(Boolean)
            .join(' · ')}
        </span>
      </button>
      <IconButton icon="info" label="Issue details" variant="ghost" size="sm" onClick={onDetails} />
      <IconButton icon="x" label="Remove" variant="ghost" size="sm" onClick={onRemove} />
    </div>
  );
}
