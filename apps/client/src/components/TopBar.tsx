import { Input, JobIndicator, IconButton, Tooltip, type JobItem } from '@comichub/ui';
import { useUiStore } from '../store/ui.js';

/** 56px utility bar: catalog search, background-job status, and the theme toggle. */
export function TopBar() {
  const search = useUiStore((s) => s.search);
  const setSearch = useUiStore((s) => s.setSearch);
  const theme = useUiStore((s) => s.theme);
  const toggleTheme = useUiStore((s) => s.toggleTheme);
  const jobs = useUiStore((s) => s.jobs);

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
        padding: '0 20px',
        borderBottom: '1px solid var(--border-hairline)',
        background: 'var(--surface-raised)',
      }}
    >
      <div style={{ flex: 1, maxWidth: 460 }}>
        <Input
          icon="search"
          size="sm"
          placeholder="Search this library"
          aria-label="Search the catalog"
          value={search}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => setSearch(e.target.value)}
        />
      </div>

      <div style={{ flex: 1 }} />

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
