import { useEffect, useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Button, Input, Icon, IconButton } from '@comichub/ui';
import type { ReadingListEntry } from '@comichub/api-client';
import { useClient } from '../lib/client.js';
import { issueLabel } from '../lib/format.js';
import {
  Cover,
  Hint,
  SectionLabel,
  rowStyle,
  rowMain,
  rowTitle,
  rowSub,
} from './AddIssuesDialog.js';

/**
 * Single-select picker that points a stale reading-list placeholder at a real issue
 * (per the design preview). Search is seeded with the placeholder's series name; series
 * results expand to their run so untitled issues are reachable. Issues already in the
 * list show as taken and can't be linked twice.
 */
export function LinkIssueDialog({
  entry,
  existingIds,
  onLink,
  onClose,
}: {
  entry: ReadingListEntry;
  existingIds: Set<string>;
  onLink: (bookId: string) => Promise<void>;
  onClose: () => void;
}) {
  const client = useClient();
  const seed = entry.seriesName ?? entry.title ?? '';
  const [query, setQuery] = useState(seed);
  const [debounced, setDebounced] = useState(seed.trim());
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
    queryKey: ['linkPicker', debounced],
    queryFn: () => client.search(debounced, { limit: 12 }),
    enabled: debounced.length >= 2,
  });

  const seriesDetail = useQuery({
    queryKey: ['seriesDetail', expanded?.id],
    queryFn: () => client.seriesDetail(expanded!.id),
    enabled: !!expanded,
  });

  const link = async (bookId: string) => {
    if (busy) return;
    setBusy(true);
    try {
      await onLink(bookId);
      onClose();
    } catch {
      setBusy(false); // surfaced by the caller's toast; keep the dialog open
    }
  };

  const label =
    [entry.seriesName, issueLabel(entry.number)].filter(Boolean).join(' ') ||
    entry.title ||
    'placeholder';

  const expandedBooks = useMemo(() => seriesDetail.data?.books ?? [], [seriesDetail.data]);

  const issueRow = (bookId: string, main: string, sub: string) => {
    const taken = existingIds.has(bookId);
    return (
      <div key={bookId} style={{ ...rowStyle, opacity: taken ? 0.55 : 1 }}>
        <Cover client={client} bookId={bookId} />
        <span style={rowMain}>
          <span style={rowTitle}>{main}</span>
          <span style={{ ...rowSub, ...(taken ? { color: 'var(--warning)' } : null) }}>
            {taken ? 'Already in this list' : sub}
          </span>
        </span>
        <Button
          size="sm"
          variant={taken ? 'ghost' : 'primary'}
          icon={taken ? undefined : 'link'}
          disabled={taken || busy}
          onClick={() => link(bookId)}
        >
          {taken ? 'In list' : 'Link'}
        </Button>
      </div>
    );
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
        aria-label="Link a real issue"
        style={{
          width: 'min(540px, 100%)',
          height: 'min(640px, 90vh)',
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
              Link a real issue
            </h2>
            <div
              className="ch-mono"
              style={{ fontSize: '0.66rem', color: 'var(--text-tertiary)', marginTop: 3 }}
            >
              Placeholder · {label}
            </div>
          </div>
          <IconButton icon="x" label="Cancel" variant="ghost" size="sm" onClick={onClose} />
        </div>

        <Input
          icon="search"
          placeholder="Search your library by series or number…"
          aria-label="Search the catalog"
          value={query}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => setQuery(e.target.value)}
        />
        <div className="ch-mono" style={{ fontSize: '0.66rem', color: 'var(--text-tertiary)' }}>
          {search.isLoading ? (
            <span style={{ color: 'var(--accent)' }}>Searching your library…</span>
          ) : (
            'Pick the issue this placeholder should point to'
          )}
        </div>

        <div
          style={{ flex: 1, minHeight: 0, overflowY: 'auto', margin: '0 -4px', padding: '0 4px' }}
        >
          {expanded ? (
            <>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6 }}>
                <button
                  type="button"
                  onClick={() => setExpanded(null)}
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
                  {expanded.name}
                  {expanded.libraryName && (
                    <span style={{ color: 'var(--text-tertiary)' }}> · {expanded.libraryName}</span>
                  )}
                </span>
              </div>
              {seriesDetail.isLoading ? (
                <Hint>Loading issues…</Hint>
              ) : (
                expandedBooks.map((b) =>
                  issueRow(
                    b.id,
                    b.title || [expanded.name, issueLabel(b.number)].filter(Boolean).join(' '),
                    [expanded.name, issueLabel(b.number)].filter(Boolean).join(' · '),
                  ),
                )
              )}
            </>
          ) : debounced.length < 2 ? (
            <Hint>Type at least two characters to search.</Hint>
          ) : search.isLoading ? (
            <Hint>Searching…</Hint>
          ) : !search.data ||
            (search.data.series.length === 0 && search.data.books.length === 0) ? (
            <div
              style={{
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                textAlign: 'center',
                padding: '40px 24px',
              }}
            >
              <div style={{ fontWeight: 600, fontSize: '0.95rem', color: 'var(--text-primary)' }}>
                Nothing in your library for “{debounced}”
              </div>
              <p
                style={{
                  margin: '8px 0 0',
                  fontSize: '0.82rem',
                  color: 'var(--text-secondary)',
                  maxWidth: 320,
                  lineHeight: 1.5,
                }}
              >
                The placeholder keeps its slot — link it whenever the issue shows up.
              </p>
            </div>
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
              {search.data.books.map((b) =>
                issueRow(
                  b.id,
                  b.title || [b.seriesName, issueLabel(b.number)].filter(Boolean).join(' '),
                  [b.seriesName, issueLabel(b.number), b.libraryName].filter(Boolean).join(' · '),
                ),
              )}
            </>
          )}
        </div>

        <div
          style={{
            display: 'flex',
            justifyContent: 'flex-end',
            gap: 10,
            borderTop: '1px solid var(--border-hairline)',
            paddingTop: 12,
          }}
        >
          <Button variant="ghost" onClick={onClose} disabled={busy}>
            Cancel
          </Button>
        </div>
      </div>
    </div>
  );
}
