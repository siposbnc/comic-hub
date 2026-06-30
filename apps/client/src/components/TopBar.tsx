import { useEffect, useState } from 'react';
import { useNavigate, useRouter, useRouterState } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { Input, JobIndicator, IconButton, Tooltip, Button, Icon, type JobItem } from '@comichub/ui';
import type { ComicHubClient, SearchResults } from '@comichub/api-client';
import { useClient } from '../lib/client.js';
import { useUiStore } from '../store/ui.js';
import { issueLabel } from '../lib/format.js';
import { AccountChip } from './AccountChip.js';

/** 56px utility bar: back nav, catalog search, view density, job status, theme, identity. */
export function TopBar() {
  const router = useRouter();
  const navigate = useNavigate();
  const client = useClient();
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const search = useUiStore((s) => s.search);
  const setSearch = useUiStore((s) => s.setSearch);
  const [searchOpen, setSearchOpen] = useState(false);
  const [debounced, setDebounced] = useState('');
  const theme = useUiStore((s) => s.theme);
  const toggleTheme = useUiStore((s) => s.toggleTheme);
  const density = useUiStore((s) => s.density);
  const setDensity = useUiStore((s) => s.setDensity);
  const jobs = useUiStore((s) => s.jobs);

  const info = useQuery({ queryKey: ['server', 'info'], queryFn: () => client.serverInfo() });
  const canGoBack = pathname.startsWith('/series/') || pathname.startsWith('/book/');

  // Debounce the catalog search so type-ahead doesn't fire a request per keystroke.
  useEffect(() => {
    const t = setTimeout(() => setDebounced(search.trim()), 200);
    return () => clearTimeout(t);
  }, [search]);

  const searchActive = debounced.length >= 2;
  const results = useQuery({
    queryKey: ['search', debounced],
    queryFn: () => client.search(debounced, { limit: 8 }),
    enabled: searchActive,
  });
  const showResults = searchOpen && searchActive;

  function goTo(to: '/series/$id' | '/book/$id', id: string) {
    navigate({ to, params: { id } });
    setSearch('');
    setSearchOpen(false);
  }

  const active = Object.values(jobs).filter((j) => j.state === 'running' || j.state === 'queued');
  const failed = Object.values(jobs).some((j) => j.state === 'failed');
  const status: 'idle' | 'scanning' | 'error' = active.length
    ? 'scanning'
    : failed
      ? 'error'
      : 'idle';

  const jobItems: JobItem[] = Object.values(jobs).map((j) => ({
    name: jobLabel(j.type),
    progress: j.total > 0 ? j.done / j.total : j.progress || 0,
  }));

  const overall =
    active.length && active.some((j) => j.total > 0)
      ? Math.round(
          (active.reduce((sum, j) => sum + (j.total > 0 ? j.done / j.total : 0), 0) /
            active.length) *
            100,
        )
      : null;

  return (
    <header
      style={{
        height: 'var(--topbar-height, 56px)',
        flex: 'none',
        display: 'flex',
        alignItems: 'center',
        gap: 14,
        padding: '0 var(--space-6)',
        borderBottom: '1px solid var(--border-hairline)',
        background: 'var(--surface-raised)',
      }}
    >
      {canGoBack && (
        <Button variant="ghost" size="sm" icon="chevron-left" onClick={() => router.history.back()}>
          Back
        </Button>
      )}

      <div
        style={{ position: 'relative', flex: 1, maxWidth: 440 }}
        onFocus={() => setSearchOpen(true)}
        onBlur={(e: React.FocusEvent<HTMLDivElement>) => {
          if (!e.currentTarget.contains(e.relatedTarget as Node | null)) setSearchOpen(false);
        }}
      >
        <Input
          icon="search"
          size="sm"
          placeholder="Search title or series…"
          aria-label="Search the catalog"
          value={search}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => setSearch(e.target.value)}
          onKeyDown={(e: React.KeyboardEvent<HTMLInputElement>) => {
            if (e.key === 'Escape') {
              setSearchOpen(false);
              e.currentTarget.blur();
            }
          }}
        />
        {showResults && (
          <SearchResultsPanel
            client={client}
            data={results.data}
            loading={results.isLoading}
            onPick={goTo}
          />
        )}
      </div>

      <div style={{ flex: 1 }} />

      <div style={{ display: 'flex', gap: 2 }}>
        <IconButton
          icon="columns"
          label="Compact covers"
          active={density === 's'}
          onClick={() => setDensity('s')}
        />
        <IconButton
          icon="grid"
          label="Comfortable covers"
          active={density === 'm'}
          onClick={() => setDensity('m')}
        />
      </div>

      <JobIndicator
        status={status}
        label={status === 'scanning' && overall != null ? `Scanning… ${overall}%` : undefined}
        jobs={jobItems}
      />

      <Tooltip label={theme === 'dark' ? 'Switch to light' : 'Switch to dark'}>
        <IconButton
          icon={theme === 'dark' ? 'sun' : 'moon'}
          variant="ghost"
          label="Toggle theme"
          onClick={toggleTheme}
        />
      </Tooltip>

      <AccountChip fallbackName={info.data?.name || 'ComicHub'} />
    </header>
  );
}

/** Type-ahead results dropdown under the search box. Rows are buttons so focus stays
 *  inside the search container (keeping the panel open until a pick or click-away). */
function SearchResultsPanel({
  client,
  data,
  loading,
  onPick,
}: {
  client: ComicHubClient;
  data: SearchResults | undefined;
  loading: boolean;
  onPick: (to: '/series/$id' | '/book/$id', id: string) => void;
}) {
  const empty = !!data && data.series.length === 0 && data.books.length === 0;
  return (
    <div
      role="listbox"
      style={{
        position: 'absolute',
        top: 'calc(100% + 6px)',
        left: 0,
        right: 0,
        zIndex: 30,
        maxHeight: 420,
        overflowY: 'auto',
        padding: 'var(--space-2)',
        background: 'var(--surface-raised)',
        border: '1px solid var(--border-hairline)',
        borderRadius: 'var(--radius-md)',
        boxShadow: 'var(--shadow-popover)',
      }}
    >
      {loading && !data && <PanelMessage>Searching…</PanelMessage>}
      {empty && <PanelMessage>No matches</PanelMessage>}

      {data && data.series.length > 0 && <PanelHeading>Series</PanelHeading>}
      {data?.series.map((s) => (
        <ResultRow
          key={s.id}
          cover={s.coverBookId ? client.coverUrl(s.coverBookId, 80) : undefined}
          title={s.name}
          subtitle={s.year ? String(s.year) : 'Series'}
          onClick={() => onPick('/series/$id', s.id)}
        />
      ))}

      {data && data.books.length > 0 && <PanelHeading>Issues</PanelHeading>}
      {data?.books.map((b) => (
        <ResultRow
          key={b.id}
          cover={client.coverUrl(b.id, 80)}
          title={b.title || b.seriesName || issueLabel(b.number) || 'Untitled'}
          subtitle={
            [b.seriesName, issueLabel(b.number)].filter(Boolean).join(' · ') ||
            b.format.toUpperCase()
          }
          onClick={() => onPick('/book/$id', b.id)}
        />
      ))}
    </div>
  );
}

function PanelHeading({ children }: { children: React.ReactNode }) {
  return (
    <div
      style={{
        padding: 'var(--space-2) var(--space-2) var(--space-1)',
        font: 'var(--text-label)',
        letterSpacing: 'var(--tracking-label)',
        textTransform: 'uppercase',
        color: 'var(--text-tertiary)',
      }}
    >
      {children}
    </div>
  );
}

function PanelMessage({ children }: { children: React.ReactNode }) {
  return (
    <div
      style={{
        padding: 'var(--space-3)',
        fontSize: 'var(--text-small)',
        color: 'var(--text-tertiary)',
      }}
    >
      {children}
    </div>
  );
}

function ResultRow({
  cover,
  title,
  subtitle,
  onClick,
}: {
  cover?: string;
  title: string;
  subtitle: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 'var(--space-3)',
        width: '100%',
        padding: 'var(--space-2)',
        border: 'none',
        borderRadius: 'var(--radius-sm)',
        background: 'transparent',
        color: 'var(--text-primary)',
        textAlign: 'left',
        cursor: 'pointer',
      }}
      onMouseEnter={(e) => (e.currentTarget.style.background = 'var(--surface-card)')}
      onMouseLeave={(e) => (e.currentTarget.style.background = 'transparent')}
    >
      <span
        style={{
          flex: 'none',
          width: 28,
          height: 42,
          borderRadius: 'var(--radius-sm)',
          overflow: 'hidden',
          background: 'var(--surface-cover)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
        }}
      >
        {cover ? (
          <img src={cover} alt="" width={28} height={42} style={{ objectFit: 'cover' }} />
        ) : (
          <Icon name="book" size={14} />
        )}
      </span>
      <span style={{ minWidth: 0 }}>
        <span
          style={{
            display: 'block',
            fontSize: 'var(--text-small)',
            whiteSpace: 'nowrap',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
          }}
        >
          {title}
        </span>
        <span
          style={{
            display: 'block',
            fontSize: 'var(--text-label)',
            color: 'var(--text-tertiary)',
            whiteSpace: 'nowrap',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
          }}
        >
          {subtitle}
        </span>
      </span>
    </button>
  );
}

function jobLabel(type: string): string {
  switch (type) {
    case 'scan':
      return 'Library scan';
    case 'thumbnail':
      return 'Thumbnails';
    case 'match':
      return 'Metadata match';
    default:
      return type.charAt(0).toUpperCase() + type.slice(1);
  }
}
