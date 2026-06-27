import { useRouter, useRouterState } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import {
  Input,
  JobIndicator,
  IconButton,
  Tooltip,
  Button,
  Avatar,
  type JobItem,
} from '@comichub/ui';
import { useClient } from '../lib/client.js';
import { useUiStore } from '../store/ui.js';

/** 56px utility bar: back nav, catalog search, view density, job status, theme, identity. */
export function TopBar() {
  const router = useRouter();
  const client = useClient();
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const search = useUiStore((s) => s.search);
  const setSearch = useUiStore((s) => s.setSearch);
  const theme = useUiStore((s) => s.theme);
  const toggleTheme = useUiStore((s) => s.toggleTheme);
  const density = useUiStore((s) => s.density);
  const setDensity = useUiStore((s) => s.setDensity);
  const jobs = useUiStore((s) => s.jobs);

  const info = useQuery({ queryKey: ['server', 'info'], queryFn: () => client.serverInfo() });
  const canGoBack = pathname.startsWith('/series/') || pathname.startsWith('/book/');

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

      <div style={{ flex: 1, maxWidth: 440 }}>
        <Input
          icon="search"
          size="sm"
          placeholder="Search title, series, writer…"
          aria-label="Search the catalog"
          value={search}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => setSearch(e.target.value)}
        />
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

      <Avatar name={info.data?.name || 'ComicHub'} />
    </header>
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
