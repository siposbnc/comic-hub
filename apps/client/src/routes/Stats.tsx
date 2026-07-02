import { useQuery } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';
import type { ReadingStats } from '@comichub/api-client';
import { useClient } from '../lib/client.js';
import { LoadingState, ErrorState, Page } from '../components/Page.js';

/**
 * G1 — the personal reading dashboard (design_handoff_stats): headline numbers,
 * issues-per-month bars, top genres, and a cover-forward recently-finished row.
 * Charts are token-styled divs, no SVG; cyan is the only saturated color.
 */
export function Stats() {
  const client = useClient();
  const stats = useQuery({ queryKey: ['me', 'stats'], queryFn: () => client.stats() });

  if (stats.isLoading) {
    return (
      <Page title="Stats">
        <LoadingState />
      </Page>
    );
  }
  if (stats.isError || !stats.data) {
    return (
      <Page title="Stats">
        <ErrorState
          message={stats.error instanceof Error ? stats.error.message : 'Could not load stats.'}
          onRetry={() => stats.refetch()}
        />
      </Page>
    );
  }
  const s = stats.data;
  // The pages number doubles as a rough time estimate (~24s a page).
  const hours = Math.round((s.pagesRead * 24) / 3600);

  return (
    <div style={{ padding: 'var(--pad-screen)', maxWidth: 'var(--content-max)', margin: '0 auto' }}>
      <div style={{ display: 'flex', alignItems: 'baseline', gap: 14 }}>
        <h1
          style={{
            margin: 0,
            fontFamily: 'var(--font-display)',
            fontWeight: 800,
            fontSize: 'var(--text-display-l)',
            letterSpacing: '-0.01em',
            color: 'var(--text-primary)',
          }}
        >
          Your reading
        </h1>
        <span
          className="ch-mono"
          style={{
            fontSize: '0.72rem',
            letterSpacing: '0.08em',
            textTransform: 'uppercase',
            color: 'var(--text-tertiary)',
          }}
        >
          Last 12 months
        </span>
      </div>

      {/* headline numbers */}
      <div
        style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 14, marginTop: 22 }}
      >
        <StatCard value={s.booksRead.toLocaleString()} unit="issues read" sub="all time" />
        <StatCard
          value={s.pagesRead.toLocaleString()}
          unit="pages read"
          sub={
            hours > 0 ? `≈ ${hours} ${hours === 1 ? 'hr' : 'hrs'} in the reader` : 'and counting'
          }
        />
        <StatCard
          value={s.streak}
          unit="day streak"
          sub={`best this year: ${s.bestStreak}`}
          accent
        />
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1.5fr 1fr', gap: 18, marginTop: 18 }}>
        <Panel title="Issues read per month">
          <MonthBars months={s.months} />
        </Panel>
        <Panel title="Top genres">
          <GenreBars genres={s.genres} />
        </Panel>
      </div>

      {s.finished.length > 0 && <RecentlyFinished stats={s} />}
    </div>
  );
}

function StatCard({
  value,
  unit,
  sub,
  accent,
}: {
  value: string | number;
  unit: string;
  sub: string;
  accent?: boolean;
}) {
  return (
    <div
      style={{
        padding: '18px 20px',
        borderRadius: 'var(--radius-lg)',
        background: 'var(--surface-raised)',
        border: '1px solid var(--border-hairline)',
      }}
    >
      <div
        style={{
          fontFamily: 'var(--font-display)',
          fontWeight: 800,
          fontSize: '2.6rem',
          lineHeight: 1,
          letterSpacing: '-0.02em',
          color: accent ? 'var(--accent)' : 'var(--paper-100)',
          fontVariantNumeric: 'tabular-nums',
        }}
      >
        {value}
      </div>
      <div
        className="ch-mono"
        style={{
          fontSize: '0.68rem',
          letterSpacing: '0.1em',
          textTransform: 'uppercase',
          color: 'var(--text-secondary)',
          marginTop: 10,
        }}
      >
        {unit}
      </div>
      <div style={{ fontSize: '0.76rem', color: 'var(--text-tertiary)', marginTop: 4 }}>{sub}</div>
    </div>
  );
}

function Panel({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div
      style={{
        padding: '16px 18px 18px',
        borderRadius: 'var(--radius-lg)',
        background: 'var(--surface-raised)',
        border: '1px solid var(--border-hairline)',
      }}
    >
      <Eyebrow>{title}</Eyebrow>
      {children}
    </div>
  );
}

function Eyebrow({ children, style }: { children: React.ReactNode; style?: React.CSSProperties }) {
  return (
    <div
      className="ch-mono"
      style={{
        fontSize: '0.62rem',
        fontWeight: 600,
        letterSpacing: '0.16em',
        textTransform: 'uppercase',
        color: 'var(--text-tertiary)',
        ...style,
      }}
    >
      {children}
    </div>
  );
}

function MonthBars({ months }: { months: ReadingStats['months'] }) {
  const maxN = Math.max(1, ...months.map((m) => m.n));
  return (
    <div style={{ display: 'flex', alignItems: 'flex-end', gap: 8, height: 168, marginTop: 6 }}>
      {months.map((m, i) => {
        const peak = m.n === maxN && m.n > 0;
        return (
          <div
            key={i}
            style={{
              flex: 1,
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              gap: 7,
              height: '100%',
              justifyContent: 'flex-end',
            }}
          >
            <span
              className="ch-mono"
              style={{ fontSize: '0.6rem', color: peak ? 'var(--accent)' : 'var(--text-tertiary)' }}
            >
              {m.n}
            </span>
            <div
              style={{
                width: '100%',
                height: `${(m.n / maxN) * 100}%`,
                minHeight: 4,
                borderRadius: '3px 3px 0 0',
                background: peak
                  ? 'var(--accent)'
                  : 'color-mix(in oklab, var(--accent) 32%, var(--surface-card))',
              }}
            />
            <span
              className="ch-mono"
              style={{
                fontSize: '0.58rem',
                letterSpacing: '0.04em',
                color: 'var(--text-tertiary)',
              }}
            >
              {m.m}
            </span>
          </div>
        );
      })}
    </div>
  );
}

function GenreBars({ genres }: { genres: ReadingStats['genres'] }) {
  if (genres.length === 0) {
    return (
      <p style={{ margin: '10px 0 0', fontSize: '0.82rem', color: 'var(--text-tertiary)' }}>
        Genres appear once matched issues are finished.
      </p>
    );
  }
  const gMax = Math.max(1, ...genres.map((g) => g.n));
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 13, marginTop: 4 }}>
      {genres.slice(0, 5).map((g) => (
        <div key={g.name}>
          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'baseline',
              marginBottom: 6,
            }}
          >
            <span style={{ fontSize: '0.82rem', color: 'var(--text-primary)' }}>{g.name}</span>
            <span className="ch-mono" style={{ fontSize: '0.7rem', color: 'var(--text-tertiary)' }}>
              {g.n}
            </span>
          </div>
          <div
            style={{
              height: 6,
              borderRadius: 999,
              background: 'var(--surface-card)',
              overflow: 'hidden',
            }}
          >
            <span
              style={{
                display: 'block',
                height: '100%',
                width: `${(g.n / gMax) * 100}%`,
                borderRadius: 999,
                background: 'var(--accent)',
              }}
            />
          </div>
        </div>
      ))}
    </div>
  );
}

function RecentlyFinished({ stats }: { stats: ReadingStats }) {
  const client = useClient();
  const navigate = useNavigate();
  return (
    <div style={{ marginTop: 18 }}>
      <Eyebrow style={{ marginBottom: 14 }}>Recently finished</Eyebrow>
      <div style={{ display: 'flex', gap: 16, flexWrap: 'wrap' }}>
        {stats.finished.map((f) => (
          <button
            key={f.bookId}
            type="button"
            onClick={() => navigate({ to: '/book/$id', params: { id: f.bookId } })}
            style={{
              width: 116,
              flex: 'none',
              padding: 0,
              border: 'none',
              background: 'transparent',
              textAlign: 'left',
              cursor: 'pointer',
            }}
          >
            <div className="ch-reg">
              <img
                src={client.coverUrl(f.bookId, 300)}
                alt=""
                style={{
                  width: '100%',
                  aspectRatio: '2 / 3',
                  objectFit: 'cover',
                  display: 'block',
                }}
              />
            </div>
            <div
              style={{
                fontSize: '0.78rem',
                fontWeight: 600,
                color: 'var(--paper-100)',
                marginTop: 8,
                whiteSpace: 'nowrap',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
              }}
            >
              {f.title || f.seriesName}
            </div>
            <div
              className="ch-mono"
              style={{ fontSize: '0.66rem', color: 'var(--text-tertiary)', marginTop: 2 }}
            >
              {[f.number ? `#${f.number.replace(/^#/, '')}` : null, 'done']
                .filter(Boolean)
                .join(' · ')}
            </div>
          </button>
        ))}
      </div>
    </div>
  );
}
