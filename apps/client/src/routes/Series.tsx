import { useState } from 'react';
import { getRouteApi, useNavigate } from '@tanstack/react-router';
import { Button, ProgressBar, Tabs, CoverCard, EmptyState } from '@comichub/ui';
import type { BookCard, SeriesDetail } from '@comichub/api-client';
import { useClient } from '../lib/client.js';
import { useSeriesDetail } from '../lib/queries.js';
import { useReadLaunch } from '../lib/launch.js';
import { LoadingState, ErrorState } from '../components/Page.js';
import { MatchDialog } from '../components/MatchDialog.js';
import { issueLabel, resumePage, toCoverStatus, progressFraction } from '../lib/format.js';

const route = getRouteApi('/series/$id');

/** Series detail: a full-bleed cover-wash hero with a resume CTA over the run of issues. */
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
  const [tab, setTab] = useState('issues');
  const [matching, setMatching] = useState(false);

  const resumeBook =
    detail.books.find((b) => b.progress?.status === 'in_progress') ??
    detail.books.find((b) => b.progress?.status !== 'read') ??
    detail.books[0];
  const isResuming = resumeBook?.progress?.status === 'in_progress';
  const coverBook = resumeBook ?? detail.books[0];
  const coverUrl = coverBook ? client.coverUrl(coverBook.id, 600) : undefined;
  const format = detail.books[0]?.format?.toUpperCase();

  const meta = [detail.publisher, detail.year, `${detail.bookCount} issues`]
    .filter(Boolean)
    .join(' · ');

  return (
    <div>
      {/* Hero */}
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
              filter: 'blur(40px) saturate(1.3)',
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
              'linear-gradient(180deg, color-mix(in oklab, var(--ink-900) 70%, transparent), var(--ink-900) 88%)',
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
                width: 200,
                aspectRatio: 'var(--cover-aspect)',
                background: 'var(--surface-cover)',
                backgroundImage: coverUrl ? `url(${coverUrl})` : undefined,
                backgroundSize: 'cover',
                backgroundPosition: 'center',
              }}
            />
          </div>
          <div style={{ flex: 1, minWidth: 0, paddingTop: 8 }}>
            <h1
              style={{
                margin: 0,
                fontFamily: 'var(--font-display)',
                fontWeight: 800,
                fontSize: 'var(--text-display-xl)',
                lineHeight: 1.02,
                letterSpacing: '-0.015em',
                color: 'var(--text-primary)',
              }}
            >
              {detail.name}
            </h1>
            {meta && (
              <p
                className="ch-label"
                style={{ margin: '12px 0 0', color: 'var(--text-secondary)' }}
              >
                {meta}
              </p>
            )}
            {detail.summary && (
              <p
                style={{
                  margin: '16px 0 0',
                  maxWidth: 560,
                  color: 'var(--text-secondary)',
                  fontSize: 'var(--text-body)',
                  lineHeight: 1.55,
                }}
              >
                {detail.summary}
              </p>
            )}
            {detail.bookCount > 0 && (
              <div style={{ maxWidth: 320, marginTop: 18 }}>
                <ProgressBar
                  value={detail.readCount}
                  max={detail.bookCount}
                  label="Read"
                  showCount
                />
              </div>
            )}
            {resumeBook && (
              <div style={{ display: 'flex', gap: 10, marginTop: 20 }}>
                <Button
                  icon="book-open"
                  onClick={() => launch(resumeBook.id, resumePage(resumeBook.progress))}
                >
                  {isResuming
                    ? `Continue · ${issueLabel(resumeBook.number) ?? ''}`
                    : 'Read first issue'}
                </Button>
                <Button
                  variant="secondary"
                  onClick={() => navigate({ to: '/book/$id', params: { id: resumeBook.id } })}
                >
                  Issue details
                </Button>
                <Button variant="ghost" icon="search" onClick={() => setMatching(true)}>
                  Match
                </Button>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Body */}
      <div
        style={{ padding: 'var(--pad-screen)', maxWidth: 'var(--content-max)', margin: '0 auto' }}
      >
        <Tabs
          value={tab}
          onChange={setTab}
          tabs={[
            { value: 'issues', label: 'Issues', count: detail.bookCount },
            { value: 'details', label: 'Details' },
            { value: 'history', label: 'History' },
          ]}
          style={{ marginBottom: 22 }}
        />

        {tab === 'issues' ? (
          detail.books.length === 0 ? (
            <EmptyState title="No issues yet">This series has no scanned issues.</EmptyState>
          ) : (
            <div
              style={{
                display: 'grid',
                gridTemplateColumns: 'repeat(auto-fill, minmax(132px, 1fr))',
                gap: 16,
              }}
            >
              {detail.books.map((book) => (
                <IssueCover key={book.id} book={book} seriesName={detail.name} />
              ))}
            </div>
          )
        ) : tab === 'details' ? (
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: '160px 1fr',
              gap: '10px 24px',
              maxWidth: 560,
              fontSize: 'var(--text-body)',
            }}
          >
            {(
              [
                ['Publisher', detail.publisher],
                ['Year', detail.year ? String(detail.year) : undefined],
                ['Issues', String(detail.bookCount)],
                ['Format', format],
                [
                  'Reading direction',
                  detail.readingDir === 'rtl' ? 'Right to left' : 'Left to right',
                ],
              ] as const
            )
              .filter(([, v]) => Boolean(v))
              .map(([k, v]) => (
                <Detail key={k} label={k} value={v as string} />
              ))}
          </div>
        ) : (
          <p style={{ color: 'var(--text-tertiary)' }}>No reading history yet.</p>
        )}
      </div>

      {matching && (
        <MatchDialog
          seriesId={detail.id}
          seriesName={detail.name}
          onClose={() => setMatching(false)}
        />
      )}
    </div>
  );
}

/** One issue as a small CoverCard that opens the reader at its resume page. */
function IssueCover({ book, seriesName }: { book: BookCard; seriesName: string }) {
  const client = useClient();
  const launch = useReadLaunch();
  const number = issueLabel(book.number);
  return (
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
  );
}

function Detail({ label, value }: { label: string; value: string }) {
  return (
    <>
      <div className="ch-label" style={{ color: 'var(--text-tertiary)', paddingTop: 2 }}>
        {label}
      </div>
      <div style={{ color: 'var(--text-primary)' }}>{value}</div>
    </>
  );
}

function Centered({ children }: { children: React.ReactNode }) {
  return <div style={{ padding: 'var(--pad-screen)' }}>{children}</div>;
}
