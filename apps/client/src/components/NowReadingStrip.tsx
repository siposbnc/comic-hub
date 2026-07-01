import { useEffect, useRef, useState } from 'react';
import { useNavigate } from '@tanstack/react-router';
import { Avatar } from '@comichub/ui';
import type { PresenceEntry } from '@comichub/api-client';
import { useClient } from '../lib/client.js';
import { usePresence } from '../lib/queries.js';
import { tokenStore, useAuthStore } from '../lib/auth.js';

/**
 * "Now reading" — ambient household presence on Home (design_handoff_presence).
 * Renders nothing in embedded / auth-off mode (no household) and when nobody is
 * reading: zero readers is the normal state, so the layout must sit flush without it.
 * Fed by the GET /presence snapshot; the WS presence topic keeps the query cache live.
 */
export function NowReadingStrip() {
  const authMode = !!tokenStore.refresh();
  const presence = usePresence(authMode);
  const entries = presence.data ?? [];
  const leaving = useLeaving(entries);

  if (!authMode || (entries.length === 0 && leaving.length === 0)) return null;

  const CAP = 4;
  const shown = entries.slice(0, CAP);
  const overflow = entries.length - shown.length;

  return (
    <div>
      <LiveStyles />
      <div style={{ display: 'flex', alignItems: 'center', gap: 9, marginBottom: 12 }}>
        <LiveDot />
        <span
          className="ch-mono"
          style={{
            fontSize: '0.62rem',
            fontWeight: 600,
            letterSpacing: '0.16em',
            textTransform: 'uppercase',
            color: 'var(--text-tertiary)',
          }}
        >
          Now reading
        </span>
        <span className="ch-mono" style={{ fontSize: '0.62rem', color: 'var(--paper-600)' }}>
          · {entries.length} on the server
        </span>
      </div>
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 12 }}>
        {/* one list, keyed by user, so a card that flips to fading keeps its DOM node
            and the 400ms opacity transition actually runs */}
        {[
          ...shown.map((p) => ({ p, fading: false })),
          ...leaving.map((p) => ({ p, fading: true })),
        ].map(({ p, fading }) => (
          <PresenceCard key={p.userId} entry={p} fading={fading} />
        ))}
        {overflow > 0 && (
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              width: 96,
              flex: 'none',
              borderRadius: 'var(--radius-md)',
              border: '1px dashed var(--border-hairline)',
              color: 'var(--text-tertiary)',
              fontFamily: 'var(--font-body)',
              fontSize: '0.82rem',
            }}
          >
            +{overflow} more
          </div>
        )}
      </div>
    </div>
  );
}

/** One household reader: cover (loud) + avatar tag + name + book line + progress. */
function PresenceCard({ entry, fading }: { entry: PresenceEntry; fading?: boolean }) {
  const client = useClient();
  const navigate = useNavigate();
  const you = useAuthStore((s) => s.user)?.id === entry.userId;
  const [hover, setHover] = useState(false);
  const pct = entry.pageCount > 0 ? Math.round((entry.page / entry.pageCount) * 100) : 0;
  const active = hover && !fading;
  const bookLine = [entry.seriesTitle, entry.bookTitle].filter(Boolean).join(' · ');

  return (
    <button
      type="button"
      onMouseEnter={() => setHover(true)}
      onMouseLeave={() => setHover(false)}
      onClick={() => navigate({ to: '/book/$id', params: { id: entry.bookId } })}
      disabled={fading}
      title={`${entry.displayName} · ${bookLine}`}
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 13,
        width: 252,
        flex: 'none',
        textAlign: 'left',
        padding: '9px 12px 9px 9px',
        borderRadius: 'var(--radius-md)',
        cursor: fading ? 'default' : 'pointer',
        background: active ? 'var(--surface-card)' : 'transparent',
        border: `1px solid ${active ? 'var(--border-hairline)' : 'transparent'}`,
        opacity: fading ? 0.4 : 1,
        transition: 'background 120ms, opacity 400ms',
      }}
    >
      <div className="ch-reg" style={{ position: 'relative', flex: 'none' }}>
        <img
          src={client.coverUrl(entry.bookId, 200)}
          alt=""
          width={42}
          height={63}
          style={{ objectFit: 'cover', display: 'block', boxShadow: 'var(--shadow-card)' }}
        />
        <span style={{ position: 'absolute', left: -7, bottom: -7 }}>
          <Avatar name={entry.displayName} size="sm" accent={userAccent(entry.userId)} />
        </span>
      </div>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 7 }}>
          <span
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
            {entry.displayName}
          </span>
          {you && (
            <span
              className="ch-mono"
              style={{
                flex: 'none',
                fontSize: '0.52rem',
                letterSpacing: '0.1em',
                textTransform: 'uppercase',
                color: 'var(--text-tertiary)',
                padding: '1px 5px',
                border: '1px solid var(--border-hairline)',
                borderRadius: 'var(--radius-sm)',
              }}
            >
              You
            </span>
          )}
        </div>
        <div
          className="ch-mono"
          style={{
            fontSize: '0.66rem',
            color: 'var(--text-tertiary)',
            margin: '3px 0 6px',
            whiteSpace: 'nowrap',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
          }}
        >
          {bookLine}
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <span className="ch-progress" style={{ flex: 1, borderRadius: 999 }}>
            <span style={{ width: `${pct}%`, borderRadius: 999 }} />
          </span>
          <span
            className="ch-mono"
            style={{
              flex: 'none',
              fontSize: '0.6rem',
              color: 'var(--paper-600)',
              fontVariantNumeric: 'tabular-nums',
            }}
          >
            p.{entry.page}/{entry.pageCount}
          </span>
        </div>
      </div>
    </button>
  );
}

/** The calm "live" cue: cyan dot + one soft expanding ring. */
function LiveDot() {
  return (
    <span
      style={{ position: 'relative', display: 'inline-flex', width: 7, height: 7, flex: 'none' }}
    >
      <span
        className="ch-live-ring"
        style={{ position: 'absolute', inset: 0, borderRadius: '50%', background: 'var(--accent)' }}
      />
      <span
        style={{
          position: 'relative',
          width: 7,
          height: 7,
          borderRadius: '50%',
          background: 'var(--accent)',
        }}
      />
    </span>
  );
}

/** The ring keyframes live here until they're folded into the DS base.css (which is
 *  synced verbatim and must not be hand-edited). Honors prefers-reduced-motion. */
function LiveStyles() {
  return (
    <style>{`
      @keyframes ch-live-ring { 0% { transform: scale(1); opacity: 0.5; } 70%, 100% { transform: scale(2.6); opacity: 0; } }
      .ch-live-ring { animation: ch-live-ring 2.4s ease-out infinite; }
      @media (prefers-reduced-motion: reduce) { .ch-live-ring { animation: none; opacity: 0; } }
    `}</style>
  );
}

/** Entries that just left the feed linger for one 400ms fade before unmounting. The
 *  removal timers are per-departure and survive later presence ticks (a tick within
 *  the fade window must not strand a ghost card); all pending timers clear on unmount. */
function useLeaving(entries: PresenceEntry[]): PresenceEntry[] {
  const [leaving, setLeaving] = useState<PresenceEntry[]>([]);
  const prev = useRef<PresenceEntry[]>([]);
  const timers = useRef<Set<ReturnType<typeof setTimeout>>>(new Set());
  useEffect(() => {
    const current = new Set(entries.map((e) => e.userId));
    const gone = prev.current.filter((e) => !current.has(e.userId));
    prev.current = entries;
    if (gone.length === 0) return;
    setLeaving((cur) => [
      ...cur.filter((e) => !current.has(e.userId) && !gone.some((g) => g.userId === e.userId)),
      ...gone,
    ]);
    const t = setTimeout(() => {
      timers.current.delete(t);
      setLeaving((cur) => cur.filter((e) => !gone.some((g) => g.userId === e.userId)));
    }, 400);
    timers.current.add(t);
  }, [entries]);
  useEffect(() => {
    const pending = timers.current;
    return () => {
      for (const t of pending) clearTimeout(t);
    };
  }, []);
  return leaving.filter((e) => !entries.some((cur) => cur.userId === e.userId));
}

/* Per-user avatar tint — the palette from the presence/user-management handoffs, keyed
   deterministically off the user id so a member keeps their color everywhere. */
const ACCENT_PALETTE = ['#16B9E6', '#E6398B', '#46A758', '#F4C13C', '#3B6BE6'];

function userAccent(userId: string): string {
  let h = 0;
  for (let i = 0; i < userId.length; i++) h = (h * 31 + userId.charCodeAt(i)) >>> 0;
  return ACCENT_PALETTE[h % ACCENT_PALETTE.length]!;
}
