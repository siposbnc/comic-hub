import React from 'react';

/**
 * ProgressBar — cyan fill on an ink track. Aggregate series progress, scan/job
 * progress, match confidence. Mono "x of y" label optional.
 */
export function ProgressBar({ value = 0, max = 1, label, tone = 'accent', height = 6, showCount = false, style }) {
  const pct = Math.max(0, Math.min(100, (value / max) * 100));
  const fill = tone === 'unread' ? 'var(--unread)' : tone === 'success' ? 'var(--success)' : 'var(--accent)';
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 6, ...style }}>
      {(label || showCount) && (
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', gap: 12 }}>
          {label && <span style={{ fontSize: 'var(--text-small)', color: 'var(--text-secondary)' }}>{label}</span>}
          {showCount && <span className="ch-mono" style={{ fontSize: '0.72rem', color: 'var(--text-tertiary)' }}>{value} / {max}</span>}
        </div>
      )}
      <div style={{ height, borderRadius: 'var(--radius-pill)', background: 'var(--ink-600)', overflow: 'hidden' }}>
        <div style={{ width: `${pct}%`, height: '100%', background: fill, borderRadius: 'var(--radius-pill)', transition: 'width var(--dur-base) var(--ease-standard)' }} />
      </div>
    </div>
  );
}
