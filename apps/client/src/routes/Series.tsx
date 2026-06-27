import { getRouteApi, useNavigate } from '@tanstack/react-router';
import { Button, Badge, ProgressBar, EmptyState, IconButton, Tooltip } from '@comichub/ui';
import type { BookCard, SeriesDetail } from '@comichub/api-client';
import { useClient } from '../lib/client.js';
import { useSeriesDetail, useMarkBook } from '../lib/queries.js';
import { useReadLaunch } from '../lib/launch.js';
import { LoadingState, ErrorState } from '../components/Page.js';
import { issueLabel, resumePage } from '../lib/format.js';

const route = getRouteApi('/series/$id');

/** Series detail: a cover hero with a resume CTA over the full run of issues. */
export function Series() {
  const { id } = route.useParams();
  const series = useSeriesDetail(id);

  if (series.isLoading)
    return (
      <Centered>
        <LoadingState />
      </Centered>
    );
  if (series.isError) {
    return (
      <Centered>
        <ErrorState
          message={
            series.error instanceof Error ? series.error.message : 'Could not load this series.'
          }
          onRetry={() => series.refetch()}
        />
      </Centered>
    );
  }
  if (!series.data) return null;
  return <SeriesView detail={series.data} />;
}

function SeriesView({ detail }: { detail: SeriesDetail }) {
  const client = useClient();
  const navigate = useNavigate();
  const launch = useReadLaunch();

  const resumeBook =
    detail.books.find((b) => b.progress?.status === 'in_progress') ??
    detail.books.find((b) => b.progress?.status !== 'read') ??
    detail.books[0];
  const isResuming = resumeBook?.progress?.status === 'in_progress';
  const coverBook = resumeBook ?? detail.books[0];

  return (
    <div style={{ padding: 'var(--pad-screen, 32px)', maxWidth: 1320, margin: '0 auto' }}>
      <section
        style={{
          display: 'grid',
          gridTemplateColumns: 'minmax(200px, 232px) 1fr',
          gap: 32,
          marginBottom: 40,
          alignItems: 'start',
        }}
      >
        <div
          style={{
            aspectRatio: 'var(--cover-aspect)',
            background: 'var(--surface-cover)',
            backgroundImage: coverBook ? `url(${client.coverUrl(coverBook.id, 400)})` : undefined,
            backgroundSize: 'cover',
            backgroundPosition: 'center',
            borderRadius: 'var(--radius-none)',
          }}
        />
        <div style={{ minWidth: 0 }}>
          <h1
            style={{
              margin: '0 0 10px',
              fontFamily: 'var(--font-display)',
              fontSize: 'var(--text-display-l)',
              lineHeight: 'var(--leading-display-l)',
              fontWeight: 800,
              letterSpacing: 'var(--tracking-tight)',
              color: 'var(--text-primary)',
            }}
          >
            {detail.name}
          </h1>
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 10,
              flexWrap: 'wrap',
              marginBottom: 18,
            }}
          >
            {detail.publisher && <Badge>{detail.publisher}</Badge>}
            {detail.year && <Badge mono>{detail.year}</Badge>}
            <Badge mono>
              {detail.bookCount} {detail.bookCount === 1 ? 'issue' : 'issues'}
            </Badge>
            {detail.readCount > 0 && (
              <Badge tone="accent" mono>
                {detail.readCount} read
              </Badge>
            )}
          </div>

          {detail.bookCount > 0 && (
            <div style={{ maxWidth: 360, marginBottom: 20 }}>
              <ProgressBar
                value={detail.readCount}
                max={detail.bookCount}
                tone="accent"
                showCount
              />
            </div>
          )}

          {detail.summary && (
            <p
              style={{
                margin: '0 0 22px',
                maxWidth: 620,
                color: 'var(--text-secondary)',
                lineHeight: 1.55,
              }}
            >
              {detail.summary}
            </p>
          )}

          {resumeBook && (
            <div style={{ display: 'flex', gap: 10 }}>
              <Button
                variant="primary"
                icon="book-open"
                onClick={() => launch(resumeBook.id, resumePage(resumeBook.progress))}
              >
                {isResuming
                  ? `Continue ${issueLabel(resumeBook.number) ?? ''}`
                  : 'Read first issue'}
              </Button>
              <Button
                variant="secondary"
                onClick={() => navigate({ to: '/book/$id', params: { id: resumeBook.id } })}
              >
                Issue details
              </Button>
            </div>
          )}
        </div>
      </section>

      <h2
        className="ch-mono"
        style={{
          margin: '0 0 16px',
          fontSize: 'var(--text-label)',
          textTransform: 'uppercase',
          letterSpacing: 'var(--tracking-label)',
          color: 'var(--text-secondary)',
        }}
      >
        Issues
      </h2>

      {detail.books.length === 0 ? (
        <EmptyState title="No issues yet">This series has no scanned issues.</EmptyState>
      ) : (
        <ul
          style={{
            listStyle: 'none',
            margin: 0,
            padding: 0,
            display: 'flex',
            flexDirection: 'column',
          }}
        >
          {detail.books.map((book) => (
            <IssueRow key={book.id} book={book} seriesName={detail.name} />
          ))}
        </ul>
      )}
    </div>
  );
}

function IssueRow({ book, seriesName }: { book: BookCard; seriesName: string }) {
  const client = useClient();
  const navigate = useNavigate();
  const launch = useReadLaunch();
  const mark = useMarkBook();

  const isRead = book.progress?.status === 'read';
  const inProgress = book.progress?.status === 'in_progress';

  return (
    <li
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 14,
        padding: '10px 8px',
        borderBottom: '1px solid var(--border-hairline)',
      }}
    >
      <button
        type="button"
        onClick={() => navigate({ to: '/book/$id', params: { id: book.id } })}
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 14,
          flex: 1,
          minWidth: 0,
          background: 'none',
          border: 'none',
          padding: 0,
          cursor: 'pointer',
          textAlign: 'left',
        }}
      >
        <div
          style={{
            width: 44,
            height: 66,
            flex: 'none',
            background: 'var(--surface-cover)',
            backgroundImage: `url(${client.coverUrl(book.id, 120)})`,
            backgroundSize: 'cover',
            backgroundPosition: 'center',
          }}
        />
        <div style={{ minWidth: 0 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <span
              className="ch-mono"
              style={{ color: 'var(--text-tertiary)', fontSize: '0.78rem' }}
            >
              {issueLabel(book.number) ?? '—'}
            </span>
            <span
              style={{
                color: 'var(--text-primary)',
                fontWeight: 600,
                whiteSpace: 'nowrap',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
              }}
            >
              {book.title || seriesName}
            </span>
          </div>
          <div
            className="ch-mono"
            style={{ fontSize: '0.72rem', color: 'var(--text-tertiary)', marginTop: 2 }}
          >
            {book.pageCount} pages
            {inProgress && book.progress ? ` · ${Math.round(book.progress.percent)}% read` : ''}
            {isRead ? ' · read' : ''}
          </div>
        </div>
      </button>

      {inProgress && book.progress && (
        <div style={{ width: 90 }}>
          <ProgressBar value={book.progress.percent} max={100} height={4} />
        </div>
      )}

      <Tooltip label={isRead ? 'Mark unread' : 'Mark read'}>
        <IconButton
          icon={isRead ? 'check' : 'bookmark'}
          variant={isRead ? 'accent' : 'ghost'}
          label={isRead ? 'Mark unread' : 'Mark read'}
          disabled={mark.isPending}
          onClick={() => mark.mutate({ bookId: book.id, status: isRead ? 'unread' : 'read' })}
        />
      </Tooltip>

      <Tooltip label={inProgress ? 'Continue reading' : 'Read'}>
        <IconButton
          icon="book-open"
          variant="solid"
          label={inProgress ? 'Continue reading' : 'Read'}
          onClick={() => launch(book.id, resumePage(book.progress))}
        />
      </Tooltip>
    </li>
  );
}

function Centered({ children }: { children: React.ReactNode }) {
  return <div style={{ padding: 'var(--pad-screen, 32px)' }}>{children}</div>;
}
