import { useState, type CSSProperties } from 'react';
import { useNavigate, useRouterState } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { Icon, type IconProps } from '@comichub/ui';
import { useClient } from '../lib/client.js';
import {
  useLibraries,
  useLibrarySeriesCounts,
  useContinueReading,
  useSeriesNames,
} from '../lib/queries.js';
import { issueLabel } from '../lib/format.js';

/**
 * The "longbox" left rail (ported from the ComicHub Preview v2 client shell): a vertical
 * spine plate, stylized SpineTab nav rows (registration-tick hover, active = a clipped
 * cyan tab), and a live "continue reading" footer card.
 */
export function Sidebar() {
  const navigate = useNavigate();
  const client = useClient();
  const pathname = useRouterState({ select: (s) => s.location.pathname });

  const { data: libraries } = useLibraries();
  const counts = useLibrarySeriesCounts();
  const stats = useQuery({ queryKey: ['server', 'stats'], queryFn: () => client.serverStats() });
  const info = useQuery({ queryKey: ['server', 'info'], queryFn: () => client.serverInfo() });

  const continueReading = useContinueReading();
  const seriesNames = useSeriesNames();
  const now = continueReading.data?.[0];

  const subtitle = stats.data
    ? `${stats.data.books.toLocaleString()} issues · ${stats.data.series} series`
    : '—';

  let idx = 0;
  const next = () => String(++idx).padStart(2, '0');

  return (
    <aside
      aria-label="Primary"
      style={{
        width: 256,
        flex: 'none',
        height: '100vh',
        display: 'flex',
        background: 'var(--surface-raised)',
        borderRight: '1px solid var(--border-hairline)',
      }}
    >
      {/* Spine plate — a longbox spine standing on its edge */}
      <div
        style={{
          width: 34,
          flex: 'none',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '12px 0 14px',
          background: 'var(--ink-900)',
          borderRight: '1px solid var(--border-hairline)',
        }}
      >
        <img src="/comichub-mark.svg" width={22} height={22} alt="" aria-hidden />
        <div
          style={{
            writingMode: 'vertical-rl',
            fontFamily: 'var(--font-display)',
            fontWeight: 800,
            fontSize: '0.82rem',
            letterSpacing: '0.32em',
            textTransform: 'uppercase',
            color: 'var(--paper-400)',
            userSelect: 'none',
          }}
        >
          Comic<span style={{ color: 'var(--accent)' }}>Hub</span>
        </div>
        <span
          className="ch-mono"
          style={{
            writingMode: 'vertical-rl',
            fontSize: '0.56rem',
            letterSpacing: '0.1em',
            color: 'var(--paper-600)',
          }}
        >
          {info.data ? `v${info.data.version}` : ''}
        </span>
      </div>

      {/* Nav column */}
      <div style={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}>
        <div
          style={{
            height: 'var(--topbar-height, 56px)',
            flex: 'none',
            display: 'flex',
            flexDirection: 'column',
            justifyContent: 'center',
            padding: '0 14px',
            borderBottom: '1px solid var(--border-hairline)',
          }}
        >
          <span
            style={{
              fontFamily: 'var(--font-display)',
              fontWeight: 800,
              fontSize: '1.15rem',
              letterSpacing: '-0.02em',
              lineHeight: 1,
            }}
          >
            Comic<span style={{ color: 'var(--accent)' }}>Hub</span>
          </span>
          <span
            className="ch-mono"
            style={{
              fontSize: '0.6rem',
              letterSpacing: '0.12em',
              textTransform: 'uppercase',
              color: 'var(--paper-600)',
              marginTop: 4,
            }}
          >
            {subtitle}
          </span>
        </div>

        <nav style={{ flex: 1, overflowY: 'auto', padding: '6px 8px 8px' }}>
          <SpineTab
            index={next()}
            icon="home"
            label="Home"
            active={pathname === '/'}
            onClick={() => navigate({ to: '/' })}
          />

          <NavSection>Libraries</NavSection>
          {libraries?.length === 0 && (
            <div
              style={{
                padding: '8px 12px',
                color: 'var(--paper-600)',
                fontSize: 'var(--text-small)',
              }}
            >
              No libraries yet
            </div>
          )}
          {libraries?.map((lib) => (
            <SpineTab
              key={lib.id}
              index={next()}
              icon="library"
              label={lib.name}
              count={counts.get(lib.id)}
              active={pathname === `/library/${lib.id}`}
              onClick={() => navigate({ to: '/library/$id', params: { id: lib.id } })}
            />
          ))}

          <NavSection>Lists</NavSection>
          <SpineTab
            index={next()}
            icon="collection"
            label="Collections"
            active={pathname.startsWith('/collections')}
            onClick={() => navigate({ to: '/collections' })}
          />
          <SpineTab
            index={next()}
            icon="bookmark"
            label="Reading Lists"
            active={pathname.startsWith('/reading-lists')}
            onClick={() => navigate({ to: '/reading-lists' })}
          />
          <SpineTab
            index={next()}
            icon="filter"
            label="Smart Lists"
            active={pathname.startsWith('/smart-lists')}
            onClick={() => navigate({ to: '/smart-lists' })}
          />
          <SpineTab
            index={next()}
            icon="star"
            label="Tags"
            active={pathname.startsWith('/tags')}
            onClick={() => navigate({ to: '/tags' })}
          />

          <NavSection>System</NavSection>
          <SpineTab
            index={next()}
            icon="stats"
            label="Stats"
            active={pathname === '/stats'}
            onClick={() => navigate({ to: '/stats' })}
          />
          <SpineTab
            index={next()}
            icon="settings"
            label="Settings"
            active={pathname === '/settings'}
            onClick={() => navigate({ to: '/settings' })}
          />
        </nav>

        {/* Live "continue reading" footer card */}
        {now && (
          <button
            type="button"
            onClick={() => navigate({ to: '/book/$id', params: { id: now.id } })}
            style={{
              flex: 'none',
              display: 'block',
              width: '100%',
              textAlign: 'left',
              border: 'none',
              borderTop: '1px solid var(--border-hairline)',
              background: 'var(--ink-700)',
              cursor: 'pointer',
              padding: '11px 12px 12px',
            }}
          >
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                marginBottom: 8,
              }}
            >
              <span
                className="ch-mono"
                style={{
                  fontSize: '0.58rem',
                  fontWeight: 600,
                  letterSpacing: '0.16em',
                  textTransform: 'uppercase',
                  color: 'var(--accent)',
                }}
              >
                Continue reading
              </span>
              <Icon name="book-open" size={13} color="var(--accent)" />
            </div>
            <div style={{ display: 'flex', gap: 10, alignItems: 'center' }}>
              <div className="ch-reg" style={{ flex: 'none' }}>
                <img
                  src={client.coverUrl(now.id, 120)}
                  alt=""
                  style={{
                    width: 34,
                    height: 51,
                    objectFit: 'cover',
                    display: 'block',
                    boxShadow: '0 2px 8px rgba(0,0,0,.5)',
                  }}
                />
              </div>
              <div style={{ flex: 1, minWidth: 0 }}>
                <div
                  style={{
                    fontFamily: 'var(--font-body)',
                    fontWeight: 600,
                    fontSize: '0.8rem',
                    color: 'var(--paper-100)',
                    whiteSpace: 'nowrap',
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                  }}
                >
                  {seriesNames.get(now.seriesId) ?? now.title ?? 'Reading'}
                </div>
                <div
                  className="ch-mono"
                  style={{ fontSize: '0.64rem', color: 'var(--paper-400)', margin: '2px 0 7px' }}
                >
                  {[issueLabel(now.number), `p.${now.progress?.page ?? 0}/${now.pageCount}`]
                    .filter(Boolean)
                    .join(' · ')}
                </div>
                <div className="ch-progress" style={{ borderRadius: 999 }}>
                  <span
                    style={{
                      width: `${Math.round(now.progress?.percent ?? 0)}%`,
                      borderRadius: 999,
                    }}
                  />
                </div>
              </div>
            </div>
          </button>
        )}
      </div>
    </aside>
  );
}

interface SpineTabProps {
  index: string;
  icon: IconProps['name'];
  label: string;
  count?: number;
  unread?: boolean;
  active?: boolean;
  onClick?: () => void;
}

/**
 * A longbox divider tab. Inactive = a flat row with a quiet spine edge; hover draws the
 * registration ticks (.ch-reg) and lifts the spine edge to cyan; active = a filled cyan
 * tab pulled forward with the clipped spine-tab silhouette.
 */
function SpineTab({ index, icon, label, count, unread, active = false, onClick }: SpineTabProps) {
  const [hover, setHover] = useState(false);
  const tabStyle: CSSProperties = {
    position: 'relative',
    display: 'flex',
    alignItems: 'center',
    gap: 9,
    width: '100%',
    padding: '8px 12px 8px 0',
    margin: '1px 0',
    border: 'none',
    cursor: 'pointer',
    textAlign: 'left',
    fontFamily: 'var(--font-body)',
    fontSize: '0.875rem',
    fontWeight: active ? 700 : 500,
    background: active ? 'var(--accent)' : hover ? 'var(--ink-700)' : 'transparent',
    color: active ? 'var(--text-on-accent)' : hover ? 'var(--paper-100)' : 'var(--paper-400)',
    clipPath: active ? 'polygon(0 0, 100% 0, calc(100% - 13px) 100%, 0 100%)' : 'none',
    boxShadow: active ? '0 3px 12px color-mix(in oklab, var(--accent) 45%, transparent)' : 'none',
    transition: 'background var(--dur-fast), color var(--dur-fast)',
  };

  return (
    <button
      type="button"
      className="ch-reg"
      onClick={onClick}
      onMouseEnter={() => setHover(true)}
      onMouseLeave={() => setHover(false)}
      style={tabStyle}
    >
      <span
        style={{
          width: 3,
          alignSelf: 'stretch',
          flex: 'none',
          background: active ? 'var(--text-on-accent)' : hover ? 'var(--accent)' : 'var(--ink-500)',
        }}
      />
      <span
        className="ch-mono"
        style={{
          width: 18,
          fontSize: '0.64rem',
          letterSpacing: '0.04em',
          color: active
            ? 'color-mix(in oklab, var(--text-on-accent) 70%, transparent)'
            : 'var(--paper-600)',
        }}
      >
        {index}
      </span>
      <Icon name={icon} size={17} />
      <span
        style={{
          flex: 1,
          letterSpacing: '-0.005em',
          whiteSpace: 'nowrap',
          overflow: 'hidden',
          textOverflow: 'ellipsis',
        }}
      >
        {label}
      </span>
      {count != null && (
        <span
          className="ch-mono"
          style={{
            fontSize: '0.7rem',
            fontVariantNumeric: 'tabular-nums',
            color: active ? 'var(--text-on-accent)' : unread ? 'var(--unread)' : 'var(--paper-600)',
          }}
        >
          {count}
        </span>
      )}
    </button>
  );
}

/** A nav group label: an accent dash, mono uppercase text, then a hairline rule. */
function NavSection({ children }: { children: React.ReactNode }) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '16px 12px 7px 8px' }}>
      <span style={{ width: 9, height: 2, background: 'var(--accent)', flex: 'none' }} />
      <span
        className="ch-mono"
        style={{
          fontSize: '0.62rem',
          fontWeight: 600,
          letterSpacing: '0.18em',
          textTransform: 'uppercase',
          color: 'var(--paper-600)',
        }}
      >
        {children}
      </span>
      <span style={{ flex: 1, height: 1, background: 'var(--border-hairline)' }} />
    </div>
  );
}
