import React from 'react';
import { Icon } from '../core/Icon';

/** Text input on an ink fill with a 6px radius; cyan focus ring. Optional leading icon. */
export function Input({
  icon, size = 'md', invalid = false, style, disabled = false, ...rest
}) {
  const [focus, setFocus] = React.useState(false);
  const h = size === 'sm' ? 32 : size === 'lg' ? 44 : 38;

  return (
    <div
      style={{
        display: 'flex', alignItems: 'center', gap: 8,
        height: h, padding: '0 12px',
        background: 'var(--surface-card)',
        border: `1px solid ${invalid ? 'var(--danger)' : focus ? 'var(--accent)' : 'var(--border-hairline)'}`,
        borderRadius: 'var(--radius-md)',
        boxShadow: focus ? '0 0 0 3px color-mix(in oklab, var(--accent) 22%, transparent)' : 'none',
        transition: 'border-color var(--dur-fast), box-shadow var(--dur-fast)',
        opacity: disabled ? 0.5 : 1,
        ...style,
      }}
    >
      {icon && <Icon name={icon} size={16} color="var(--text-tertiary)" />}
      <input
        disabled={disabled}
        onFocus={(e) => { setFocus(true); rest.onFocus && rest.onFocus(e); }}
        onBlur={(e) => { setFocus(false); rest.onBlur && rest.onBlur(e); }}
        {...rest}
        style={{
          flex: 1, minWidth: 0, height: '100%',
          background: 'transparent', border: 'none', outline: 'none',
          color: 'var(--text-primary)',
          fontFamily: 'var(--font-body)', fontSize: 'var(--text-body)',
        }}
      />
    </div>
  );
}
