import { useState } from 'react';
import { useNavigate } from '@tanstack/react-router';
import { Button, Rail, EmptyState } from '@comichub/ui';
import type { NextUp } from '@comichub/api-client';
import { useContinueReading, useDiscover, useLibraries, useSeriesNames } from '../lib/queries.js';
import { useClient } from '../lib/client.js';
import { useReadLaunch } from '../lib/launch.js';
import { Page, LoadingState, ErrorState } from '../components/Page.js';
import { BookCover } from '../components/cards.js';
import { AddLibraryDialog } from '../components/AddLibraryDialog.js';
import { NowReadingStrip } from '../components/NowReadingStrip.js';
import { issueLabel, resumePage } from '../lib/format.js';

/** The Home feed: pick-up-where-you-left-off and what's new across every library. */
export function Home() {
  const navigate = useNavigate();
  const [adding, setAdding] = useState(false);
  const libraries = useLibraries();
  const discover = useDiscover();
  const continueReading = useContinueReading();
  const seriesNames = useSeriesNames();

  const hasLibraries = (libraries.data?.length ?? 0) > 0;

  if (libraries.isLoading) {
    return (
      <Page title="Home">
        <LoadingState />
      </Page>
    );
  }

  if (!hasLibraries) {
    return (
      <Page title="Home">
        <EmptyState
          title="Start your longbox"
          action={
            <Button variant="primary" icon="plus" onClick={() => setAdding(true)}>
              Add a library
            </Button>
          }
        >
          Point ComicHub at a folder of .cbz or .cbr files and it will organize them into series and
          issues you can read anywhere.
        </EmptyState>
        {adding && <AddLibraryDialog onClose={() => setAdding(false)} />}
      </Page>
    );
  }

  const cr = continueReading.data ?? [];
  const recent = discover.data?.recentlyAdded ?? [];
  const nextUp = discover.data?.nextUp;
  const nothingToShow = cr.length === 0 && recent.length === 0 && !nextUp;
  const libCount = libraries.data?.length ?? 0;

  return (
    <div
      style={{
        padding: 'var(--pad-screen)',
        maxWidth: 'var(--content-max)',
        margin: '0 auto',
        display: 'flex',
        flexDirection: 'column',
        gap: 'var(--gap-section)',
      }}
    >
      <div
        style={{
          display: 'flex',
          alignItems: 'flex-end',
          justifyContent: 'space-between',
          gap: 16,
        }}
      >
        <div style={{ minWidth: 0 }}>
          <h1
            style={{
              margin: 0,
              fontFamily: 'var(--font-display)',
              fontWeight: 800,
              fontSize: 'var(--text-display-l)',
              lineHeight: 'var(--leading-display-l)',
              letterSpacing: 'var(--tracking-tight)',
              color: 'var(--text-primary)',
            }}
          >
            {greeting()}
          </h1>
          <p className="ch-label" style={{ margin: '8px 0 0', color: 'var(--text-tertiary)' }}>
            {weekday()} · {cr.length} in progress · {libCount}{' '}
            {libCount === 1 ? 'library' : 'libraries'}
          </p>
        </div>
        <Button variant="ghost" size="sm" icon="plus" onClick={() => setAdding(true)}>
          Add library
        </Button>
      </div>

      {/* Household presence (auth mode only; collapses when nobody is reading). */}
      <NowReadingStrip />

      {nextUp && <NextUpCard nextUp={nextUp} seriesName={seriesNames.get(nextUp.book.seriesId)} />}

      {discover.isError ? (
        <ErrorState
          message={
            discover.error instanceof Error ? discover.error.message : 'Could not load the feed.'
          }
          onRetry={() => discover.refetch()}
        />
      ) : nothingToShow && !discover.isLoading ? (
        <EmptyState title="Nothing here yet">
          Once a scan finishes, recently added issues and your reading progress show up here.
        </EmptyState>
      ) : (
        <>
          {cr.length > 0 && (
            <Rail label="Continue reading">
              {cr.map((book) => (
                <BookCover key={book.id} book={book} seriesName={seriesNames.get(book.seriesId)} />
              ))}
            </Rail>
          )}
          {recent.length > 0 && (
            <Rail
              label="Recently added"
              action={
                libraries.data && libraries.data.length === 1
                  ? {
                      label: 'See all',
                      onClick: () =>
                        navigate({
                          to: '/library/$id',
                          params: { id: libraries.data![0]!.id },
                        }),
                    }
                  : undefined
              }
            >
              {recent.map((book) => (
                <BookCover key={book.id} book={book} seriesName={seriesNames.get(book.seriesId)} />
              ))}
            </Rail>
          )}
        </>
      )}
      {adding && <AddLibraryDialog onClose={() => setAdding(false)} />}
    </div>
  );
}

/** A prominent "up next" card sourced from the active reading list. */
function NextUpCard({ nextUp, seriesName }: { nextUp: NextUp; seriesName?: string }) {
  const client = useClient();
  const navigate = useNavigate();
  const launch = useReadLaunch();
  const { book } = nextUp;
  const inProgress = book.progress?.status === 'in_progress';
  return (
    <div
      style={{
        display: 'flex',
        gap: 16,
        alignItems: 'center',
        padding: 16,
        background: 'var(--surface-raised)',
        border: '1px solid var(--border-hairline)',
        borderRadius: 'var(--radius-lg)',
      }}
    >
      <button
        type="button"
        onClick={() => navigate({ to: '/book/$id', params: { id: book.id } })}
        style={{
          flex: 'none',
          width: 64,
          height: 96,
          padding: 0,
          border: 'none',
          borderRadius: 'var(--radius-sm)',
          overflow: 'hidden',
          background: 'var(--surface-cover)',
          cursor: 'pointer',
        }}
      >
        <img
          src={client.coverUrl(book.id, 200)}
          alt=""
          width={64}
          height={96}
          style={{ objectFit: 'cover' }}
        />
      </button>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div
          className="ch-mono"
          style={{
            fontSize: 'var(--text-label)',
            textTransform: 'uppercase',
            letterSpacing: 'var(--tracking-label)',
            color: 'var(--accent)',
            marginBottom: 4,
          }}
        >
          Up next · {nextUp.listName}
        </div>
        <div
          style={{
            fontFamily: 'var(--font-display)',
            fontWeight: 700,
            fontSize: 'var(--text-title)',
            color: 'var(--text-primary)',
            whiteSpace: 'nowrap',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
          }}
        >
          {book.title || seriesName || issueLabel(book.number) || 'Untitled'}
        </div>
        <div style={{ fontSize: 'var(--text-small)', color: 'var(--text-tertiary)', marginTop: 2 }}>
          {[seriesName, issueLabel(book.number)].filter(Boolean).join(' · ')}
        </div>
      </div>
      <Button
        variant="primary"
        icon="book-open"
        onClick={() => launch(book.id, resumePage(book.progress))}
      >
        {inProgress ? 'Continue' : 'Read'}
      </Button>
    </div>
  );
}

/** Time-of-day greeting in the design's voice (sentence case, no exclamation). */
function greeting(): string {
  const h = new Date().getHours();
  if (h < 12) return 'Good morning';
  if (h < 18) return 'Good afternoon';
  return 'Good evening';
}

function weekday(): string {
  return new Date().toLocaleDateString(undefined, { weekday: 'long' });
}
