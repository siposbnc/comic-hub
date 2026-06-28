import { useEffect } from 'react';
import { useNavigate } from '@tanstack/react-router';
import { Button, Icon } from '@comichub/ui';
import type { HealthItem } from '@comichub/api-client';
import { useLibraryHealth } from '../lib/queries.js';
import { LoadingState, ErrorState } from './Page.js';
import { issueLabel } from '../lib/format.js';

/** Modal: a library's health report (corrupt / orphaned / unmatched / duplicates). */
export function HealthDialog({ libraryId, onClose }: { libraryId: string; onClose: () => void }) {
  const navigate = useNavigate();
  const health = useLibraryHealth(libraryId);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [onClose]);

  const open = (bookId: string) => {
    navigate({ to: '/book/$id', params: { id: bookId } });
    onClose();
  };

  const data = health.data;

  return (
    <div
      role="presentation"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) onClose();
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
        aria-label="Library health"
        style={{
          width: 'min(560px, 100%)',
          maxHeight: '80vh',
          overflowY: 'auto',
          background: 'var(--surface-raised)',
          border: '1px solid var(--border-strong)',
          borderRadius: 'var(--radius-lg)',
          boxShadow: 'var(--shadow-popover)',
          padding: 22,
          display: 'flex',
          flexDirection: 'column',
          gap: 16,
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
          Library health
        </h2>

        {health.isLoading ? (
          <LoadingState />
        ) : health.isError || !data ? (
          <ErrorState
            message={
              health.error instanceof Error ? health.error.message : 'Could not load health.'
            }
            onRetry={() => health.refetch()}
          />
        ) : (
          <>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 10 }}>
              <Stat label="Issues" value={data.counts.books} />
              <Stat label="Corrupt" value={data.counts.corrupt} tone="danger" />
              <Stat label="Orphaned" value={data.counts.orphans} tone="danger" />
              <Stat label="Unmatched" value={data.counts.unmatched} />
              <Stat label="Duplicate sets" value={data.counts.duplicateGroups} />
            </div>

            {data.counts.corrupt +
              data.counts.orphans +
              data.counts.unmatched +
              data.counts.duplicateGroups ===
            0 ? (
              <p
                style={{ margin: 0, color: 'var(--text-tertiary)', fontSize: 'var(--text-small)' }}
              >
                Everything looks healthy. ✨
              </p>
            ) : (
              <>
                <Section title="Corrupt" items={data.corrupt} onOpen={open} />
                <Section title="Orphaned (file missing)" items={data.orphans} onOpen={open} />
                <Section title="Unmatched (no metadata)" items={data.unmatched} onOpen={open} />
                {data.duplicates.length > 0 && (
                  <div>
                    <SectionHeading>Duplicate files</SectionHeading>
                    {data.duplicates.map((g) => (
                      <div key={g.contentHash} style={{ marginBottom: 8 }}>
                        {g.books.map((b) => (
                          <ItemRow key={b.id} item={b} onOpen={open} />
                        ))}
                      </div>
                    ))}
                  </div>
                )}
              </>
            )}
          </>
        )}

        <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
          <Button type="button" variant="ghost" onClick={onClose}>
            Close
          </Button>
        </div>
      </div>
    </div>
  );
}

function Stat({ label, value, tone }: { label: string; value: number; tone?: 'danger' }) {
  const danger = tone === 'danger' && value > 0;
  return (
    <div
      style={{
        flex: '1 0 90px',
        padding: '10px 12px',
        background: 'var(--surface-card)',
        border: '1px solid var(--border-hairline)',
        borderRadius: 'var(--radius-md)',
      }}
    >
      <div
        style={{
          fontFamily: 'var(--font-display)',
          fontSize: 'var(--text-title)',
          fontWeight: 800,
          color: danger ? 'var(--danger)' : 'var(--text-primary)',
        }}
      >
        {value}
      </div>
      <div
        className="ch-mono"
        style={{
          fontSize: 'var(--text-label)',
          textTransform: 'uppercase',
          letterSpacing: 'var(--tracking-label)',
          color: 'var(--text-tertiary)',
        }}
      >
        {label}
      </div>
    </div>
  );
}

function Section({
  title,
  items,
  onOpen,
}: {
  title: string;
  items: HealthItem[];
  onOpen: (bookId: string) => void;
}) {
  if (items.length === 0) return null;
  return (
    <div>
      <SectionHeading>{title}</SectionHeading>
      {items.map((it) => (
        <ItemRow key={it.id} item={it} onOpen={onOpen} />
      ))}
    </div>
  );
}

function SectionHeading({ children }: { children: React.ReactNode }) {
  return (
    <div
      className="ch-mono"
      style={{
        fontSize: 'var(--text-label)',
        textTransform: 'uppercase',
        letterSpacing: 'var(--tracking-label)',
        color: 'var(--text-secondary)',
        margin: '4px 0 6px',
      }}
    >
      {children}
    </div>
  );
}

function ItemRow({ item, onOpen }: { item: HealthItem; onOpen: (bookId: string) => void }) {
  return (
    <button
      type="button"
      onClick={() => onOpen(item.id)}
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 8,
        width: '100%',
        padding: '6px 8px',
        textAlign: 'left',
        background: 'transparent',
        border: 'none',
        borderRadius: 'var(--radius-sm)',
        cursor: 'pointer',
        color: 'var(--text-secondary)',
        fontSize: 'var(--text-small)',
      }}
    >
      <Icon name="book" size={13} color="var(--text-tertiary)" />
      <span
        style={{
          flex: 1,
          minWidth: 0,
          whiteSpace: 'nowrap',
          overflow: 'hidden',
          textOverflow: 'ellipsis',
        }}
      >
        {item.title || issueLabel(item.number) || item.path}
      </span>
    </button>
  );
}
