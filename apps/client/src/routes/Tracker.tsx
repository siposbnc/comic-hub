import { useMemo, useRef, useState, type CSSProperties } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';
import type { TrackerTrack, TrackerIssue, BookCard } from '@comichub/api-client';
import { Icon, Button, IconButton, Input, Select, Switch, Badge, EmptyState } from '@comichub/ui';
import { useClient } from '../lib/client.js';
import {
  useLibraries,
  useCollections,
  useReadingLists,
  useSmartLists,
  useTracker,
  useCreateTrack,
  useRenameTrack,
  useDeleteTrack,
  useAddTrackIssues,
  useToggleTrackerIssue,
  useRangeMarkTracker,
} from '../lib/queries.js';
import { LoadingState, ErrorState, Page } from '../components/Page.js';

/**
 * Tracker — the per-user reading matrix (design-handoff-tracker). Every tracked series is a
 * row, every issue a clickable cell; a wall of cyan is your progress. Click a number to
 * toggle read; shift-click to mark a run. Library series project in automatically; standalone
 * tracks and gap issues are the user's own additions. Frozen panes: sticky ruler + row header.
 */
export function Tracker() {
  const tracker = useTracker();

  if (tracker.isLoading) {
    return (
      <Page title="Tracker">
        <LoadingState />
      </Page>
    );
  }
  if (tracker.isError || !tracker.data) {
    return (
      <Page title="Tracker">
        <ErrorState
          message={
            tracker.error instanceof Error ? tracker.error.message : 'Could not load the tracker.'
          }
          onRetry={() => tracker.refetch()}
        />
      </Page>
    );
  }
  return <TrackerScreen tracks={tracker.data} />;
}

// ── Cell state model ───────────────────────────────────────────────────────────────────

type CellState = 'read' | 'reading' | 'partial' | 'owned' | 'gap' | 'manual-read';
type Density = 's' | 'm' | 'l';

const DENSITY: Record<
  Density,
  { cell: number; cellH: number; font: string; header: number; row: number; bar: boolean }
> = {
  s: { cell: 22, cellH: 22, font: '0.6rem', header: 180, row: 26, bar: false },
  m: { cell: 30, cellH: 28, font: '0.68rem', header: 220, row: 34, bar: true },
  l: { cell: 40, cellH: 34, font: '0.76rem', header: 260, row: 42, bar: true },
};

const CELL_STYLES: Record<CellState, CSSProperties> = {
  read: { background: 'var(--accent)', color: 'var(--text-on-accent)' },
  'manual-read': { background: 'var(--accent)', color: 'var(--text-on-accent)' },
  reading: {
    background: 'color-mix(in oklab, var(--accent) 20%, var(--surface-card))',
    color: 'var(--accent)',
    boxShadow: 'inset 0 -3px 0 var(--accent)',
    fontWeight: 700,
  },
  // Half-read: a complex issue (with sub-issues) where some but not all are read.
  // Diagonal fill — top-left accent triangle over the owned surface — with a dark
  // halo on the number so it stays legible across the bright-cyan / dark-ink seam.
  partial: {
    background: 'linear-gradient(135deg, var(--accent) 50%, var(--surface-card) 50%)',
    color: 'var(--paper-100)',
    border: '1px solid var(--border-hairline)',
    textShadow: '0 0 3px var(--bg-app)',
  },
  owned: {
    background: 'var(--surface-card)',
    color: 'var(--paper-100)',
    border: '1px solid var(--border-hairline)',
  },
  gap: {
    background: 'transparent',
    color: 'var(--paper-600)',
    border: '1px dashed var(--border-hairline)',
    opacity: 0.75,
  },
};

function cellState(issue: TrackerIssue): CellState {
  if (issue.state === 'read') return issue.bookId ? 'read' : 'manual-read';
  if (issue.state === 'reading') return 'reading';
  return issue.bookId ? 'owned' : 'gap';
}

function aggState(stack: TrackerIssue[]): CellState {
  const readCount = stack.filter((i) => i.state === 'read').length;
  if (readCount === stack.length) return stack.some((i) => i.bookId) ? 'read' : 'manual-read';
  // Some (but not all) of the stack is read — the half-read complex issue.
  if (readCount > 0) return 'partial';
  if (stack.some((i) => i.state === 'reading')) return 'reading';
  if (stack.some((i) => i.bookId)) return 'owned';
  return 'gap';
}

interface TrackStats {
  total: number;
  read: number;
  reading: number;
  gaps: number;
  pct: number;
}
function trackStats(t: TrackerTrack): TrackStats {
  const total = t.issues.length;
  const read = t.issues.filter((i) => i.state === 'read').length;
  const reading = t.issues.filter((i) => i.state === 'reading').length;
  const gaps = t.issues.filter((i) => !i.bookId).length;
  return { total, read, reading, gaps, pct: total ? Math.round((read / total) * 100) : 0 };
}

// ── Screen ─────────────────────────────────────────────────────────────────────────────

// Scope encodes the active filter source: 'all' | 'standalone' | 'lib:<id>' | 'col:<id>' |
// 'rl:<id>' | 'sm:<id>'. Collection / reading-list / smart-list scopes are book-level, so a
// series is in scope when any of its books belongs to that grouping.
type Scope = 'all' | 'standalone' | string;
type Status = 'all' | 'progress' | 'incomplete' | 'gaps';

const GROUPING_PREFIXES = ['col:', 'rl:', 'sm:'];
const isGrouping = (scope: Scope) => GROUPING_PREFIXES.some((p) => scope.startsWith(p));

/**
 * For a collection / reading-list / smart-list scope, the set of series ids whose books fall
 * in that grouping. Returns null when the scope isn't a grouping, or while it's still loading
 * (so the grid shows the library rows rather than flashing "no matches").
 */
function useScopeSeriesIds(scope: Scope): Set<string> | null {
  const client = useClient();
  const kind: 'col' | 'rl' | 'sm' | null = scope.startsWith('col:')
    ? 'col'
    : scope.startsWith('rl:')
      ? 'rl'
      : scope.startsWith('sm:')
        ? 'sm'
        : null;
  const id = kind ? scope.slice(scope.indexOf(':') + 1) : '';
  const q = useQuery({
    queryKey: ['trackerScope', kind, id],
    enabled: kind !== null,
    queryFn: async (): Promise<BookCard[]> => {
      if (kind === 'col') return (await client.collection(id)).books;
      if (kind === 'rl') return (await client.readingList(id)).books;
      return (await client.smartListResults(id)).books;
    },
  });
  return useMemo(() => {
    if (kind === null || !q.data) return null;
    const set = new Set<string>();
    q.data.forEach((b) => b.seriesId && set.add(b.seriesId));
    return set;
  }, [kind, q.data]);
}

function TrackerScreen({ tracks }: { tracks: TrackerTrack[] }) {
  const [q, setQ] = useState('');
  const [scope, setScope] = useState<Scope>('all');
  const [hideRead, setHideRead] = useState(false);
  const [status, setStatus] = useState<Status>('all');
  const [density, setDensity] = useState<Density>('m');
  const [addOpen, setAddOpen] = useState(false);

  const libraries = useLibraries();
  const collections = useCollections();
  const readingLists = useReadingLists();
  const smartLists = useSmartLists();
  const scopeSeriesIds = useScopeSeriesIds(scope);

  const visible = useMemo(
    () =>
      tracks.filter((t) => {
        if (q && !t.name.toLowerCase().includes(q.toLowerCase())) return false;
        if (scope === 'standalone' && t.link !== 'manual') return false;
        if (scope.startsWith('lib:') && t.libraryId !== scope.slice(4)) return false;
        if (isGrouping(scope)) {
          if (!t.seriesId) return false; // standalone tracks aren't in a collection/list
          if (scopeSeriesIds && !scopeSeriesIds.has(t.seriesId)) return false;
        }
        const st = trackStats(t);
        const complete = st.total > 0 && st.read === st.total;
        if (hideRead && complete) return false;
        // "In progress" = a series you've started but not finished (some issues read, or one
        // mid-read) — not just one with an actively mid-read issue.
        if (status === 'progress' && (!(st.read > 0 || st.reading > 0) || complete)) return false;
        if (status === 'incomplete' && complete) return false;
        if (status === 'gaps' && st.gaps === 0) return false;
        return true;
      }),
    [tracks, q, scope, hideRead, status, scopeSeriesIds],
  );

  const sum = useMemo(
    () =>
      visible.reduce(
        (a, t) => {
          const s = trackStats(t);
          return {
            issues: a.issues + s.total,
            read: a.read + s.read,
            reading: a.reading + s.reading,
            gaps: a.gaps + s.gaps,
          };
        },
        { issues: 0, read: 0, reading: 0, gaps: 0 },
      ),
    [visible],
  );
  const pct = sum.issues ? Math.round((sum.read / sum.issues) * 100) : 0;
  const isFiltered = visible.length !== tracks.length;

  const scopeLabel = useMemo(() => {
    if (scope === 'all') return 'All';
    if (scope === 'standalone') return 'Standalone';
    const id = scope.slice(scope.indexOf(':') + 1);
    const name = (arr?: { id: string; name: string }[], fallback = 'Scope') =>
      arr?.find((x) => x.id === id)?.name ?? fallback;
    if (scope.startsWith('lib:')) return name(libraries.data, 'Library');
    if (scope.startsWith('col:')) return name(collections.data, 'Collection');
    if (scope.startsWith('rl:')) return name(readingLists.data, 'List');
    if (scope.startsWith('sm:')) return name(smartLists.data, 'Smart list');
    return 'All';
  }, [scope, libraries.data, collections.data, readingLists.data, smartLists.data]);

  const clearFilters = () => {
    setQ('');
    setScope('all');
    setHideRead(false);
    setStatus('all');
  };

  return (
    <div
      style={{
        height: '100%',
        minHeight: 0,
        display: 'flex',
        flexDirection: 'column',
        padding: '22px 24px 20px',
        gap: 14,
      }}
    >
      {/* header */}
      <div style={{ flex: 'none' }}>
        <Eyebrow color="var(--accent)">Tracker</Eyebrow>
        <div style={{ display: 'flex', alignItems: 'flex-end', gap: 16, flexWrap: 'wrap' }}>
          <h1
            style={{
              margin: '6px 0 0',
              fontFamily: 'var(--font-display)',
              fontWeight: 800,
              fontSize: 'var(--text-display-l)',
              letterSpacing: '-0.01em',
              lineHeight: 1,
              color: 'var(--text-primary)',
            }}
          >
            Tracker
          </h1>
          <p
            style={{
              margin: 0,
              maxWidth: 560,
              fontSize: '0.86rem',
              color: 'var(--text-secondary)',
            }}
          >
            Every series, every issue, at a glance. Click a number to mark it read.
          </p>
        </div>
      </div>

      {tracks.length === 0 ? (
        <div style={{ flex: 1, minHeight: 0 }}>
          <EmptyState
            title="Nothing tracked yet."
            action={
              <Button variant="primary" icon="plus" onClick={() => setAddOpen(true)}>
                Add series
              </Button>
            }
          >
            ComicHub tracks every series in your libraries here automatically — or add a series to
            start.
          </EmptyState>
        </div>
      ) : (
        <>
          {/* summary */}
          <SummaryPanel
            sum={sum}
            pct={pct}
            count={visible.length}
            total={tracks.length}
            filtered={isFiltered}
          />

          {/* toolbar */}
          <div style={{ flex: 'none' }}>
            <Toolbar
              q={q}
              setQ={setQ}
              scope={scope}
              setScope={setScope}
              libraries={libraries.data ?? []}
              collections={collections.data ?? []}
              readingLists={readingLists.data ?? []}
              smartLists={smartLists.data ?? []}
              hideRead={hideRead}
              setHideRead={setHideRead}
              status={status}
              setStatus={setStatus}
              density={density}
              setDensity={setDensity}
              onAdd={() => setAddOpen(true)}
            />
          </div>

          {/* grid */}
          {visible.length === 0 ? (
            <div style={{ flex: 1, minHeight: 0 }}>
              <EmptyState
                title={q ? `No series match “${q}”.` : 'No series match these filters.'}
                action={
                  <Button variant="ghost" onClick={clearFilters}>
                    Clear filters
                  </Button>
                }
              >
                Try a different name, or clear the filters.
              </EmptyState>
            </div>
          ) : (
            <TrackerGrid tracks={visible} density={density} scopeLabel={scopeLabel} />
          )}
        </>
      )}

      {addOpen && <AddSeriesDialog onClose={() => setAddOpen(false)} />}
    </div>
  );
}

// ── Summary panel ──────────────────────────────────────────────────────────────────────

function SummaryPanel({
  sum,
  pct,
  count,
  total,
  filtered,
}: {
  sum: { issues: number; read: number; reading: number; gaps: number };
  pct: number;
  count: number;
  total: number;
  filtered: boolean;
}) {
  return (
    <div
      style={{
        flex: 'none',
        padding: '14px 18px',
        background: 'var(--surface-raised)',
        border: '1px solid var(--border-hairline)',
        borderRadius: 8,
      }}
    >
      <div
        className="ch-mono"
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 12,
          flexWrap: 'wrap',
          fontSize: '0.7rem',
          fontVariantNumeric: 'tabular-nums',
          color: 'var(--text-secondary)',
        }}
      >
        <span style={{ color: 'var(--paper-100)', fontWeight: 600 }}>{count} series</span>
        {filtered && <span style={{ color: 'var(--paper-600)' }}>of {total}</span>}
        <Divider />
        <span>{sum.issues.toLocaleString()} issues</span>
        <Divider />
        <span>
          <span style={{ color: 'var(--accent)' }}>{sum.read.toLocaleString()} read</span> ·{' '}
          {sum.reading} reading · {(sum.issues - sum.read - sum.reading).toLocaleString()} to go
        </span>
        <Divider />
        <span>{sum.gaps} missing</span>
        <span style={{ flex: 1 }} />
        <Badge tone="accent" mono>
          {pct}% complete
        </Badge>
      </div>
      <div className="ch-progress" style={{ marginTop: 11, borderRadius: 999 }}>
        <span style={{ width: `${pct}%`, borderRadius: 999 }} />
      </div>
    </div>
  );
}

function Divider() {
  return <span style={{ width: 1, height: 12, background: 'var(--border-hairline)' }} />;
}

// ── Toolbar ────────────────────────────────────────────────────────────────────────────

type NamedRef = { id: string; name: string };

function Toolbar({
  q,
  setQ,
  scope,
  setScope,
  libraries,
  collections,
  readingLists,
  smartLists,
  hideRead,
  setHideRead,
  status,
  setStatus,
  density,
  setDensity,
  onAdd,
}: {
  q: string;
  setQ: (v: string) => void;
  scope: Scope;
  setScope: (v: Scope) => void;
  libraries: NamedRef[];
  collections: NamedRef[];
  readingLists: NamedRef[];
  smartLists: NamedRef[];
  hideRead: boolean;
  setHideRead: (v: boolean) => void;
  status: Status;
  setStatus: (v: Status) => void;
  density: Density;
  setDensity: (v: Density) => void;
  onAdd: () => void;
}) {
  const scopeGroups: { label: string; prefix: string; items: NamedRef[] }[] = [
    { label: 'Libraries', prefix: 'lib', items: libraries },
    { label: 'Collections', prefix: 'col', items: collections },
    { label: 'Reading lists', prefix: 'rl', items: readingLists },
    { label: 'Smart lists', prefix: 'sm', items: smartLists },
  ];
  const chips: [Status, string][] = [
    ['all', 'All'],
    ['progress', 'In progress'],
    ['incomplete', 'Incomplete'],
    ['gaps', 'Has gaps'],
  ];
  const legend: { label: string; sw: CSSProperties }[] = [
    { label: 'Read', sw: { background: 'var(--accent)' } },
    {
      label: 'Reading',
      sw: {
        background: 'color-mix(in oklab, var(--accent) 20%, var(--surface-card))',
        boxShadow: 'inset 0 -3px 0 var(--accent)',
      },
    },
    {
      label: 'Half read',
      sw: {
        background: 'linear-gradient(135deg, var(--accent) 50%, var(--surface-card) 50%)',
        border: '1px solid var(--border-hairline)',
      },
    },
    {
      label: 'Unread',
      sw: { background: 'var(--surface-card)', border: '1px solid var(--border-hairline)' },
    },
    { label: 'Missing', sw: { border: '1px dashed var(--border-hairline)', opacity: 0.8 } },
  ];
  return (
    <div style={{ display: 'flex', alignItems: 'center', flexWrap: 'wrap', gap: 10, rowGap: 12 }}>
      <Button variant="primary" size="sm" icon="plus" onClick={onAdd}>
        Add series
      </Button>
      <div style={{ width: 190 }}>
        <Input
          placeholder="Filter series…"
          icon="search"
          size="sm"
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
      </div>
      <Select size="sm" value={scope} onChange={(e) => setScope(e.target.value)} aria-label="Scope">
        <option value="all">All series</option>
        <option value="standalone">Standalone only</option>
        {scopeGroups
          .filter((g) => g.items.length > 0)
          .flatMap((g) => [
            <option key={g.prefix} disabled>
              {`— ${g.label} —`}
            </option>,
            ...g.items.map((it) => (
              <option key={`${g.prefix}:${it.id}`} value={`${g.prefix}:${it.id}`}>
                {it.name}
              </option>
            )),
          ])}
      </Select>
      <Switch
        checked={hideRead}
        onChange={setHideRead}
        label={
          <span
            className="ch-mono"
            style={{
              fontSize: '0.68rem',
              letterSpacing: '0.08em',
              textTransform: 'uppercase',
              color: 'var(--text-secondary)',
            }}
          >
            Hide read
          </span>
        }
      />
      <div style={{ display: 'flex', gap: 4 }}>
        {chips.map(([v, l]) => {
          const on = status === v;
          return (
            <button
              key={v}
              type="button"
              onClick={() => setStatus(v)}
              style={{
                height: 26,
                padding: '0 10px',
                borderRadius: 'var(--radius-sm)',
                cursor: 'pointer',
                border: `1px solid ${on ? 'var(--accent)' : 'var(--border-hairline)'}`,
                background: on ? 'var(--accent-soft)' : 'transparent',
                fontFamily: 'var(--font-body)',
                fontSize: '0.76rem',
                fontWeight: on ? 600 : 500,
                color: on ? 'var(--accent)' : 'var(--text-secondary)',
              }}
            >
              {l}
            </button>
          );
        })}
      </div>
      <div
        style={{
          display: 'flex',
          gap: 2,
          padding: 2,
          background: 'var(--bg-app)',
          borderRadius: 'var(--radius-md)',
          border: '1px solid var(--border-hairline)',
        }}
      >
        {(['s', 'm', 'l'] as Density[]).map((d) => {
          const on = density === d;
          return (
            <button
              key={d}
              type="button"
              onClick={() => setDensity(d)}
              aria-label={`Density ${d.toUpperCase()}`}
              style={{
                width: 26,
                height: 24,
                border: 'none',
                borderRadius: 'var(--radius-sm)',
                cursor: 'pointer',
                fontFamily: 'var(--font-mono)',
                fontSize: '0.66rem',
                fontWeight: 600,
                textTransform: 'uppercase',
                background: on ? 'var(--accent)' : 'transparent',
                color: on ? 'var(--text-on-accent)' : 'var(--text-tertiary)',
              }}
            >
              {d}
            </button>
          );
        })}
      </div>
      <span style={{ flex: 1 }} />
      <div style={{ display: 'flex', alignItems: 'center', gap: 13, flexWrap: 'wrap' }}>
        {legend.map((k) => (
          <span key={k.label} style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
            <span
              style={{
                width: 14,
                height: 13,
                borderRadius: 2,
                flex: 'none',
                display: 'inline-block',
                ...k.sw,
              }}
            />
            <span
              className="ch-mono"
              style={{
                fontSize: '0.62rem',
                letterSpacing: '0.06em',
                textTransform: 'uppercase',
                color: 'var(--paper-600)',
              }}
            >
              {k.label}
            </span>
          </span>
        ))}
      </div>
    </div>
  );
}

// ── Grid ───────────────────────────────────────────────────────────────────────────────

// Only identifiers + geometry are captured on hover; the popover derives live data from the
// current `tracks` each render, so optimistic toggles show instantly. A cell hover opens the
// issue popover; a row-header hover opens the series popover.
type HoverState =
  | { type: 'cell'; trackId: string; baseNum: number; rect: DOMRect }
  | { type: 'header'; trackId: string; rect: DOMRect };

function TrackerGrid({
  tracks,
  density,
  scopeLabel,
}: {
  tracks: TrackerTrack[];
  density: Density;
  scopeLabel: string;
}) {
  const d = DENSITY[density];
  const [hover, setHover] = useState<HoverState | null>(null);
  const [addFor, setAddFor] = useState<string | null>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const hideTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const anchor = useRef<{ trackId: string; issueId: string; wasRead: boolean } | null>(null);

  const toggle = useToggleTrackerIssue();
  const range = useRangeMarkTracker();

  const GAP = 1;

  const ruler = useMemo(() => {
    const set = new Set<number>();
    tracks.forEach((t) => t.issues.forEach((i) => set.add(Math.floor(i.sort))));
    return [...set].sort((a, b) => a - b);
  }, [tracks]);

  // Group each series' main row with its specials (Annuals, one-shots, …) so they
  // render as one block: the main row, then its specials as indented sub-rows.
  // Standalone tracks (no seriesId) each form their own single-row group.
  const rows = useMemo(() => {
    const groups: { key: string; main: TrackerTrack | null; specials: TrackerTrack[] }[] = [];
    const idx = new Map<string, number>();
    for (const t of tracks) {
      const key = t.seriesId ?? `manual:${t.id}`;
      let gi = idx.get(key);
      if (gi === undefined) {
        gi = groups.length;
        idx.set(key, gi);
        groups.push({ key, main: null, specials: [] });
      }
      if (t.special) groups[gi]!.specials.push(t);
      else groups[gi]!.main = t;
    }
    // Flatten to a render list, tagging each row's role within its group.
    const out: { track: TrackerTrack; isSub: boolean; isLastInGroup: boolean }[] = [];
    for (const g of groups) {
      // A group with only specials (no main row) shows its first special as the head.
      const members = g.main ? [g.main, ...g.specials] : g.specials;
      members.forEach((track, i) =>
        out.push({
          track,
          isSub: g.main ? track !== g.main : i > 0,
          isLastInGroup: i === members.length - 1,
        }),
      );
    }
    return out;
  }, [tracks]);

  const scheduleHide = () => {
    if (hideTimer.current) clearTimeout(hideTimer.current);
    hideTimer.current = setTimeout(() => setHover(null), 160);
  };
  const onHover = (h: HoverState | null) => {
    if (!h) {
      scheduleHide();
      return;
    }
    if (hideTimer.current) clearTimeout(hideTimer.current);
    setHover(h);
  };
  const holdPopover = (hold: boolean) => {
    if (hold && hideTimer.current) clearTimeout(hideTimer.current);
    else if (!hold) scheduleHide();
  };

  const onToggle = (track: TrackerTrack, issue: TrackerIssue, shift: boolean) => {
    const cur = anchor.current;
    if (shift && cur && cur.trackId === track.id) {
      const aIdx = track.issues.findIndex(
        (i) => i.id === issue.id || (i.bookId && i.bookId === issue.bookId),
      );
      const cIdx = track.issues.findIndex((i) => i === issue);
      if (aIdx >= 0 && cIdx >= 0) {
        // anchor stays where the run began; use its stored id
        const startIdx = track.issues.findIndex(
          (i) => i.id === cur.issueId || (i.bookId && i.bookId === cur.issueId),
        );
        const [lo, hi] =
          startIdx >= 0 ? [Math.min(startIdx, cIdx), Math.max(startIdx, cIdx)] : [cIdx, cIdx];
        const run = track.issues.slice(lo, hi + 1);
        const read = !cur.wasRead;
        range.mutate({
          bookIds: run.filter((i) => i.bookId).map((i) => i.bookId as string),
          issueIds: run.filter((i) => !i.bookId && i.id).map((i) => i.id as string),
          read,
        });
        return;
      }
    }
    anchor.current = {
      trackId: track.id,
      issueId: (issue.bookId || issue.id) ?? '',
      wasRead: issue.state === 'read',
    };
    toggle.mutate({ bookId: issue.bookId, issueId: issue.id, read: issue.state !== 'read' });
  };

  // Derive the hovered row/cell from live data so optimistic toggles update the popover.
  const hoverTrack = hover ? tracks.find((t) => t.id === hover.trackId) : undefined;
  const hoverStack =
    hover?.type === 'cell' && hoverTrack
      ? hoverTrack.issues.filter((i) => Math.floor(i.sort) === hover.baseNum)
      : [];

  return (
    <div
      ref={containerRef}
      style={{
        position: 'relative',
        flex: 1,
        minHeight: 0,
        border: '1px solid var(--border-hairline)',
        borderRadius: 'var(--radius-md)',
        overflow: 'hidden',
        background: 'var(--bg-app)',
      }}
    >
      <div style={{ position: 'absolute', inset: 0, overflow: 'auto' }} onMouseLeave={scheduleHide}>
        <div style={{ display: 'inline-block', minWidth: '100%' }}>
          {/* sticky ruler */}
          <div
            style={{
              display: 'flex',
              position: 'sticky',
              top: 0,
              zIndex: 30,
              background: 'var(--surface-raised)',
              borderBottom: '1px solid var(--border-hairline)',
            }}
          >
            <div
              className="ch-mono"
              style={{
                position: 'sticky',
                left: 0,
                zIndex: 31,
                width: d.header,
                flex: 'none',
                display: 'flex',
                alignItems: 'center',
                padding: '0 12px',
                height: 30,
                fontSize: '0.6rem',
                fontWeight: 600,
                letterSpacing: '0.16em',
                textTransform: 'uppercase',
                color: 'var(--accent)',
                background: 'var(--surface-raised)',
                borderRight: '1px solid var(--border-strong)',
              }}
            >
              {scopeLabel}
            </div>
            {ruler.map((s) => {
              const tick = s % 5 === 0;
              return (
                <div
                  key={s}
                  className="ch-mono"
                  style={{
                    width: d.cell,
                    marginRight: GAP,
                    flex: 'none',
                    height: 30,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    fontSize: d.font,
                    fontVariantNumeric: 'tabular-nums',
                    color: tick ? 'var(--paper-100)' : 'var(--paper-400)',
                    fontWeight: tick ? 600 : 500,
                    boxShadow: tick ? 'inset 0 -2px 0 var(--border-strong)' : 'none',
                  }}
                >
                  {s}
                </div>
              );
            })}
            <div style={{ width: d.cell + 8, flex: 'none' }} />
          </div>

          {/* rows */}
          {rows.map(({ track, isSub, isLastInGroup }) => (
            <TrackRow
              key={track.id}
              t={track}
              ruler={ruler}
              d={d}
              density={density}
              gap={GAP}
              onToggle={onToggle}
              onHover={onHover}
              addOpen={addFor === track.id}
              setAddFor={setAddFor}
              isSub={isSub}
              isLastInGroup={isLastInGroup}
            />
          ))}
        </div>
      </div>
      {hover?.type === 'cell' && hoverTrack && hoverStack.length > 0 && (
        <CellPopover
          track={hoverTrack}
          stack={hoverStack}
          baseNum={hover.baseNum}
          rect={hover.rect}
          container={containerRef.current}
          onToggle={onToggle}
          onHold={holdPopover}
        />
      )}
      {hover?.type === 'header' && hoverTrack && (
        <SeriesPopover
          track={hoverTrack}
          rect={hover.rect}
          container={containerRef.current}
          onHold={holdPopover}
        />
      )}
    </div>
  );
}

// ── Row ────────────────────────────────────────────────────────────────────────────────

function TrackRow({
  t,
  ruler,
  d,
  density,
  gap,
  onToggle,
  onHover,
  addOpen,
  setAddFor,
  isSub,
  isLastInGroup,
}: {
  t: TrackerTrack;
  ruler: number[];
  d: (typeof DENSITY)[Density];
  density: Density;
  gap: number;
  onToggle: (track: TrackerTrack, issue: TrackerIssue, shift: boolean) => void;
  onHover: (h: HoverState | null) => void;
  addOpen: boolean;
  setAddFor: (id: string | null) => void;
  isSub: boolean;
  isLastInGroup: boolean;
}) {
  const navigate = useNavigate();
  const [rowHover, setRowHover] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const st = trackStats(t);
  const complete = st.total > 0 && st.read === st.total;

  const byBase = useMemo(() => {
    const m = new Map<number, TrackerIssue[]>();
    t.issues.forEach((i) => {
      const b = Math.floor(i.sort);
      const arr = m.get(b);
      if (arr) arr.push(i);
      else m.set(b, [i]);
    });
    return m;
  }, [t.issues]);
  const lastBase = t.issues.length ? Math.floor(t.issues[t.issues.length - 1]!.sort) : -1;

  const openHeader = () => {
    if (t.seriesId) navigate({ to: '/series/$id', params: { id: t.seriesId } });
    else setEditOpen(true);
  };

  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'stretch',
        // The seam only closes a group — a series and its specials share one block.
        borderBottom: isLastInGroup ? '1px solid var(--border-hairline)' : 'none',
        height: d.row,
        position: 'relative',
      }}
      onMouseEnter={() => setRowHover(true)}
      onMouseLeave={() => setRowHover(false)}
    >
      {/* header (sticky left) */}
      <div
        style={{
          position: 'sticky',
          left: 0,
          zIndex: 20,
          width: d.header,
          flex: 'none',
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          padding: '0 8px 0 12px',
          background: complete ? 'var(--accent-soft)' : 'var(--surface-raised)',
          borderRight: '1px solid var(--border-strong)',
        }}
      >
        {isSub && (
          <span
            aria-hidden
            className="ch-mono"
            style={{
              flex: 'none',
              width: 14,
              textAlign: 'center',
              color: 'var(--paper-600)',
              fontSize: '0.8rem',
              lineHeight: 1,
            }}
          >
            └
          </span>
        )}
        <button
          type="button"
          onClick={openHeader}
          onMouseEnter={(e) =>
            onHover({
              type: 'header',
              trackId: t.id,
              rect: e.currentTarget.getBoundingClientRect(),
            })
          }
          onMouseLeave={() => onHover(null)}
          style={{
            flex: 1,
            minWidth: 0,
            border: 'none',
            background: 'transparent',
            padding: 0,
            textAlign: 'left',
            cursor: 'pointer',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
            <span
              style={{
                flex: 1,
                minWidth: 0,
                fontFamily: 'var(--font-body)',
                fontWeight: isSub ? 500 : 600,
                fontSize: density === 's' ? '0.7rem' : '0.78rem',
                color: isSub ? 'var(--paper-400)' : 'var(--paper-100)',
                whiteSpace: 'nowrap',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
              }}
            >
              {isSub ? (t.special ?? t.name) : t.name}
            </span>
            {t.special && !isSub && (
              <span
                className="ch-mono"
                style={{
                  flex: 'none',
                  fontSize: '0.54rem',
                  fontWeight: 700,
                  letterSpacing: '0.06em',
                  textTransform: 'uppercase',
                  color: 'var(--accent)',
                  background: 'var(--accent-soft)',
                  padding: '1px 5px',
                  borderRadius: 3,
                  whiteSpace: 'nowrap',
                }}
              >
                {t.special}
              </span>
            )}
            {complete && (
              <Icon name="check" size={12} color="var(--accent)" style={{ flex: 'none' }} />
            )}
          </div>
          {density !== 's' && (
            <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginTop: 2 }}>
              {t.link === 'library' ? (
                <Icon name="library" size={10} color="var(--accent)" />
              ) : (
                <span
                  className="ch-mono"
                  style={{
                    fontSize: '0.52rem',
                    fontWeight: 600,
                    letterSpacing: '0.08em',
                    textTransform: 'uppercase',
                    color: 'var(--paper-600)',
                    padding: '1px 4px',
                    border: '1px solid var(--border-hairline)',
                    borderRadius: 2,
                  }}
                >
                  Manual
                </span>
              )}
              <span
                className="ch-mono"
                style={{
                  fontSize: '0.6rem',
                  fontVariantNumeric: 'tabular-nums',
                  color: 'var(--paper-600)',
                }}
              >
                {st.read}/{st.total}
              </span>
              <span
                className="ch-mono"
                style={{
                  fontSize: '0.6rem',
                  fontVariantNumeric: 'tabular-nums',
                  color: complete ? 'var(--accent)' : 'var(--paper-600)',
                }}
              >
                {st.pct}%
              </span>
            </div>
          )}
        </button>
        <Icon
          name="chevron-right"
          size={13}
          color="var(--text-tertiary)"
          style={{ flex: 'none', opacity: rowHover ? 1 : 0, transition: 'opacity 100ms' }}
        />
      </div>

      {/* cells */}
      <div style={{ display: 'flex', alignItems: 'center', gap }}>
        {ruler.map((s) => {
          if (s > lastBase) return null;
          const stack = byBase.get(s);
          return stack ? (
            <IssueCell
              key={s}
              stack={stack}
              baseNum={s}
              track={t}
              d={d}
              onToggle={onToggle}
              onHover={onHover}
            />
          ) : (
            <div key={s} style={{ width: d.cell, height: d.cellH, flex: 'none' }} />
          );
        })}
        {/* end-of-row add */}
        <div style={{ position: 'relative', flex: 'none' }}>
          <button
            type="button"
            onClick={() => setAddFor(addOpen ? null : t.id)}
            aria-label={`Add issue to ${t.name}`}
            style={{
              width: d.cell,
              height: d.cellH,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              padding: 0,
              borderRadius: 2,
              border: '1px dashed var(--border-hairline)',
              background: 'transparent',
              cursor: 'pointer',
              color: 'var(--paper-600)',
              opacity: rowHover || addOpen ? 1 : 0.35,
              transition: 'opacity 100ms',
            }}
          >
            <Icon name="plus" size={12} color="currentColor" />
          </button>
          {addOpen && <AddIssuePopover track={t} onClose={() => setAddFor(null)} />}
        </div>
      </div>

      {editOpen && !t.seriesId && <EditTrackSheet track={t} onClose={() => setEditOpen(false)} />}
    </div>
  );
}

// ── Issue cell ─────────────────────────────────────────────────────────────────────────

function IssueCell({
  stack,
  baseNum,
  track,
  d,
  onToggle,
  onHover,
}: {
  stack: TrackerIssue[];
  baseNum: number;
  track: TrackerTrack;
  d: (typeof DENSITY)[Density];
  onToggle: (track: TrackerTrack, issue: TrackerIssue, shift: boolean) => void;
  onHover: (h: HoverState | null) => void;
}) {
  const [hover, setHover] = useState(false);
  const main = stack.find((i) => Number.isInteger(i.sort));
  const points = stack.filter((i) => !Number.isInteger(i.sort));
  // A complex issue (base + sub-issues) reflects the whole stack, so a read base
  // with unread points reads as half-read; a plain cell uses its own state.
  const st = points.length > 0 || !main ? aggState(stack) : cellState(main);

  return (
    <button
      type="button"
      className={hover ? 'ch-reg' : undefined}
      onClick={(e) => {
        if (main) onToggle(track, main, e.shiftKey);
        else if (stack[0]) onToggle(track, stack[0], e.shiftKey);
      }}
      onMouseEnter={(e) => {
        setHover(true);
        onHover({
          type: 'cell',
          trackId: track.id,
          baseNum,
          rect: e.currentTarget.getBoundingClientRect(),
        });
      }}
      onMouseLeave={() => {
        setHover(false);
        onHover(null);
      }}
      aria-label={`${track.name} #${baseNum}${points.length ? ` (+${points.length})` : ''} — ${st}`}
      style={{
        position: 'relative',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        width: d.cell,
        height: d.cellH,
        flex: 'none',
        padding: 0,
        fontFamily: 'var(--font-mono)',
        fontSize: d.font,
        fontWeight: 500,
        fontVariantNumeric: 'tabular-nums',
        letterSpacing: '-0.01em',
        borderRadius: 2,
        cursor: 'pointer',
        border: '1px solid transparent',
        transition: 'background 100ms, color 100ms',
        ...CELL_STYLES[st],
      }}
    >
      <span
        style={
          st === 'manual-read'
            ? { borderBottom: '1px dotted var(--text-on-accent)', lineHeight: 1.3 }
            : undefined
        }
      >
        {baseNum}
      </span>
      {points.length > 0 && (
        <span
          style={{
            position: 'absolute',
            bottom: 1,
            right: 1,
            width: 0,
            height: 0,
            borderLeft: '5px solid transparent',
            borderBottom: '5px solid currentColor',
            opacity: 0.65,
          }}
        />
      )}
    </button>
  );
}

// ── Cell popover ───────────────────────────────────────────────────────────────────────

function CellPopover({
  track,
  stack,
  baseNum,
  rect,
  container,
  onToggle,
  onHold,
}: {
  track: TrackerTrack;
  stack: TrackerIssue[];
  baseNum: number;
  rect: DOMRect;
  container: HTMLDivElement | null;
  onToggle: (track: TrackerTrack, issue: TrackerIssue, shift: boolean) => void;
  onHold: (hold: boolean) => void;
}) {
  const client = useClient();
  const navigate = useNavigate();
  const main = stack.find((i) => Number.isInteger(i.sort));
  const issue = main ?? stack[0];
  if (!issue) return null;
  const points = stack;
  const owned = !!issue.bookId;

  const cRect = container?.getBoundingClientRect() ?? { left: 0, top: 0, width: 1000, height: 600 };
  const left = Math.min(
    Math.max(rect.left - cRect.left + rect.width / 2 - 130, 8),
    cRect.width - 268,
  );
  const below = rect.top - cRect.top < 220;
  const top = below ? rect.bottom - cRect.top + 10 : rect.top - cRect.top - 10;

  return (
    <div
      onMouseEnter={() => onHold(true)}
      onMouseLeave={() => onHold(false)}
      style={{
        position: 'absolute',
        left,
        top,
        transform: below ? 'none' : 'translateY(-100%)',
        width: 260,
        zIndex: 60,
        background: 'var(--surface-raised)',
        border: '1px solid var(--border-strong)',
        borderRadius: 'var(--radius-lg)',
        boxShadow: 'var(--shadow-popover)',
        padding: 12,
      }}
    >
      <div style={{ display: 'flex', gap: 11 }}>
        {owned && issue.bookId && (
          <div className="ch-reg" style={{ flex: 'none' }}>
            <img
              src={client.coverUrl(issue.bookId, 120)}
              alt=""
              style={{ width: 40, height: 60, objectFit: 'cover', display: 'block' }}
            />
          </div>
        )}
        <div style={{ flex: 1, minWidth: 0 }}>
          <div
            style={{
              fontFamily: 'var(--font-body)',
              fontWeight: 600,
              fontSize: '0.84rem',
              color: 'var(--paper-100)',
              whiteSpace: 'nowrap',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
            }}
          >
            {track.name} #{baseNum}
          </div>
          <div
            className="ch-mono"
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 5,
              fontSize: '0.64rem',
              marginTop: 6,
              color: owned ? 'var(--accent)' : 'var(--paper-600)',
            }}
          >
            <Icon
              name={owned ? 'check' : 'alert-triangle'}
              size={11}
              color={owned ? 'var(--accent)' : 'var(--paper-600)'}
            />
            {owned
              ? `${issue.pages ?? 0} pp · in your library`
              : issue.state === 'read'
                ? 'Read elsewhere · no file'
                : 'Missing — no file'}
          </div>
        </div>
      </div>

      {points.length > 1 && (
        <div
          style={{ marginTop: 10, paddingTop: 9, borderTop: '1px solid var(--border-hairline)' }}
        >
          <span
            className="ch-mono"
            style={{
              fontSize: '0.56rem',
              fontWeight: 600,
              letterSpacing: '0.16em',
              textTransform: 'uppercase',
              color: 'var(--text-tertiary)',
            }}
          >
            Point issues · click to toggle
          </span>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 3, marginTop: 7 }}>
            {points.map((i) => {
              const ist = cellState(i);
              return (
                <button
                  key={i.id ?? i.bookId ?? i.number}
                  type="button"
                  onClick={() => onToggle(track, i, false)}
                  aria-label={`#${i.number} — ${ist}`}
                  style={{
                    minWidth: 38,
                    height: 24,
                    padding: '0 6px',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    fontFamily: 'var(--font-mono)',
                    fontSize: '0.62rem',
                    fontWeight: 500,
                    fontVariantNumeric: 'tabular-nums',
                    borderRadius: 2,
                    cursor: 'pointer',
                    border: '1px solid transparent',
                    ...CELL_STYLES[ist],
                  }}
                >
                  {i.number}
                </button>
              );
            })}
          </div>
        </div>
      )}

      {owned && issue.bookId && (
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 8,
            marginTop: 10,
            paddingTop: 9,
            borderTop: '1px solid var(--border-hairline)',
          }}
        >
          <button
            type="button"
            onClick={() => navigate({ to: '/book/$id', params: { id: issue.bookId as string } })}
            className="ch-mono"
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: 5,
              fontSize: '0.64rem',
              fontWeight: 600,
              color: 'var(--accent)',
              border: 'none',
              background: 'transparent',
              cursor: 'pointer',
              padding: 0,
            }}
          >
            <Icon name="book-open" size={12} color="var(--accent)" /> Open
          </button>
          <span style={{ flex: 1 }} />
          <button
            type="button"
            onClick={() => navigate({ to: '/book/$id', params: { id: issue.bookId as string } })}
            className="ch-mono"
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: 3,
              fontSize: '0.64rem',
              color: 'var(--text-tertiary)',
              border: 'none',
              background: 'transparent',
              cursor: 'pointer',
              padding: 0,
            }}
          >
            Issue details <Icon name="chevron-right" size={11} color="var(--text-tertiary)" />
          </button>
        </div>
      )}
    </div>
  );
}

// ── Series popover (row-header hover) ──────────────────────────────────────────────────

function SeriesPopover({
  track,
  rect,
  container,
  onHold,
}: {
  track: TrackerTrack;
  rect: DOMRect;
  container: HTMLDivElement | null;
  onHold: (hold: boolean) => void;
}) {
  const client = useClient();
  const navigate = useNavigate();
  const st = trackStats(track);
  const firstOwned = track.issues.find((i) => i.bookId);

  const cRect = container?.getBoundingClientRect() ?? { left: 0, top: 0, width: 1000, height: 600 };
  const left = Math.min(Math.max(rect.left - cRect.left, 8), cRect.width - 288);
  const below = rect.bottom - cRect.top < cRect.height - 180;
  const top = below ? rect.bottom - cRect.top + 8 : rect.top - cRect.top - 8;

  return (
    <div
      onMouseEnter={() => onHold(true)}
      onMouseLeave={() => onHold(false)}
      style={{
        position: 'absolute',
        left,
        top,
        transform: below ? 'none' : 'translateY(-100%)',
        width: 280,
        zIndex: 60,
        background: 'var(--surface-raised)',
        border: '1px solid var(--border-strong)',
        borderRadius: 'var(--radius-lg)',
        boxShadow: 'var(--shadow-popover)',
        padding: 12,
      }}
    >
      <div style={{ display: 'flex', gap: 11 }}>
        {firstOwned?.bookId && (
          <div className="ch-reg" style={{ flex: 'none' }}>
            <img
              src={client.coverUrl(firstOwned.bookId, 160)}
              alt=""
              style={{ width: 46, height: 69, objectFit: 'cover', display: 'block' }}
            />
          </div>
        )}
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
            <span
              style={{
                fontFamily: 'var(--font-body)',
                fontWeight: 600,
                fontSize: '0.86rem',
                color: 'var(--paper-100)',
                whiteSpace: 'nowrap',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
              }}
            >
              {track.name}
            </span>
            {track.special && (
              <span
                className="ch-mono"
                style={{
                  flex: 'none',
                  fontSize: '0.52rem',
                  fontWeight: 700,
                  letterSpacing: '0.06em',
                  textTransform: 'uppercase',
                  color: 'var(--accent)',
                  background: 'var(--accent-soft)',
                  padding: '1px 5px',
                  borderRadius: 3,
                }}
              >
                {track.special}
              </span>
            )}
          </div>
          <div
            className="ch-mono"
            style={{ fontSize: '0.64rem', color: 'var(--text-tertiary)', marginTop: 4 }}
          >
            {track.link === 'library' ? 'In your library' : 'Standalone track'} · {st.total} issue
            {st.total === 1 ? '' : 's'}
          </div>
          <div
            className="ch-mono"
            style={{ fontSize: '0.64rem', color: 'var(--text-secondary)', marginTop: 6 }}
          >
            <span style={{ color: 'var(--accent)' }}>{st.read} read</span> · {st.total - st.read} to
            go
            {st.gaps > 0 ? ` · ${st.gaps} missing` : ''}
          </div>
          <div className="ch-progress" style={{ borderRadius: 999, height: 5, marginTop: 8 }}>
            <span style={{ width: `${st.pct}%`, borderRadius: 999 }} />
          </div>
        </div>
      </div>
      {track.seriesId && (
        <div
          style={{
            display: 'flex',
            justifyContent: 'flex-end',
            marginTop: 10,
            paddingTop: 9,
            borderTop: '1px solid var(--border-hairline)',
          }}
        >
          <button
            type="button"
            onClick={() =>
              navigate({ to: '/series/$id', params: { id: track.seriesId as string } })
            }
            className="ch-mono"
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: 4,
              fontSize: '0.64rem',
              fontWeight: 600,
              color: 'var(--accent)',
              border: 'none',
              background: 'transparent',
              cursor: 'pointer',
              padding: 0,
            }}
          >
            Open series <Icon name="chevron-right" size={12} color="var(--accent)" />
          </button>
        </div>
      )}
    </div>
  );
}

// ── Add-issue popover ──────────────────────────────────────────────────────────────────

function AddIssuePopover({ track, onClose }: { track: TrackerTrack; onClose: () => void }) {
  const [val, setVal] = useState('');
  const add = useAddTrackIssues();
  const target = track.seriesId
    ? { seriesId: track.seriesId }
    : { trackId: track.id.replace(/^track:/, '') };

  const submit = () => {
    const numbers = parseIssueSpec(val);
    if (numbers.length === 0) return;
    add.mutate({ ...target, numbers }, { onSuccess: onClose });
  };

  return (
    <div
      onClick={(e) => e.stopPropagation()}
      style={{
        position: 'absolute',
        zIndex: 70,
        width: 240,
        marginTop: 6,
        right: 0,
        background: 'var(--surface-raised)',
        border: '1px solid var(--border-strong)',
        borderRadius: 'var(--radius-lg)',
        boxShadow: 'var(--shadow-popover)',
        padding: 14,
      }}
    >
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          marginBottom: 10,
        }}
      >
        <span
          className="ch-mono"
          style={{
            fontSize: '0.6rem',
            fontWeight: 600,
            letterSpacing: '0.16em',
            textTransform: 'uppercase',
            color: 'var(--text-tertiary)',
            whiteSpace: 'nowrap',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
          }}
        >
          Add issues
        </span>
        <IconButton icon="x" label="Close" size="sm" onClick={onClose} />
      </div>
      <div style={{ display: 'flex', gap: 8 }}>
        <Input
          placeholder="#24 or 24–52"
          size="sm"
          value={val}
          onChange={(e) => setVal(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') submit();
          }}
        />
        <Button size="sm" variant="secondary" onClick={submit} disabled={add.isPending}>
          Add
        </Button>
      </div>
    </div>
  );
}

// ── Edit standalone track ──────────────────────────────────────────────────────────────

function EditTrackSheet({ track, onClose }: { track: TrackerTrack; onClose: () => void }) {
  const trackId = track.id.replace(/^track:/, '');
  const [name, setName] = useState(track.name);
  const rename = useRenameTrack();
  const del = useDeleteTrack();

  return (
    <div
      style={{
        position: 'absolute',
        inset: 0,
        zIndex: 90,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        padding: 24,
        background: 'color-mix(in oklab, var(--ink-900) 70%, transparent)',
      }}
      onClick={onClose}
    >
      <div
        onClick={(e) => e.stopPropagation()}
        style={{
          width: 380,
          maxWidth: '100%',
          background: 'var(--surface-raised)',
          border: '1px solid var(--border-strong)',
          borderRadius: 'var(--radius-lg)',
          boxShadow: 'var(--shadow-dialog)',
          padding: 20,
        }}
      >
        <h2
          style={{
            margin: 0,
            fontFamily: 'var(--font-body)',
            fontWeight: 600,
            fontSize: '1.05rem',
            color: 'var(--text-primary)',
          }}
        >
          Edit track
        </h2>
        <div style={{ marginTop: 14 }}>
          <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="Series name" />
        </div>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            marginTop: 18,
          }}
        >
          <Button
            variant="danger"
            icon="trash"
            onClick={() => del.mutate(trackId, { onSuccess: onClose })}
            disabled={del.isPending}
          >
            Delete
          </Button>
          <div style={{ display: 'flex', gap: 10 }}>
            <Button variant="ghost" onClick={onClose}>
              Cancel
            </Button>
            <Button
              variant="primary"
              onClick={() =>
                rename.mutate({ id: trackId, name: name.trim() }, { onSuccess: onClose })
              }
              disabled={rename.isPending || !name.trim()}
            >
              Save
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}

// ── Add-series dialog (standalone track) ───────────────────────────────────────────────

function AddSeriesDialog({ onClose }: { onClose: () => void }) {
  const [name, setName] = useState('');
  const [issues, setIssues] = useState('');
  const create = useCreateTrack();
  const add = useAddTrackIssues();

  const submit = async () => {
    const trimmed = name.trim();
    if (!trimmed) return;
    const track = await create.mutateAsync(trimmed);
    const numbers = parseIssueSpec(issues);
    if (numbers.length > 0) {
      await add.mutateAsync({ trackId: track.id, numbers });
    }
    onClose();
  };

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 100,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        padding: 24,
        background: 'color-mix(in oklab, var(--ink-900) 70%, transparent)',
        backdropFilter: 'blur(3px)',
      }}
      onClick={onClose}
    >
      <div
        onClick={(e) => e.stopPropagation()}
        style={{
          width: 460,
          maxWidth: '100%',
          background: 'var(--surface-raised)',
          border: '1px solid var(--border-strong)',
          borderRadius: 'var(--radius-lg)',
          boxShadow: 'var(--shadow-dialog)',
          padding: 22,
        }}
      >
        <h2
          style={{
            margin: 0,
            fontFamily: 'var(--font-body)',
            fontWeight: 600,
            fontSize: '1.1rem',
            color: 'var(--text-primary)',
          }}
        >
          Track a series
        </h2>
        <p style={{ margin: '4px 0 0', fontSize: '0.82rem', color: 'var(--text-tertiary)' }}>
          Follow a series that isn’t in a library. Series in your libraries are tracked
          automatically.
        </p>
        <div style={{ marginTop: 16, display: 'flex', flexDirection: 'column', gap: 10 }}>
          <Input placeholder="Series name" value={name} onChange={(e) => setName(e.target.value)} />
          <Input
            placeholder="Issues — e.g. 1–12 (optional)"
            value={issues}
            onChange={(e) => setIssues(e.target.value)}
          />
        </div>
        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 10, marginTop: 20 }}>
          <Button variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button
            variant="primary"
            onClick={submit}
            disabled={!name.trim() || create.isPending || add.isPending}
          >
            Create
          </Button>
        </div>
      </div>
    </div>
  );
}

// ── helpers ────────────────────────────────────────────────────────────────────────────

/** Parse an issue spec into explicit numbers: "24" → ["24"]; "1-12"/"1–12" → ["1"…"12"];
 *  "1,3,5" → ["1","3","5"]. Only integer ranges expand. */
function parseIssueSpec(input: string): string[] {
  const out: string[] = [];
  for (const raw of input.split(',')) {
    const tok = raw.trim().replace(/^#/, '');
    if (!tok) continue;
    const m = tok.match(/^(\d+)\s*[-–]\s*(\d+)$/);
    if (m && m[1] && m[2]) {
      const lo = parseInt(m[1], 10);
      const hi = parseInt(m[2], 10);
      if (hi >= lo && hi - lo <= 500) {
        for (let n = lo; n <= hi; n++) out.push(String(n));
        continue;
      }
    }
    out.push(tok);
  }
  return out;
}

function Eyebrow({ children, color }: { children: React.ReactNode; color?: string }) {
  return (
    <div
      className="ch-mono"
      style={{
        fontSize: '0.66rem',
        fontWeight: 600,
        letterSpacing: '0.16em',
        textTransform: 'uppercase',
        color: color ?? 'var(--text-tertiary)',
      }}
    >
      {children}
    </div>
  );
}
