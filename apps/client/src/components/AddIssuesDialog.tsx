import { useEffect, useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Button, Input, Icon } from '@comichub/ui';
import type { BookCard, BookHit } from '@comichub/api-client';
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
 */
export function AddIssuesDialog({
  title,
  existingIds,
  onAdd,
  onClose,
}: {
  title: string;
  existingIds: Set<string>;
  onAdd: (bookIds: string[]) => Promise<void>;
  onClose: () => void;
}) {
  const client = useClient();
  const [query, setQuery] = useState('');
  const [debounced, setDebounced] = useState('');
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [expanded, setExpanded] = useState<{
    id: string;
    name: string;
    libraryName?: string;
  } | null>(null);
  const [busy, setBusy] = useState(false);

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

        <Input
          icon="search"
          placeholder="Search series or issues…"
          aria-label="Search the catalog"
          value={query}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => setQuery(e.target.value)}
        />

        <div
          style={{ flex: 1, minHeight: 0, overflowY: 'auto', margin: '0 -4px', padding: '0 4px' }}
        >
          {expanded ? (
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
          <span style={{ fontSize: 'var(--text-small)', color: 'var(--text-secondary)' }}>
            {selected.size} selected
          </span>
          <div style={{ display: 'flex', gap: 10 }}>
            <Button variant="ghost" onClick={onClose} disabled={busy}>
              Cancel
            </Button>
            <Button variant="primary" onClick={submit} disabled={selected.size === 0 || busy}>
              {busy ? 'Adding…' : `Add ${selected.size || ''}`.trim()}
            </Button>
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

function Cover({ client, bookId }: { client: ReturnType<typeof useClient>; bookId?: string }) {
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

function SectionLabel({ children }: { children: React.ReactNode }) {
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

function Hint({ children }: { children: React.ReactNode }) {
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

const rowStyle: React.CSSProperties = {
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

const rowMain: React.CSSProperties = {
  flex: 1,
  minWidth: 0,
  display: 'flex',
  flexDirection: 'column',
};

const rowTitle: React.CSSProperties = {
  fontSize: 'var(--text-small)',
  whiteSpace: 'nowrap',
  overflow: 'hidden',
  textOverflow: 'ellipsis',
};

const rowSub: React.CSSProperties = {
  fontSize: 'var(--text-label)',
  color: 'var(--text-tertiary)',
  whiteSpace: 'nowrap',
  overflow: 'hidden',
  textOverflow: 'ellipsis',
};
