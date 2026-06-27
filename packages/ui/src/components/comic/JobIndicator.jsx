import React from 'react';
import { Icon } from '../core/Icon';

/**
 * JobIndicator — the top-bar scan/job status pill. Spinning glyph + mono label while
 * active; a popover lists per-job progress. Non-blocking; lives in the utility bar.
 */
export function JobIndicator({ status = 'idle', label, jobs = [], style }) {
  const [open, setOpen] = React.useState(false);
  const tone = status === 'scanning' ? 'var(--accent)' : status === 'error' ? 'var(--danger)' : 'var(--text-tertiary)';
  const icon = status === 'scanning' ? 'refresh' : status === 'error' ? 'alert-triangle' : 'check';

  return (
    <div style={{ position: 'relative', ...style }}>
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        style={{
          display: 'inline-flex', alignItems: 'center', gap: 7,
          height: 30, padding: '0 11px',
          background: status === 'idle' ? 'transparent' : 'var(--surface-card)',
          border: `1px solid ${status === 'idle' ? 'transparent' : 'var(--border-hairline)'}`,
          borderRadius: 'var(--radius-pill)', cursor: 'pointer',
          color: tone, fontFamily: 'var(--font-mono)', fontSize: '0.72rem', letterSpacing: '0.02em',
        }}
      >
        <span style={{ display: 'inline-flex', animation: status === 'scanning' ? 'ch-spin 1.1s linear infinite' : 'none' }}>
          <Icon name={icon} size={14} color={tone} />
        </span>
        {label || (status === 'scanning' ? 'Scanning…' : status === 'error' ? 'Error' : 'Up to date')}
      </button>

      {open && jobs.length > 0 && (
        <div
          style={{
            position: 'absolute', top: '100%', right: 0, marginTop: 8, zIndex: 900, width: 300,
            background: 'var(--surface-raised)', border: '1px solid var(--border-strong)',
            borderRadius: 'var(--radius-lg)', boxShadow: 'var(--shadow-popover)', padding: 12,
          }}
        >
          <div className="ch-label" style={{ color: 'var(--text-tertiary)', marginBottom: 10 }}>Background jobs</div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            {jobs.map((j, i) => (
              <div key={i}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', marginBottom: 6 }}>
                  <span style={{ fontSize: 'var(--text-small)', color: 'var(--text-primary)' }}>{j.name}</span>
                  <span className="ch-mono" style={{ fontSize: '0.7rem', color: 'var(--text-tertiary)' }}>{Math.round((j.progress || 0) * 100)}%</span>
                </div>
                <div className="ch-progress" style={{ borderRadius: 'var(--radius-pill)' }}>
                  <span style={{ width: `${Math.round((j.progress || 0) * 100)}%` }} />
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
