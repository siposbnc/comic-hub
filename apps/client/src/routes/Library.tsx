import { useMemo, useState } from 'react';
import { getRouteApi, useNavigate } from '@tanstack/react-router';
import { Button, EmptyState, Select } from '@comichub/ui';
import type { SeriesCard } from '@comichub/api-client';
import { useClient } from '../lib/client.js';
import { useLibrary, useSeriesList } from '../lib/queries.js';
import { useUiStore } from '../store/ui.js';
import { LoadingState, ErrorState } from '../components/Page.js';
import { CoverGrid } from '../components/CoverGrid.js';
import { SeriesCover } from '../components/cards.js';

const route = getRouteApi('/library/$id');

type Sort = 'added' | 'name' | 'year' | 'unread';

/** A single library: filter bar + a virtualized wall of series covers with scan/remove. */
export function Library() {
  const { id } = route.useParams();
  const client = useClient();
  const navigate = useNavigate();
  const search = useUiStore((s) => s.search)
    .trim()
    .toLowerCase();
  const density = useUiStore((s) => s.density);
  const addToast = useUiStore((s) => s.addToast);

  const [sort, setSort] = useState<Sort>('added');

  const library = useLibrary(id);
  const series = useSeriesList(id);

  const items = useMemo(() => {
    const list = (series.data ?? []).filter((s) =>
      search ? s.name.toLowerCase().includes(search) : true,
    );
    return sortSeries(list, sort);
  }, [series.data, search, sort]);

  const handleScan = async () => {
    try {
      await client.scanLibrary(id, 'incremental');
      addToast({ tone: 'info', title: 'Scanning library', message: 'Checking for new issues.' });
    } catch (err) {
      addToast({
        tone: 'danger',
        title: 'Could not start scan',
        message: err instanceof Error ? err.message : 'Unknown error.',
      });
    }
  };

  const handleRemove = async () => {
    if (
      !window.confirm(
        `Remove "${library.data?.name ?? 'this library'}"? Your files are not deleted.`,
      )
    ) {
      return;
    }
    try {
      await client.deleteLibrary(id);
      addToast({ tone: 'success', title: 'Library removed' });
      navigate({ to: '/' });
    } catch (err) {
      addToast({
        tone: 'danger',
        title: 'Could not remove library',
        message: err instanceof Error ? err.message : 'Unknown error.',
      });
    }
  };

  const colW = density === 's' ? 132 : 168;
  const rowHeight = colW * 1.5 + 38;

  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <div
        style={{
          flex: 'none',
          padding: 'var(--pad-screen) var(--pad-screen) 0',
          maxWidth: 'var(--content-max)',
          margin: '0 auto',
          width: '100%',
        }}
      >
        <div
          style={{
            display: 'flex',
            alignItems: 'baseline',
            justifyContent: 'space-between',
            gap: 16,
            marginBottom: 18,
          }}
        >
          <div style={{ display: 'flex', alignItems: 'baseline', gap: 14, minWidth: 0 }}>
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
              {library.data?.name ?? 'Library'}
            </h1>
          </div>
          <div style={{ flex: 'none', display: 'flex', gap: 10 }}>
            <Button variant="ghost" icon="trash" onClick={handleRemove}>
              Remove
            </Button>
            <Button variant="secondary" icon="refresh" onClick={handleScan}>
              Scan
            </Button>
          </div>
        </div>

        {/* Filter bar */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 12,
            flexWrap: 'wrap',
            paddingBottom: 16,
            borderBottom: '1px solid var(--border-hairline)',
          }}
        >
          <span className="ch-mono" style={{ fontSize: '0.78rem', color: 'var(--text-secondary)' }}>
            {items.length} series
          </span>
          <div style={{ flex: 1 }} />
          <Select
            value={sort}
            onChange={(e: React.ChangeEvent<HTMLSelectElement>) => setSort(e.target.value as Sort)}
            size="sm"
          >
            <option value="added">Recently added</option>
            <option value="name">Name</option>
            <option value="year">Year</option>
            <option value="unread">Unread count</option>
          </Select>
        </div>
      </div>

      <div style={{ flex: 1, minHeight: 0, marginTop: 8 }}>
        {series.isLoading ? (
          <LoadingState />
        ) : series.isError ? (
          <ErrorState
            message={
              series.error instanceof Error ? series.error.message : 'Could not load series.'
            }
            onRetry={() => series.refetch()}
          />
        ) : items.length === 0 ? (
          <div style={{ padding: 'var(--pad-screen)' }}>
            {search ? (
              <EmptyState title="No matches">
                Nothing in this library matches “{search}”. Try a different search.
              </EmptyState>
            ) : (
              <EmptyState
                title="This library is empty"
                action={
                  <Button variant="primary" icon="refresh" onClick={handleScan}>
                    Scan now
                  </Button>
                }
              >
                Run a scan to index the comics in this folder.
              </EmptyState>
            )}
          </div>
        ) : (
          <CoverGrid
            items={items}
            cardWidth={colW}
            rowHeight={rowHeight}
            gap={18}
            getKey={(s) => s.id}
            renderItem={(s) => <SeriesCover series={s} size={density} />}
          />
        )}
      </div>
    </div>
  );
}

function sortSeries(list: SeriesCard[], sort: Sort): SeriesCard[] {
  if (sort === 'added') return list;
  const sorted = [...list];
  sorted.sort((a, b) => {
    if (sort === 'name') return a.name.localeCompare(b.name);
    if (sort === 'year') return (b.year ?? 0) - (a.year ?? 0);
    if (sort === 'unread') return b.bookCount - b.readCount - (a.bookCount - a.readCount);
    return 0;
  });
  return sorted;
}
