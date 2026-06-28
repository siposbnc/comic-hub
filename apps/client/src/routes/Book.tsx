import { Fragment, useState } from 'react';
import { getRouteApi, useNavigate } from '@tanstack/react-router';
import { Button, Badge, ProgressBar } from '@comichub/ui';
import type { BookDetail } from '@comichub/api-client';
import { useClient } from '../lib/client.js';
import {
  useBookDetail,
  useMarkBook,
  useCollections,
  useReadingLists,
  useAddToCollection,
  useAddToReadingList,
} from '../lib/queries.js';
import { useReadLaunch } from '../lib/launch.js';
import { useUiStore } from '../store/ui.js';
import { LoadingState, ErrorState } from '../components/Page.js';
import { AddToListDialog } from '../components/lists.js';
import { TagChips, TagEditor } from '../components/tags.js';
import { issueLabel, resumePage } from '../lib/format.js';

const route = getRouteApi('/book/$id');

/** Book detail: cover, metadata, a page-thumbnail strip, and the one-click Read CTA. */
export function Book() {
  const { id } = route.useParams();
  const book = useBookDetail(id);

  if (book.isLoading)
    return (
      <div style={{ padding: 'var(--pad-screen, 32px)' }}>
        <LoadingState />
      </div>
    );
  if (book.isError) {
    return (
      <div style={{ padding: 'var(--pad-screen, 32px)' }}>
        <ErrorState
          message={book.error instanceof Error ? book.error.message : 'Could not load this issue.'}
          onRetry={() => book.refetch()}
        />
      </div>
    );
  }
  if (!book.data) return null;
  return <BookView detail={book.data} />;
}

function BookView({ detail }: { detail: BookDetail }) {
  const client = useClient();
  const navigate = useNavigate();
  const launch = useReadLaunch();
  const mark = useMarkBook();

  const isRead = detail.progress?.status === 'read';
  const inProgress = detail.progress?.status === 'in_progress';
  const startPage = resumePage(detail.progress);

  return (
    <div style={{ padding: 'var(--pad-screen, 32px)', maxWidth: 1320, margin: '0 auto' }}>
      <button
        type="button"
        onClick={() => navigate({ to: '/series/$id', params: { id: detail.seriesId } })}
        style={{
          display: 'inline-flex',
          alignItems: 'center',
          gap: 6,
          background: 'none',
          border: 'none',
          padding: 0,
          marginBottom: 22,
          cursor: 'pointer',
          color: 'var(--text-secondary)',
          fontSize: 'var(--text-small)',
        }}
      >
        ← {detail.seriesName}
      </button>

      <section
        style={{
          display: 'grid',
          gridTemplateColumns: 'minmax(200px, 260px) 1fr',
          gap: 32,
          marginBottom: 40,
          alignItems: 'start',
        }}
      >
        <div
          style={{
            aspectRatio: 'var(--cover-aspect)',
            background: 'var(--surface-cover)',
            backgroundImage: `url(${client.coverUrl(detail.id, 500)})`,
            backgroundSize: 'cover',
            backgroundPosition: 'center',
          }}
        />
        <div style={{ minWidth: 0 }}>
          <div
            className="ch-mono"
            style={{
              fontSize: 'var(--text-label)',
              textTransform: 'uppercase',
              letterSpacing: 'var(--tracking-label)',
              color: 'var(--text-tertiary)',
              marginBottom: 8,
            }}
          >
            {detail.seriesName} {issueLabel(detail.number) ?? ''}
          </div>
          <h1
            style={{
              margin: '0 0 14px',
              fontFamily: 'var(--font-display)',
              fontSize: 'var(--text-display-l)',
              lineHeight: 'var(--leading-display-l)',
              fontWeight: 800,
              letterSpacing: 'var(--tracking-tight)',
              color: 'var(--text-primary)',
            }}
          >
            {detail.title || `${detail.seriesName} ${issueLabel(detail.number) ?? ''}`.trim()}
          </h1>

          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 10,
              flexWrap: 'wrap',
              marginBottom: 20,
            }}
          >
            <Badge mono>{detail.pageCount} pages</Badge>
            <Badge mono>{detail.format.toUpperCase()}</Badge>
            <Badge mono>{detail.readingDir === 'rtl' ? 'RTL' : 'LTR'}</Badge>
            {detail.language && <Badge mono>{detail.language.toUpperCase()}</Badge>}
            {detail.ageRating && <Badge>{detail.ageRating}</Badge>}
            {isRead && (
              <Badge tone="accent" mono dot>
                read
              </Badge>
            )}
            {detail.isCorrupt && (
              <Badge tone="danger" mono>
                corrupt
              </Badge>
            )}
          </div>

          {inProgress && detail.progress && (
            <div style={{ maxWidth: 320, marginBottom: 20 }}>
              <ProgressBar
                value={detail.progress.percent}
                max={100}
                label={`Page ${detail.progress.page + 1} of ${detail.pageCount}`}
              />
            </div>
          )}

          {detail.summary && (
            <p
              style={{
                margin: '0 0 24px',
                maxWidth: 620,
                color: 'var(--text-secondary)',
                lineHeight: 1.55,
              }}
            >
              {detail.summary}
            </p>
          )}

          <Facts detail={detail} />

          <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap' }}>
            <Button
              variant="primary"
              icon="book-open"
              disabled={detail.isCorrupt}
              onClick={() => launch(detail.id, startPage)}
            >
              {inProgress ? 'Continue reading' : 'Read'}
            </Button>
            <Button
              variant="secondary"
              icon={isRead ? 'refresh' : 'check'}
              disabled={mark.isPending}
              onClick={() => mark.mutate({ bookId: detail.id, status: isRead ? 'unread' : 'read' })}
            >
              {isRead ? 'Mark unread' : 'Mark read'}
            </Button>
            {inProgress && (
              <Button
                variant="ghost"
                icon="refresh"
                disabled={mark.isPending}
                onClick={() => mark.mutate({ bookId: detail.id, status: 'unread' })}
              >
                Reset progress
              </Button>
            )}
            <AddToActions
              bookId={detail.id}
              inCollections={detail.collectionIds ?? []}
              inReadingLists={detail.readingListIds ?? []}
            />
          </div>

          <BookTags detail={detail} />
        </div>
      </section>

      <h2
        className="ch-mono"
        style={{
          margin: '0 0 14px',
          fontSize: 'var(--text-label)',
          textTransform: 'uppercase',
          letterSpacing: 'var(--tracking-label)',
          color: 'var(--text-secondary)',
        }}
      >
        Pages
      </h2>
      <PageStrip detail={detail} onOpen={(idx) => launch(detail.id, idx)} />
    </div>
  );
}

/** "Add to…" actions: drop the issue into a collection or a personal reading list.
 *  Lists the book already belongs to are filtered out so it can't be added twice. */
function AddToActions({
  bookId,
  inCollections,
  inReadingLists,
}: {
  bookId: string;
  inCollections: string[];
  inReadingLists: string[];
}) {
  const collections = useCollections();
  const lists = useReadingLists();
  const addCollection = useAddToCollection();
  const addList = useAddToReadingList();
  const addToast = useUiStore((s) => s.addToast);
  const [open, setOpen] = useState<null | 'collection' | 'list'>(null);

  const collectionSet = new Set(inCollections);
  const readingListSet = new Set(inReadingLists);

  const add = async (kind: 'collection' | 'list', id: string, label: string) => {
    try {
      if (kind === 'collection') await addCollection.mutateAsync({ id, bookIds: [bookId] });
      else await addList.mutateAsync({ id, bookIds: [bookId] });
      addToast({ tone: 'success', title: `Added to ${label}` });
    } catch (err) {
      addToast({
        tone: 'danger',
        title: 'Could not add',
        message: err instanceof Error ? err.message : 'Unknown error.',
      });
    }
    setOpen(null);
  };

  return (
    <>
      <Button variant="ghost" icon="collection" onClick={() => setOpen('collection')}>
        Add to collection
      </Button>
      <Button variant="ghost" icon="bookmark" onClick={() => setOpen('list')}>
        Add to list
      </Button>
      {open === 'collection' && (
        <AddToListDialog
          title="Add to collection"
          options={(collections.data ?? [])
            .filter((c) => !collectionSet.has(c.id))
            .map((c) => ({ id: c.id, name: c.name }))}
          onPick={(id) => add('collection', id, 'collection')}
          onClose={() => setOpen(null)}
          busy={addCollection.isPending}
          emptyHint={
            (collections.data?.length ?? 0) === 0
              ? 'No collections yet — create one from the Collections screen.'
              : 'Already in every collection.'
          }
        />
      )}
      {open === 'list' && (
        <AddToListDialog
          title="Add to reading list"
          options={(lists.data ?? [])
            .filter((l) => !readingListSet.has(l.id))
            .map((l) => ({ id: l.id, name: l.name }))}
          onPick={(id) => add('list', id, 'reading list')}
          onClose={() => setOpen(null)}
          busy={addList.isPending}
          emptyHint={
            (lists.data?.length ?? 0) === 0
              ? 'No reading lists yet — create one from the Reading Lists screen.'
              : 'Already in every reading list.'
          }
        />
      )}
    </>
  );
}

/** The book's tags with an inline editor to assign/unassign/create. */
function BookTags({ detail }: { detail: BookDetail }) {
  const [editing, setEditing] = useState(false);
  const tags = detail.tags ?? [];
  return (
    <div
      style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap', marginTop: 18 }}
    >
      <TagChips tags={tags} />
      <Button variant="ghost" icon="edit" onClick={() => setEditing(true)}>
        {tags.length ? 'Edit tags' : 'Add tags'}
      </Button>
      {editing && (
        <TagEditor
          bookId={detail.id}
          assignedIds={new Set(tags.map((t) => t.id))}
          onClose={() => setEditing(false)}
        />
      )}
    </div>
  );
}

/** Online-match metadata: release date, credits by role, genres, characters. */
function Facts({ detail }: { detail: BookDetail }) {
  const rows: [string, string][] = [];
  if (detail.releaseDate) rows.push(['Released', formatRelease(detail.releaseDate)]);
  for (const [role, names] of Object.entries(detail.credits ?? {})) {
    if (names.length) rows.push([capitalize(role), names.join(', ')]);
  }
  if (detail.genres?.length) rows.push(['Genres', detail.genres.join(', ')]);
  if (detail.characters?.length) rows.push(['Characters', detail.characters.join(', ')]);
  if (rows.length === 0) return null;

  return (
    <dl
      style={{
        display: 'grid',
        gridTemplateColumns: '120px 1fr',
        gap: '8px 18px',
        margin: '0 0 24px',
        maxWidth: 580,
        fontSize: 'var(--text-small)',
      }}
    >
      {rows.map(([label, value]) => (
        <Fragment key={label}>
          <dt className="ch-label" style={{ color: 'var(--text-tertiary)', paddingTop: 1 }}>
            {label}
          </dt>
          <dd style={{ margin: 0, color: 'var(--text-secondary)' }}>{value}</dd>
        </Fragment>
      ))}
    </dl>
  );
}

function formatRelease(ms: number): string {
  return new Date(ms).toLocaleDateString(undefined, { year: 'numeric', month: 'short' });
}

function capitalize(s: string): string {
  return s.charAt(0).toUpperCase() + s.slice(1);
}

/** A lazy-loaded thumbnail rail; clicking a page opens the reader at that page. */
function PageStrip({ detail, onOpen }: { detail: BookDetail; onOpen: (idx: number) => void }) {
  const client = useClient();
  const pages = Array.from({ length: detail.pageCount }, (_, i) => i);

  return (
    <div
      style={{
        display: 'flex',
        gap: 10,
        overflowX: 'auto',
        paddingBottom: 8,
        scrollbarWidth: 'thin',
      }}
    >
      {pages.map((idx) => (
        <button
          key={idx}
          type="button"
          onClick={() => onOpen(idx)}
          title={`Read from page ${idx + 1}`}
          style={{
            flex: 'none',
            padding: 0,
            border: '1px solid var(--border-hairline)',
            borderRadius: 'var(--radius-sm)',
            background: 'var(--surface-cover)',
            cursor: 'pointer',
            overflow: 'hidden',
            display: 'flex',
            flexDirection: 'column',
          }}
        >
          <img
            src={client.pageThumbUrl(detail.id, idx)}
            alt={`Page ${idx + 1}`}
            loading="lazy"
            width={96}
            height={144}
            style={{ display: 'block', width: 96, height: 144, objectFit: 'cover' }}
          />
          <span
            className="ch-mono"
            style={{
              fontSize: '0.66rem',
              color: 'var(--text-tertiary)',
              textAlign: 'center',
              padding: '3px 0',
            }}
          >
            {idx + 1}
          </span>
        </button>
      ))}
    </div>
  );
}
