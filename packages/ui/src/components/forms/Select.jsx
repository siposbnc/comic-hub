import React from 'react';
import { Icon } from '../core/Icon';

/** Native-backed select styled to match Input — ink fill, chevron, cyan focus. */
export function Select({ children, size = 'md', style, disabled = false, ...rest }) {
  const [focus, setFocus] = React.useState(false);
  const h = size === 'sm' ? 32 : size === 'lg' ? 44 : 38;

  return (
    <div
      style={{
        position: 'relative', display: 'inline-flex', alignItems: 'center',
        height: h, background: 'var(--surface-card)',
        border: `1px solid ${focus ? 'var(--accent)' : 'var(--border-hairline)'}`,
        borderRadius: 'var(--radius-md)',
        boxShadow: focus ? '0 0 0 3px color-mix(in oklab, var(--accent) 22%, transparent)' : 'none',
        transition: 'border-color var(--dur-fast), box-shadow var(--dur-fast)',
        opacity: disabled ? 0.5 : 1, ...style,
      }}
    >
      <select
        disabled={disabled}
        onFocus={() => setFocus(true)}
        onBlur={() => setFocus(false)}
        {...rest}
        style={{
          appearance: 'none', WebkitAppearance: 'none',
          height: '100%', padding: '0 34px 0 12px',
          background: 'transparent', border: 'none', outline: 'none',
          color: 'var(--text-primary)',
          fontFamily: 'var(--font-body)', fontSize: 'var(--text-body)',
          cursor: disabled ? 'not-allowed' : 'pointer',
        }}
      >
        {children}
      </select>
      <span style={{ position: 'absolute', right: 10, pointerEvents: 'none' }}>
        <Icon name="chevron-down" size={16} color="var(--text-tertiary)" />
      </span>
    </div>
  );
}
