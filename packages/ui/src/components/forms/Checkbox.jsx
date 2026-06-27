import React from 'react';
import { Icon } from '../core/Icon';

/** Checkbox — ink box, cyan fill + check when on. Used for multiselect and lock toggles. */
export function Checkbox({ checked = false, onChange, label, disabled = false, indeterminate = false, style, ...rest }) {
  return (
    <label
      style={{
        display: 'inline-flex', alignItems: 'center', gap: 9,
        cursor: disabled ? 'not-allowed' : 'pointer', opacity: disabled ? 0.5 : 1,
        fontFamily: 'var(--font-body)', fontSize: 'var(--text-body)', color: 'var(--text-primary)',
        userSelect: 'none', ...style,
      }}
      {...rest}
    >
      <span
        onClick={() => !disabled && onChange && onChange(!checked)}
        style={{
          display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
          width: 18, height: 18, flex: 'none',
          borderRadius: 'var(--radius-sm)',
          background: checked || indeterminate ? 'var(--accent)' : 'var(--surface-card)',
          border: `1px solid ${checked || indeterminate ? 'var(--accent)' : 'var(--border-strong)'}`,
          color: 'var(--text-on-accent)',
          transition: 'background var(--dur-fast), border-color var(--dur-fast)',
        }}
      >
        {indeterminate ? <Icon name="minus" size={13} /> : checked ? <Icon name="check" size={13} /> : null}
      </span>
      {label && <span>{label}</span>}
    </label>
  );
}
