import React from 'react';
import { Icon } from './Icon';

/** Tag — a removable free-form label (genre, character, mood) on a quiet ink chip. */
export function Tag({ children, removable = false, onRemove, accent = false, style, ...rest }) {
  return (
    <span
      style={{
        display: 'inline-flex', alignItems: 'center', gap: 6,
        height: 24, padding: removable ? '0 5px 0 10px' : '0 10px',
        fontFamily: 'var(--font-body)', fontSize: '0.78rem', fontWeight: 500,
        color: accent ? 'var(--accent)' : 'var(--text-secondary)',
        background: 'var(--surface-card)',
        border: `1px solid ${accent ? 'var(--accent)' : 'var(--border-hairline)'}`,
        borderRadius: 'var(--radius-sm)', whiteSpace: 'nowrap', ...style,
      }}
      {...rest}
    >
      {children}
      {removable && (
        <button
          type="button"
          aria-label="Remove"
          onClick={onRemove}
          style={{
            display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
            width: 16, height: 16, padding: 0, border: 'none', borderRadius: 3,
            background: 'transparent', color: 'var(--text-tertiary)', cursor: 'pointer',
          }}
          onMouseEnter={(e) => { e.currentTarget.style.background = 'var(--surface-hover)'; e.currentTarget.style.color = 'var(--text-primary)'; }}
          onMouseLeave={(e) => { e.currentTarget.style.background = 'transparent'; e.currentTarget.style.color = 'var(--text-tertiary)'; }}
        >
          <Icon name="x" size={12} />
        </button>
      )}
    </span>
  );
}
