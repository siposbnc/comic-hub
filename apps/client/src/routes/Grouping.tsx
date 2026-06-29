import { getRouteApi, useNavigate } from '@tanstack/react-router';
import { Button, CoverCard, IconButton, ProgressBar } from '@comichub/ui';
import type { BookCard, GroupingDetail } from '@comichub/api-client';
import { useClient } from '../lib/client.js';
import { useStoryArc, useVolume } from '../lib/queries.js';
import { useReadLaunch } from '../lib/launch.js';
import { useMarkBook } from '../lib/queries.js';
import { LoadingState, ErrorState } from '../components/Page.js';
import { issueLabel, resumePage, toCoverStatus, progressFraction } from '../lib/format.js';

const arcRoute = getRouteApi('/series/$id/story-arcs/$arcId');
const volumeRoute = getRouteApi('/series/$id/volumes/$volume');

/** Story-arc detail (route component). */
export function StoryArc() {
  const { id, arcId } = arcRoute.useParams();
  const q = useStoryArc(id, arcId);
  return <GroupingScreen q={q} seriesId={id} />;
}

/** Volume detail (route component). */
export function Volume() {
  const { id, volume } = volumeRoute.useParams();
  const q = useVolume(id, volume);
  return <GroupingScreen q={q} seriesId={id} />;
}

type Query = {
  isLoading: boolean;
  isError: boolean;
  error: unknown;
  data?: GroupingDetail;
  refetch: () => void;
};

function GroupingScreen({ q, seriesId }: { q: Query; seriesId: string }) {
  if (q.isLoading)
    return (
      <div style={{ padding: 'var(--pad-screen)' }}>
        <LoadingState />
      </div>
    );
  if (q.isError || !q.data)
    return (
      <div style={{ padding: 'var(--pad-screen)' }}>
        <ErrorState
          message={q.error instanceof Error ? q.error.message : 'Could not load this.'}
          onRetry={() => q.refetch()}
        />
      </div>
    );
  return <GroupingView detail={q.data} seriesId={seriesId} />;
}

function GroupingView({ detail, seriesId }: { detail: GroupingDetail; seriesId: string }) {
  const client = useClient();
  const navigate = useNavigate();
  const launch = useReadLaunch();
  const mark = useMarkBook();

  const books = detail.books;
  const coverUrl = books[0] ? client.coverUrl(books[0].id, 600) : undefined;
  const span =
    books.length > 0
      ? `${issueLabel(books[0]!.number) ?? '#?'} – ${issueLabel(books[books.length - 1]!.number) ?? '#?'}`
      : '';
  const next = books.find((b) => b.progress?.status !== 'read') ?? books[0];
  const isResuming = next?.progress?.status === 'in_progress';
  const kindLabel = detail.kind === 'arc' ? 'Story arc' : 'Volume';
  const metaLine = [detail.year || undefined, `${detail.issueCount} issues`, span]
    .filter(Boolean)
    .join(' · ');

  const markAllRead = () => {
    for (const b of books) {
      if (b.progress?.status !== 'read') mark.mutate({ bookId: b.id, status: 'read' });
    }
  };

  return (
    <div>
      {/* Cover-wash hero */}
      <div
        style={{
          position: 'relative',
          overflow: 'hidden',
          borderBottom: '1px solid var(--border-hairline)',
        }}
      >
        {coverUrl && (
          <div
            aria-hidden
            style={{
              position: 'absolute',
              inset: 0,
              backgroundImage: `url(${coverUrl})`,
              backgroundSize: 'cover',
              backgroundPosition: 'center',
              filter: 'blur(44px) saturate(1.3)',
              transform: 'scale(1.2)',
              opacity: 0.5,
            }}
          />
        )}
        <div
          aria-hidden
          style={{
            position: 'absolute',
            inset: 0,
            background:
              'linear-gradient(180deg, color-mix(in oklab, var(--ink-900) 72%, transparent), var(--ink-900) 90%)',
          }}
        />
        <div
          style={{
            position: 'relative',
            display: 'flex',
            gap: 28,
            padding: 'var(--pad-screen)',
            maxWidth: 'var(--content-max)',
            margin: '0 auto',
          }}
        >
          <div className="ch-reg" style={{ flex: 'none' }}>
            <div
              style={{
                width: 168,
                aspectRatio: 'var(--cover-aspect)',
                background: 'var(--surface-cover)',
                backgroundImage: coverUrl ? `url(${coverUrl})` : undefined,
                backgroundSize: 'cover',
                backgroundPosition: 'center',
                boxShadow: '0 12px 40px rgba(0,0,0,.55)',
              }}
            />
          </div>
          <div style={{ flex: 1, minWidth: 0, paddingTop: 6 }}>
            <button
              type="button"
              className="ch-mono"
              onClick={() =>
                navigate({
                  to: '/series/$id',
                  params: { id: seriesId },
                })
              }
              style={{
                background: 'none',
                border: 'none',
                padding: 0,
                cursor: 'pointer',
                fontSize: '0.66rem',
                fontWeight: 600,
                letterSpacing: '0.16em',
                textTransform: 'uppercase',
                color: 'var(--accent)',
              }}
            >
              {kindLabel} · {detail.seriesName}
            </button>
            <h1
              style={{
                margin: '10px 0 0',
                fontFamily: 'var(--font-display)',
                fontWeight: 800,
                fontSize: 'var(--text-display-l)',
                letterSpacing: '-0.015em',
                lineHeight: 1.04,
                color: 'var(--text-primary)',
              }}
            >
              {detail.name}
            </h1>
            <p
              className="ch-mono"
              style={{ margin: '12px 0 0', fontSize: '0.74rem', color: 'var(--text-secondary)' }}
            >
              {metaLine}
            </p>
            {detail.description && (
              <p
                style={{
                  margin: '14px 0 0',
                  maxWidth: 560,
                  color: 'var(--text-secondary)',
                  fontSize: 'var(--text-body)',
                  lineHeight: 1.55,
                }}
              >
                {detail.description}
              </p>
            )}
            <div style={{ maxWidth: 320, marginTop: 16 }}>
              <ProgressBar
                value={detail.readCount}
                max={detail.issueCount}
                label="Read"
                showCount
              />
            </div>
            <div style={{ display: 'flex', gap: 10, marginTop: 18 }}>
              {next && (
                <Button icon="book-open" onClick={() => launch(next.id, resumePage(next.progress))}>
                  {isResuming
                    ? `Continue · ${issueLabel(next.number) ?? ''}`
                    : `Read ${issueLabel(books[0]?.number) ?? ''}`}
                </Button>
              )}
              <Button
                variant="secondary"
                icon="check"
                disabled={mark.isPending || detail.readCount === detail.issueCount}
                onClick={markAllRead}
              >
                Mark {detail.kind === 'arc' ? 'arc' : 'volume'} read
              </Button>
            </div>
          </div>
        </div>
      </div>

      {/* Issues */}
      <div
        style={{ padding: 'var(--pad-screen)', maxWidth: 'var(--content-max)', margin: '0 auto' }}
      >
        <div
          className="ch-mono"
          style={{
            fontSize: '0.62rem',
            fontWeight: 600,
            letterSpacing: '0.16em',
            textTransform: 'uppercase',
            color: 'var(--text-tertiary)',
            marginBottom: 18,
          }}
        >
          {detail.kind === 'arc' ? 'In reading order' : 'Collected issues'}
        </div>
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fill, minmax(132px, 1fr))',
            gap: 16,
          }}
        >
          {books.map((b) => (
            <IssueCover key={b.id} book={b} seriesName={detail.seriesName} />
          ))}
        </div>
      </div>
    </div>
  );
}

/** One issue cover that opens the reader at its resume page (details via hover button). */
function IssueCover({ book, seriesName }: { book: BookCard; seriesName: string }) {
  const client = useClient();
  const navigate = useNavigate();
  const launch = useReadLaunch();
  const number = issueLabel(book.number);
  return (
    <div style={{ position: 'relative' }}>
      <CoverCard
        cover={client.coverUrl(book.id, 300)}
        title={number ?? book.title ?? seriesName}
        subtitle={`${book.pageCount} pp`}
        number={number}
        status={toCoverStatus(book.progress?.status)}
        progress={progressFraction(book.progress)}
        size="s"
        style={{ width: '100%' }}
        onClick={() => launch(book.id, resumePage(book.progress))}
      />
      <div style={{ position: 'absolute', top: 6, right: 6, zIndex: 1 }}>
        <IconButton
          icon="info"
          label="Issue details"
          variant="solid"
          size="sm"
          onClick={(e: React.MouseEvent) => {
            e.stopPropagation();
            navigate({ to: '/book/$id', params: { id: book.id } });
          }}
        />
      </div>
    </div>
  );
}
