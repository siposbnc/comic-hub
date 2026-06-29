import { useState } from 'react';
import { useQuery, useMutation } from '@tanstack/react-query';
import { Dialog, Button, Badge, Icon } from '@comichub/ui';
import type { BadgeProps } from '@comichub/ui';
import type { SeriesMatchCandidate } from '@comichub/api-client';
import { useClient } from '../lib/client.js';
import { useUiStore } from '../store/ui.js';
import { LoadingState, ErrorState } from './Page.js';

/**
 * Candidate picker for online metadata matching: lists ranked provider candidates for a
 * series and, on pick, kicks off the batch match job (its progress shows in the top-bar
 * JobIndicator; the WS jobs topic refreshes the screen when it lands).
 */
export function MatchDialog({
  seriesId,
  seriesName,
  onClose,
}: {
  seriesId: string;
  seriesName: string;
  onClose: () => void;
}) {
  const client = useClient();
  const addToast = useUiStore((s) => s.addToast);

  // The query actually run against the provider. Starts blank so the candidates query uses
  // the series' own name (server default); a manual re-search sets a custom name.
  const [draft, setDraft] = useState(seriesName);
  const [active, setActive] = useState('');

  const providers = useQuery({ queryKey: ['providers'], queryFn: () => client.providers() });
  const configured = providers.data?.some((p) => p.configured) ?? false;

  const candidates = useQuery({
    queryKey: ['matchCandidates', seriesId, active],
    queryFn: () => client.seriesMatchCandidates(seriesId, active ? { query: active } : {}),
    enabled: providers.isSuccess && configured,
    retry: false,
  });

  const search = () => {
    const q = draft.trim();
    setActive(q);
  };
  const activeLabel = active || seriesName;

  const apply = useMutation({
    mutationFn: (providerId: string) => client.applySeriesMatch(seriesId, providerId),
    onSuccess: () => {
      addToast({
        tone: 'info',
        title: 'Matching metadata',
        message: `Updating ${seriesName} from Comic Vine…`,
      });
      onClose();
    },
    onError: (e) =>
      addToast({
        tone: 'danger',
        title: 'Match failed',
        message: e instanceof Error ? e.message : 'Could not start the match.',
      }),
  });

  return (
    <Dialog
      title="Match metadata"
      width={560}
      onClose={onClose}
      footer={
        <Button variant="ghost" onClick={onClose}>
          Cancel
        </Button>
      }
    >
      {providers.isLoading ? (
        <LoadingState />
      ) : !configured ? (
        <p style={{ margin: 0, color: 'var(--text-secondary)', lineHeight: 'var(--leading-body)' }}>
          No metadata provider is configured. Set <span className="ch-mono">COMICVINE_API_KEY</span>{' '}
          on the server to match against Comic Vine, then reload.
        </p>
      ) : (
        <>
          {/* Re-search by a custom name when the auto-derived query is wrong. */}
          <div style={{ display: 'flex', gap: 8, marginBottom: 8 }}>
            <div
              style={{
                flex: 1,
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                height: 38,
                padding: '0 12px',
                background: 'var(--surface-card)',
                border: '1px solid var(--border-hairline)',
                borderRadius: 'var(--radius-md)',
              }}
            >
              <Icon name="search" size={16} color="var(--text-tertiary)" />
              <input
                value={draft}
                onChange={(e) => setDraft(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') search();
                }}
                placeholder="Search Comic Vine by name…"
                style={{
                  flex: 1,
                  minWidth: 0,
                  height: '100%',
                  background: 'transparent',
                  border: 'none',
                  outline: 'none',
                  color: 'var(--text-primary)',
                  fontFamily: 'var(--font-body)',
                  fontSize: 'var(--text-body)',
                }}
              />
            </div>
            <Button icon="search" onClick={search}>
              Search
            </Button>
          </div>
          <div
            className="ch-mono"
            style={{ fontSize: '0.66rem', color: 'var(--text-tertiary)', marginBottom: 12 }}
          >
            {candidates.isFetching ? (
              <span style={{ color: 'var(--accent)' }}>
                Searching Comic Vine for “{activeLabel}”…
              </span>
            ) : (
              <span>Showing results for “{activeLabel}”</span>
            )}
          </div>

          {candidates.isLoading ? (
            <LoadingState label="Searching Comic Vine…" />
          ) : candidates.isError ? (
            <ErrorState
              message={
                candidates.error instanceof Error ? candidates.error.message : 'Search failed.'
              }
              onRetry={() => candidates.refetch()}
            />
          ) : (candidates.data?.length ?? 0) === 0 ? (
            <p style={{ margin: 0, color: 'var(--text-tertiary)' }}>
              No candidates found for “{activeLabel}”. Try a different name — drop the volume year,
              or use the publisher’s title.
            </p>
          ) : (
            <div
              style={{
                display: 'flex',
                flexDirection: 'column',
                gap: 8,
                maxHeight: 420,
                overflowY: 'auto',
              }}
            >
              {candidates.data!.map((c) => (
                <CandidateRow
                  key={c.providerId}
                  candidate={c}
                  busy={apply.isPending}
                  onUse={() => apply.mutate(c.providerId)}
                />
              ))}
            </div>
          )}
        </>
      )}
    </Dialog>
  );
}

function CandidateRow({
  candidate: c,
  busy,
  onUse,
}: {
  candidate: SeriesMatchCandidate;
  busy: boolean;
  onUse: () => void;
}) {
  const pct = Math.round(c.score * 100);
  const tone: BadgeProps['tone'] =
    c.score >= 0.8 ? 'success' : c.score >= 0.5 ? 'accent' : 'neutral';
  const meta = [
    c.publisher,
    c.year || undefined,
    c.issueCount ? `${c.issueCount} issues` : undefined,
  ]
    .filter(Boolean)
    .join(' · ');

  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 12,
        padding: 8,
        borderRadius: 'var(--radius-md)',
        background: 'var(--surface-card)',
        border: '1px solid var(--border-hairline)',
      }}
    >
      <div
        style={{
          width: 40,
          height: 60,
          flex: 'none',
          background: 'var(--surface-cover)',
          backgroundImage: c.coverUrl ? `url(${c.coverUrl})` : undefined,
          backgroundSize: 'cover',
          backgroundPosition: 'center',
          borderRadius: 'var(--radius-sm)',
        }}
      />
      <div style={{ flex: 1, minWidth: 0 }}>
        <div
          style={{
            fontWeight: 600,
            color: 'var(--text-primary)',
            whiteSpace: 'nowrap',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
          }}
        >
          {c.name}
        </div>
        <div
          className="ch-mono"
          style={{ fontSize: '0.72rem', color: 'var(--text-tertiary)', marginTop: 2 }}
        >
          {meta || '—'}
        </div>
      </div>
      <Badge tone={tone} mono>
        {pct}%
      </Badge>
      <Button size="sm" variant="secondary" disabled={busy} onClick={onUse}>
        Use
      </Button>
    </div>
  );
}
