import React from 'react';

/** Tabs — underline-style segmented nav. Active tab gets a cyan underline + paper text. */
export function Tabs({ tabs = [], value, onChange, style }) {
  return (
    <div
      role="tablist"
      style={{
        display: 'flex', gap: 4,
        borderBottom: '1px solid var(--border-hairline)',
        ...style,
      }}
    >
      {tabs.map((t) => {
        const key = typeof t === 'string' ? t : t.value;
        const label = typeof t === 'string' ? t : t.label;
        const count = typeof t === 'object' ? t.count : undefined;
        const active = key === value;
        return (
          <button
            key={key}
            role="tab"
            aria-selected={active}
            onClick={() => onChange && onChange(key)}
            style={{
              position: 'relative', display: 'inline-flex', alignItems: 'center', gap: 7,
              padding: '10px 12px', marginBottom: -1,
              background: 'transparent', border: 'none', cursor: 'pointer',
              fontFamily: 'var(--font-body)', fontSize: 'var(--text-body)',
              fontWeight: active ? 600 : 500,
              color: active ? 'var(--text-primary)' : 'var(--text-secondary)',
              borderBottom: `2px solid ${active ? 'var(--accent)' : 'transparent'}`,
              transition: 'color var(--dur-fast)',
            }}
            onMouseEnter={(e) => { if (!active) e.currentTarget.style.color = 'var(--text-primary)'; }}
            onMouseLeave={(e) => { if (!active) e.currentTarget.style.color = 'var(--text-secondary)'; }}
          >
            {label}
            {count != null && (
              <span className="ch-mono" style={{ fontSize: '0.7rem', color: active ? 'var(--accent)' : 'var(--text-tertiary)' }}>{count}</span>
            )}
          </button>
        );
      })}
    </div>
  );
}
