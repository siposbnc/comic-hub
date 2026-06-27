import { useState } from 'react';
import { useNavigate } from '@tanstack/react-router';
import { Button, Rail, EmptyState } from '@comichub/ui';
import { useContinueReading, useDiscover, useLibraries, useSeriesNames } from '../lib/queries.js';
import { Page, LoadingState, ErrorState } from '../components/Page.js';
import { BookCover } from '../components/cards.js';
import { AddLibraryDialog } from '../components/AddLibraryDialog.js';

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
  const nothingToShow = cr.length === 0 && recent.length === 0;

  return (
    <Page
      title="Home"
      actions={
        <Button variant="secondary" icon="plus" onClick={() => setAdding(true)}>
          Add library
        </Button>
      }
    >
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
        <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--gap-section, 48px)' }}>
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
                      label: 'Browse all',
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
        </div>
      )}
      {adding && <AddLibraryDialog onClose={() => setAdding(false)} />}
    </Page>
  );
}
