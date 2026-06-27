import React from 'react';
import { Icon } from '../core/Icon';

/**
 * Rail — a horizontally-scrolling row of CoverCards under a mono section label.
 * The Home/Discover building block (Continue Reading, On Deck, Recently Added).
 */
export function Rail({ label, action, children, style }) {
  return (
    <section style={{ display: 'flex', flexDirection: 'column', gap: 14, ...style }}>
      <div style={{ display: 'flex', alignItems: 'baseline', justifyContent: 'space-between', gap: 16 }}>
        <h2 className="ch-label" style={{ margin: 0, color: 'var(--text-secondary)' }}>{label}</h2>
        {action && (
          <button
            type="button"
            onClick={action.onClick}
            style={{ display: 'inline-flex', alignItems: 'center', gap: 3, background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-tertiary)', fontFamily: 'var(--font-body)', fontSize: 'var(--text-small)', fontWeight: 500 }}
            onMouseEnter={(e) => { e.currentTarget.style.color = 'var(--accent)'; }}
            onMouseLeave={(e) => { e.currentTarget.style.color = 'var(--text-tertiary)'; }}
          >
            {action.label}<Icon name="chevron-right" size={14} />
          </button>
        )}
      </div>
      <div
        style={{
          display: 'flex', gap: 14, overflowX: 'auto', paddingBottom: 4,
          scrollbarWidth: 'thin', scrollSnapType: 'x proximity',
        }}
      >
        {React.Children.map(children, (c) => (
          <div style={{ flex: 'none', scrollSnapAlign: 'start' }}>{c}</div>
        ))}
      </div>
    </section>
  );
}
