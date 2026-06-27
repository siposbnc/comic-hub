import { getRouteApi, useNavigate } from '@tanstack/react-router';
import { Button, EmptyState } from '@comichub/ui';
import { useClient } from '../lib/client.js';
import { useLibrary, useSeriesList } from '../lib/queries.js';
import { useUiStore } from '../store/ui.js';
import { LoadingState, ErrorState } from '../components/Page.js';
import { CoverGrid } from '../components/CoverGrid.js';
import { SeriesCover, COVER_CELL_HEIGHT } from '../components/cards.js';

const route = getRouteApi('/library/$id');
const COVER_PX = 168;

/** A single library: a virtualized wall of series covers with scan + remove controls. */
export function Library() {
  const { id } = route.useParams();
  const client = useClient();
  const navigate = useNavigate();
  const search = useUiStore((s) => s.search)
    .trim()
    .toLowerCase();
  const addToast = useUiStore((s) => s.addToast);

  const library = useLibrary(id);
  const series = useSeriesList(id);

  const items = (series.data ?? []).filter((s) =>
    search ? s.name.toLowerCase().includes(search) : true,
  );

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

  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <div
        style={{
          flex: 'none',
          padding: 'var(--pad-screen, 32px) var(--pad-screen, 32px) 0',
          display: 'flex',
          alignItems: 'flex-end',
          justifyContent: 'space-between',
          gap: 16,
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
            {series.data ? `${series.data.length} series` : 'Library'}
          </div>
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
          <div style={{ padding: 'var(--pad-screen, 32px)' }}>
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
            cardWidth={COVER_PX}
            rowHeight={COVER_CELL_HEIGHT}
            getKey={(s) => s.id}
            renderItem={(s) => <SeriesCover series={s} />}
          />
        )}
      </div>
    </div>
  );
}
