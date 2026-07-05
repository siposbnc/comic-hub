import { useEffect, useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Button, Badge, Input, Icon, IconButton } from '@comichub/ui';
import type { BookCard, BookHit, ManualListEntry } from '@comichub/api-client';
import { useClient } from '../lib/client.js';
import { issueLabel } from '../lib/format.js';

/** A pickable issue, normalized from either a search hit or a series' issue list. */
interface Pick {
  id: string;
  seriesName?: string;
  number?: string;
  title?: string;
  /** Shown on search hits so same-named series in different libraries are tellable apart. */
  libraryName?: string;
}

function fromHit(b: BookHit): Pick {
  return {
    id: b.id,
    seriesName: b.seriesName,
    number: b.number,
    title: b.title,
    libraryName: b.libraryName,
  };
}
function fromCard(b: BookCard, seriesName: string): Pick {
  return { id: b.id, seriesName, number: b.number, title: b.title };
}
function pickLabel(p: Pick): string {
  return p.title || [p.seriesName, issueLabel(p.number)].filter(Boolean).join(' ') || 'Untitled';
}

/**
 * Search-and-multi-select dialog for adding issues to a collection or reading list. Search
 * surfaces series and titled issues; expanding a series lets you pick from its run (with a
 * Select-all). The selection accumulates across searches; issues already in the list show
 * as added and aren't selectable.
 *
 * When `onAddManual` is given (reading lists), a second "Add missing" tab jots want-list
 * placeholders for issues not in the library — they hold a slot in the reading order and
 * link themselves to a real file once it appears (per the design preview).
 */
export function AddIssuesDialog({
  title,
  subtitle,
  existingIds,
  onAdd,
  onAddManual,
  onClose,
}: {
  title: string;
  subtitle?: string;
  existingIds: Set<string>;
  onAdd: (bookIds: string[]) => Promise<void>;
  onAddManual?: (entries: ManualListEntry[]) => Promise<void>;
  onClose: () => void;
}) {
  const client = useClient();
  const [tab, setTab] = useState<'search' | 'missing'>('search');
  const [query, setQuery] = useState('');
  const [debounced, setDebounced] = useState('');
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [expanded, setExpanded] = useState<{
    id: string;
    name: string;
    libraryName?: string;
  } | null>(null);
  const [busy, setBusy] = useState(false);

  // "Add missing" jot form + accumulated want-list lines.
  const [mSeries, setMSeries] = useState('');
  const [mNumber, setMNumber] = useState('');
  const [mTitle, setMTitle] = useState('');
  const [jotted, setJotted] = useState<ManualListEntry[]>([]);
  const canJot = mSeries.trim() !== '' || mTitle.trim() !== '';
  const jotLine = () => {
    if (!canJot) return;
    setJotted((j) => [
      ...j,
      { seriesName: mSeries.trim(), number: mNumber.trim(), title: mTitle.trim() },
    ]);
    setMNumber('');
    setMTitle('');
  };

  useEffect(() => {
    const t = setTimeout(() => setDebounced(query.trim()), 200);
    return () => clearTimeout(t);
  }, [query]);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !busy) onClose();
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [busy, onClose]);

  const search = useQuery({
    queryKey: ['issuePicker', debounced],
    queryFn: () => client.search(debounced, { limit: 12 }),
    enabled: debounced.length >= 2,
  });

  const seriesDetail = useQuery({
    queryKey: ['seriesDetail', expanded?.id],
    queryFn: () => client.seriesDetail(expanded!.id),
    enabled: !!expanded,
  });

  const toggle = (id: string) => {
    if (existingIds.has(id)) return;
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const addMany = (ids: string[]) =>
    setSelected((prev) => {
      const next = new Set(prev);
      for (const id of ids) if (!existingIds.has(id)) next.add(id);
      return next;
    });

  const expandedPicks: Pick[] = useMemo(
    () =>
      seriesDetail.data
        ? seriesDetail.data.books.map((b) => fromCard(b, seriesDetail.data!.name))
        : [],
    [seriesDetail.data],
  );

  const submit = async () => {
    if (selected.size === 0 || busy) return;
    setBusy(true);
    try {
      await onAdd([...selected]);
      onClose();
    } catch {
      setBusy(false); // surfaced by the caller's toast; keep the dialog open
    }
  };

  const submitManual = async () => {
    if (jotted.length === 0 || busy || !onAddManual) return;
    setBusy(true);
    try {
      await onAddManual(jotted);
      onClose();
    } catch {
      setBusy(false);
    }
  };

  return (
    <div
      role="presentation"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget && !busy) onClose();
      }}
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 1100,
        display: 'grid',
        placeItems: 'center',
        background: 'color-mix(in oklab, var(--ink-900) 70%, transparent)',
        padding: 24,
      }}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-label={title}
        style={{
          width: 'min(560px, 100%)',
          height: 'min(680px, 90vh)',
          display: 'flex',
          flexDirection: 'column',
          background: 'var(--surface-raised)',
          border: '1px solid var(--border-strong)',
          borderRadius: 'var(--radius-lg)',
          boxShadow: 'var(--shadow-popover)',
          padding: 20,
          gap: 12,
        }}
      >
        <div style={{ display: 'flex', alignItems: 'flex-start', gap: 10 }}>
          <div style={{ flex: 1, minWidth: 0 }}>
            <h2
              style={{
                margin: 0,
                fontFamily: 'var(--font-display)',
                fontSize: 'var(--text-heading)',
                fontWeight: 700,
                color: 'var(--text-primary)',
              }}
            >
              {title}
            </h2>
            {subtitle && (
              <div
                className="ch-mono"
                style={{ fontSize: '0.66rem', color: 'var(--text-tertiary)', marginTop: 3 }}
              >
                {subtitle}
              </div>
            )}
          </div>
          <IconButton icon="x" label="Cancel" variant="ghost" size="sm" onClick={onClose} />
        </div>

        {onAddManual && (
          <div
            style={{
              display: 'flex',
              gap: 22,
              borderBottom: '1px solid var(--border-hairline)',
            }}
          >
            {(
              [
                ['search', 'Search library'],
                ['missing', 'Add missing'],
              ] as const
            ).map(([id, label]) => (
              <button
                key={id}
                type="button"
                onClick={() => setTab(id)}
                style={{
                  position: 'relative',
                  padding: '0 4px 12px',
                  background: 'none',
                  border: 'none',
                  cursor: 'pointer',
                  fontFamily: 'var(--font-body)',
                  fontSize: '0.86rem',
                  fontWeight: 600,
                  color: tab === id ? 'var(--text-primary)' : 'var(--text-tertiary)',
                }}
              >
                {label}
                <span
                  style={{
                    position: 'absolute',
                    left: 0,
                    right: 0,
                    bottom: -1,
                    height: 2,
                    borderRadius: 2,
                    background: tab === id ? 'var(--accent)' : 'transparent',
                  }}
                />
              </button>
            ))}
          </div>
        )}

        {tab === 'search' && (
          <Input
            icon="search"
            placeholder="Search series or issues…"
            aria-label="Search the catalog"
            value={query}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) => setQuery(e.target.value)}
          />
        )}

        <div
          style={{ flex: 1, minHeight: 0, overflowY: 'auto', margin: '0 -4px', padding: '0 4px' }}
        >
          {tab === 'missing' ? (
            <ManualTab
              mSeries={mSeries}
              mNumber={mNumber}
              mTitle={mTitle}
              setMSeries={setMSeries}
              setMNumber={setMNumber}
              setMTitle={setMTitle}
              canJot={canJot}
              onJot={jotLine}
              jotted={jotted}
              onRemove={(i) => setJotted((arr) => arr.filter((_, k) => k !== i))}
            />
          ) : expanded ? (
            <ExpandedSeries
              name={expanded.name}
              libraryName={expanded.libraryName}
              picks={expandedPicks}
              loading={seriesDetail.isLoading}
              selected={selected}
              existingIds={existingIds}
              client={client}
              onBack={() => setExpanded(null)}
              onToggle={toggle}
              onSelectAll={() => addMany(expandedPicks.map((p) => p.id))}
            />
          ) : debounced.length < 2 ? (
            <Hint>Type at least two characters to search.</Hint>
          ) : search.isLoading ? (
            <Hint>Searching…</Hint>
          ) : !search.data ||
            (search.data.series.length === 0 && search.data.books.length === 0) ? (
            <Hint>No matches for “{debounced}”.</Hint>
          ) : (
            <>
              {search.data.series.length > 0 && <SectionLabel>Series</SectionLabel>}
              {search.data.series.map((s) => (
                <button
                  key={s.id}
                  type="button"
                  onClick={() =>
                    setExpanded({ id: s.id, name: s.name, libraryName: s.libraryName })
                  }
                  style={rowStyle}
                >
                  <Cover client={client} bookId={s.coverBookId} />
                  <span style={rowMain}>
                    <span style={rowTitle}>{s.name}</span>
                    <span style={rowSub}>
                      {['Series', s.year, s.libraryName].filter(Boolean).join(' · ')}
                    </span>
                  </span>
                  <Icon name="chevron-right" size={16} color="var(--text-tertiary)" />
                </button>
              ))}

              {search.data.books.length > 0 && <SectionLabel>Issues</SectionLabel>}
              {search.data.books.map((b) => (
                <IssueRow
                  key={b.id}
                  pick={fromHit(b)}
                  client={client}
                  checked={selected.has(b.id)}
                  added={existingIds.has(b.id)}
                  onToggle={toggle}
                />
              ))}
            </>
          )}
        </div>

        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            gap: 10,
            borderTop: '1px solid var(--border-hairline)',
            paddingTop: 12,
          }}
        >
          {tab === 'missing' ? (
            <Badge tone="neutral" mono>
              {jotted.length} placeholder{jotted.length === 1 ? '' : 's'}
            </Badge>
          ) : (
            <span style={{ fontSize: 'var(--text-small)', color: 'var(--text-secondary)' }}>
              {selected.size} selected
            </span>
          )}
          <div style={{ display: 'flex', gap: 10 }}>
            <Button variant="ghost" onClick={onClose} disabled={busy}>
              Cancel
            </Button>
            {tab === 'missing' ? (
              <Button
                variant="primary"
                icon="plus"
                onClick={submitManual}
                disabled={jotted.length === 0 || busy}
              >
                {busy ? 'Adding…' : 'Add to list'}
              </Button>
            ) : (
              <Button variant="primary" onClick={submit} disabled={selected.size === 0 || busy}>
                {busy ? 'Adding…' : `Add ${selected.size || ''}`.trim()}
              </Button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function ExpandedSeries({
  name,
  libraryName,
  picks,
  loading,
  selected,
  existingIds,
  client,
  onBack,
  onToggle,
  onSelectAll,
}: {
  name: string;
  libraryName?: string;
  picks: Pick[];
  loading: boolean;
  selected: Set<string>;
  existingIds: Set<string>;
  client: ReturnType<typeof useClient>;
  onBack: () => void;
  onToggle: (id: string) => void;
  onSelectAll: () => void;
}) {
  return (
    <>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6 }}>
        <button
          type="button"
          onClick={onBack}
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: 4,
            background: 'none',
            border: 'none',
            cursor: 'pointer',
            color: 'var(--text-secondary)',
            fontSize: 'var(--text-small)',
          }}
        >
          <Icon name="chevron-left" size={15} /> Back
        </button>
        <span style={{ ...rowTitle, flex: 1, minWidth: 0 }}>
          {name}
          {libraryName && <span style={{ color: 'var(--text-tertiary)' }}> · {libraryName}</span>}
        </span>
        <Button variant="ghost" size="sm" onClick={onSelectAll} disabled={picks.length === 0}>
          Select all
        </Button>
      </div>
      {loading ? (
        <Hint>Loading issues…</Hint>
      ) : (
        picks.map((p) => (
          <IssueRow
            key={p.id}
            pick={p}
            client={client}
            checked={selected.has(p.id)}
            added={existingIds.has(p.id)}
            onToggle={onToggle}
          />
        ))
      )}
    </>
  );
}

function IssueRow({
  pick,
  client,
  checked,
  added,
  onToggle,
}: {
  pick: Pick;
  client: ReturnType<typeof useClient>;
  checked: boolean;
  added: boolean;
  onToggle: (id: string) => void;
}) {
  return (
    <button
      type="button"
      disabled={added}
      onClick={() => onToggle(pick.id)}
      style={{ ...rowStyle, opacity: added ? 0.55 : 1, cursor: added ? 'default' : 'pointer' }}
    >
      <Cover client={client} bookId={pick.id} />
      <span style={rowMain}>
        <span style={rowTitle}>{pickLabel(pick)}</span>
        <span style={rowSub}>
          {[pick.seriesName, issueLabel(pick.number), pick.libraryName].filter(Boolean).join(' · ')}
        </span>
      </span>
      {added ? (
        <span style={{ fontSize: 'var(--text-label)', color: 'var(--text-tertiary)' }}>Added</span>
      ) : (
        <span
          aria-hidden
          style={{
            width: 20,
            height: 20,
            borderRadius: 6,
            border: `1px solid ${checked ? 'var(--accent)' : 'var(--border-strong)'}`,
            background: checked ? 'var(--accent)' : 'transparent',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}
        >
          {checked && <Icon name="check" size={13} color="var(--text-on-accent)" />}
        </span>
      )}
    </button>
  );
}

export function Cover({
  client,
  bookId,
}: {
  client: ReturnType<typeof useClient>;
  bookId?: string;
}) {
  return (
    <span
      style={{
        flex: 'none',
        width: 30,
        height: 45,
        borderRadius: 'var(--radius-sm)',
        overflow: 'hidden',
        background: 'var(--surface-cover)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
      }}
    >
      {bookId ? (
        <img
          src={client.coverUrl(bookId, 80)}
          alt=""
          width={30}
          height={45}
          style={{ objectFit: 'cover' }}
        />
      ) : (
        <Icon name="book" size={14} />
      )}
    </span>
  );
}

/** The "Add missing" tab: jot a want-list placeholder for an issue not owned yet. */
function ManualTab({
  mSeries,
  mNumber,
  mTitle,
  setMSeries,
  setMNumber,
  setMTitle,
  canJot,
  onJot,
  jotted,
  onRemove,
}: {
  mSeries: string;
  mNumber: string;
  mTitle: string;
  setMSeries: (v: string) => void;
  setMNumber: (v: string) => void;
  setMTitle: (v: string) => void;
  canJot: boolean;
  onJot: () => void;
  jotted: ManualListEntry[];
  onRemove: (index: number) => void;
}) {
  const field = (
    label: string,
    value: string,
    set: (v: string) => void,
    placeholder: string,
    flex: number,
  ) => (
    <label style={{ flex, minWidth: 0, display: 'flex', flexDirection: 'column', gap: 6 }}>
      <span className="ch-label" style={{ color: 'var(--text-tertiary)' }}>
        {label}
      </span>
      <Input
        value={value}
        placeholder={placeholder}
        onChange={(e: React.ChangeEvent<HTMLInputElement>) => set(e.target.value)}
        onKeyDown={(e: React.KeyboardEvent) => {
          if (e.key === 'Enter') onJot();
        }}
      />
    </label>
  );

  return (
    <div style={{ padding: '4px 2px' }}>
      <p
        style={{
          margin: '0 0 16px',
          fontSize: '0.82rem',
          color: 'var(--text-secondary)',
          lineHeight: 1.5,
        }}
      >
        Jot an issue you don’t own yet — it holds a slot in the reading order and links itself to a
        real file once you add it. A series name or a title is enough.
      </p>
      <div style={{ display: 'flex', gap: 10, alignItems: 'flex-end' }}>
        {field('Series', mSeries, setMSeries, 'e.g. Wonder Woman', 1.4)}
        {field('Number', mNumber, setMNumber, '#001', 0.6)}
        {field('Title', mTitle, setMTitle, 'optional', 1.4)}
        <Button variant="secondary" icon="plus" onClick={onJot} disabled={!canJot}>
          Add
        </Button>
      </div>
      <div
        className="ch-mono"
        style={{ fontSize: '0.62rem', color: 'var(--text-tertiary)', marginTop: 8 }}
      >
        At least a series name or a title.
      </div>

      {jotted.length > 0 && (
        <div style={{ marginTop: 22 }}>
          <div
            className="ch-mono"
            style={{
              fontSize: '0.62rem',
              fontWeight: 600,
              letterSpacing: '0.16em',
              textTransform: 'uppercase',
              color: 'var(--paper-600)',
              marginBottom: 10,
            }}
          >
            Want-list · {jotted.length}
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
            {jotted.map((j, i) => (
              <div
                key={i}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 12,
                  padding: '9px 12px',
                  background: 'var(--surface-card)',
                  border: '1px solid var(--border-hairline)',
                  borderRadius: 6,
                }}
              >
                <span
                  style={{
                    flex: 'none',
                    width: 30,
                    height: 45,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    border: '1px dashed var(--border-strong)',
                    borderRadius: 3,
                  }}
                >
                  <Icon name="book-open" size={13} color="var(--paper-600)" />
                </span>
                <span style={{ flex: 1, minWidth: 0 }}>
                  <span
                    style={{
                      display: 'block',
                      fontSize: '0.86rem',
                      fontWeight: 600,
                      color: 'var(--paper-100)',
                      whiteSpace: 'nowrap',
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                    }}
                  >
                    {[j.seriesName || j.title, j.number].filter(Boolean).join(' ')}
                  </span>
                  {j.title && j.seriesName && (
                    <span
                      className="ch-mono"
                      style={{
                        display: 'block',
                        fontSize: '0.64rem',
                        color: 'var(--text-tertiary)',
                        marginTop: 2,
                      }}
                    >
                      {j.title}
                    </span>
                  )}
                </span>
                <MissingPill />
                <IconButton
                  icon="x"
                  label="Remove placeholder"
                  variant="ghost"
                  size="sm"
                  onClick={() => onRemove(i)}
                />
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

/** The quiet warning pill marking a placeholder ("missing") entry. */
export function MissingPill() {
  return (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 5,
        height: 19,
        padding: '0 8px',
        borderRadius: 'var(--radius-pill)',
        background: 'color-mix(in oklab, var(--warning) 16%, transparent)',
        color: 'var(--warning)',
        fontFamily: 'var(--font-mono)',
        fontSize: '0.58rem',
        letterSpacing: '0.06em',
        textTransform: 'uppercase',
      }}
    >
      <Icon name="alert-triangle" size={11} color="var(--warning)" /> Missing
    </span>
  );
}

export function SectionLabel({ children }: { children: React.ReactNode }) {
  return (
    <div
      className="ch-mono"
      style={{
        fontSize: 'var(--text-label)',
        textTransform: 'uppercase',
        letterSpacing: 'var(--tracking-label)',
        color: 'var(--text-tertiary)',
        padding: 'var(--space-2) var(--space-1) var(--space-1)',
      }}
    >
      {children}
    </div>
  );
}

export function Hint({ children }: { children: React.ReactNode }) {
  return (
    <div
      style={{
        padding: 'var(--space-4)',
        fontSize: 'var(--text-small)',
        color: 'var(--text-tertiary)',
      }}
    >
      {children}
    </div>
  );
}

export const rowStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  gap: 10,
  width: '100%',
  padding: '6px 8px',
  textAlign: 'left',
  background: 'transparent',
  border: 'none',
  borderRadius: 'var(--radius-sm)',
  color: 'var(--text-primary)',
};

export const rowMain: React.CSSProperties = {
  flex: 1,
  minWidth: 0,
  display: 'flex',
  flexDirection: 'column',
};

export const rowTitle: React.CSSProperties = {
  fontSize: 'var(--text-small)',
  whiteSpace: 'nowrap',
  overflow: 'hidden',
  textOverflow: 'ellipsis',
};

export const rowSub: React.CSSProperties = {
  fontSize: 'var(--text-label)',
  color: 'var(--text-tertiary)',
  whiteSpace: 'nowrap',
  overflow: 'hidden',
  textOverflow: 'ellipsis',
};
