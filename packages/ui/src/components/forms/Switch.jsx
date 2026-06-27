import React from 'react';

/** Switch — a binary toggle. Cyan track when on. Settings (watch folder, theme, etc). */
export function Switch({ checked = false, onChange, disabled = false, label, style, ...rest }) {
  return (
    <label
      style={{
        display: 'inline-flex', alignItems: 'center', gap: 10,
        cursor: disabled ? 'not-allowed' : 'pointer', opacity: disabled ? 0.5 : 1,
        fontFamily: 'var(--font-body)', fontSize: 'var(--text-body)', color: 'var(--text-primary)',
        userSelect: 'none', ...style,
      }}
      {...rest}
    >
      <span
        role="switch"
        aria-checked={checked}
        onClick={() => !disabled && onChange && onChange(!checked)}
        style={{
          position: 'relative', width: 36, height: 20, flex: 'none',
          borderRadius: 'var(--radius-pill)',
          background: checked ? 'var(--accent)' : 'var(--ink-600)',
          border: '1px solid', borderColor: checked ? 'var(--accent)' : 'var(--border-strong)',
          transition: 'background var(--dur-fast), border-color var(--dur-fast)',
        }}
      >
        <span
          style={{
            position: 'absolute', top: 2, left: checked ? 17 : 2,
            width: 14, height: 14, borderRadius: '50%',
            background: checked ? 'var(--text-on-accent)' : 'var(--paper-100)',
            transition: 'left var(--dur-fast) var(--ease-standard)',
          }}
        />
      </span>
      {label && <span>{label}</span>}
    </label>
  );
}
