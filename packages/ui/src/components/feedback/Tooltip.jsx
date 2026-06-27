import React from 'react';

/** Tooltip — a small mono-label popover on hover. Quiet, dark, hairline-bordered. */
export function Tooltip({ label, side = 'top', children, style }) {
  const [show, setShow] = React.useState(false);
  const pos = {
    top: { bottom: '100%', left: '50%', transform: 'translateX(-50%)', marginBottom: 6 },
    bottom: { top: '100%', left: '50%', transform: 'translateX(-50%)', marginTop: 6 },
    left: { right: '100%', top: '50%', transform: 'translateY(-50%)', marginRight: 6 },
    right: { left: '100%', top: '50%', transform: 'translateY(-50%)', marginLeft: 6 },
  };
  return (
    <span
      style={{ position: 'relative', display: 'inline-flex' }}
      onMouseEnter={() => setShow(true)}
      onMouseLeave={() => setShow(false)}
      onFocus={() => setShow(true)}
      onBlur={() => setShow(false)}
    >
      {children}
      {show && (
        <span
          role="tooltip"
          style={{
            position: 'absolute', zIndex: 900, whiteSpace: 'nowrap', pointerEvents: 'none',
            padding: '5px 8px',
            fontFamily: 'var(--font-mono)', fontSize: '0.7rem', letterSpacing: '0.02em',
            color: 'var(--text-primary)',
            background: 'var(--ink-700)',
            border: '1px solid var(--border-strong)',
            borderRadius: 'var(--radius-sm)',
            boxShadow: 'var(--shadow-popover)',
            ...pos[side], ...style,
          }}
        >
          {label}
        </span>
      )}
    </span>
  );
}
