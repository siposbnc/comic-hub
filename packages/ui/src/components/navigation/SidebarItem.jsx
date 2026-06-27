import React from 'react';
import { Icon } from '../core/Icon';

/** A single sidebar nav row — icon + label, optional mono count. Cyan when active. */
export function SidebarItem({ icon, label, count, active = false, onClick, style }) {
  const [hover, setHover] = React.useState(false);
  return (
    <button
      type="button"
      onClick={onClick}
      onMouseEnter={() => setHover(true)}
      onMouseLeave={() => setHover(false)}
      style={{
        display: 'flex', alignItems: 'center', gap: 11, width: '100%',
        height: 36, padding: '0 10px',
        background: active ? 'var(--accent-soft)' : hover ? 'var(--surface-hover)' : 'transparent',
        border: 'none', borderRadius: 'var(--radius-md)', cursor: 'pointer',
        color: active ? 'var(--accent)' : 'var(--text-secondary)',
        fontFamily: 'var(--font-body)', fontSize: 'var(--text-body)', fontWeight: active ? 600 : 500,
        textAlign: 'left', transition: 'background var(--dur-fast), color var(--dur-fast)',
        ...style,
      }}
    >
      {icon && <Icon name={icon} size={18} color={active ? 'var(--accent)' : 'var(--text-tertiary)'} />}
      <span style={{ flex: 1, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{label}</span>
      {count != null && (
        <span className="ch-mono" style={{ fontSize: '0.7rem', color: active ? 'var(--accent)' : 'var(--text-tertiary)' }}>{count}</span>
      )}
    </button>
  );
}

/** A mono section label that separates sidebar groups. */
export function SidebarSection({ children, style }) {
  return (
    <div className="ch-label" style={{ color: 'var(--text-tertiary)', padding: '14px 10px 6px', ...style }}>
      {children}
    </div>
  );
}
