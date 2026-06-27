import React from 'react';

/**
 * Badge — a compact status/count pill. Issue counts and "unread" markers use the
 * mono tone; status badges use semantic colors. Quiet by default.
 */
export function Badge({ children, tone = 'neutral', mono = false, dot = false, style, ...rest }) {
  const tones = {
    neutral: { background: 'var(--surface-card)', color: 'var(--text-secondary)', border: 'var(--border-hairline)' },
    accent: { background: 'var(--accent-soft)', color: 'var(--accent)', border: 'transparent' },
    unread: { background: 'var(--unread)', color: '#fff', border: 'transparent' },
    success: { background: 'color-mix(in oklab, var(--success) 18%, transparent)', color: 'var(--success)', border: 'transparent' },
    warning: { background: 'color-mix(in oklab, var(--warning) 20%, transparent)', color: 'var(--warning)', border: 'transparent' },
    danger: { background: 'color-mix(in oklab, var(--danger) 18%, transparent)', color: 'var(--danger)', border: 'transparent' },
  };
  const t = tones[tone] || tones.neutral;

  return (
    <span
      style={{
        display: 'inline-flex', alignItems: 'center', gap: 5,
        height: 20, padding: '0 8px',
        fontFamily: mono ? 'var(--font-mono)' : 'var(--font-body)',
        fontSize: mono ? '0.7rem' : '0.72rem',
        fontWeight: mono ? 500 : 600,
        fontVariantNumeric: mono ? 'tabular-nums' : 'normal',
        letterSpacing: mono ? '0.02em' : '0.01em',
        lineHeight: 1,
        color: t.color,
        background: t.background,
        border: `1px solid ${t.border}`,
        borderRadius: 'var(--radius-pill)',
        whiteSpace: 'nowrap',
        ...style,
      }}
      {...rest}
    >
      {dot && <span style={{ width: 6, height: 6, borderRadius: '50%', background: 'currentColor', flex: 'none' }} />}
      {children}
    </span>
  );
}
