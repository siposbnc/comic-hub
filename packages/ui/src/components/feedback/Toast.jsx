import React from 'react';
import { Icon } from '../core/Icon';
import { IconButton } from '../core/IconButton';

/** Toast — a transient ink notification with a colored status edge. Concrete, actionable. */
const TONES = {
  info: { icon: 'info', color: 'var(--accent)' },
  success: { icon: 'check', color: 'var(--success)' },
  warning: { icon: 'alert-triangle', color: 'var(--warning)' },
  danger: { icon: 'alert-triangle', color: 'var(--danger)' },
};

export function Toast({ tone = 'info', title, children, action, onClose, style }) {
  const t = TONES[tone] || TONES.info;
  return (
    <div
      role="status"
      style={{
        display: 'flex', alignItems: 'flex-start', gap: 12,
        width: 380, maxWidth: '100%', padding: '13px 14px',
        background: 'var(--surface-raised)',
        border: '1px solid var(--border-strong)',
        borderLeft: `3px solid ${t.color}`,
        borderRadius: 'var(--radius-md)',
        boxShadow: 'var(--shadow-popover)',
        ...style,
      }}
    >
      <span style={{ marginTop: 1 }}><Icon name={t.icon} size={18} color={t.color} /></span>
      <div style={{ flex: 1, minWidth: 0 }}>
        {title && <div style={{ fontWeight: 600, fontSize: 'var(--text-body)', color: 'var(--text-primary)' }}>{title}</div>}
        {children && <div style={{ marginTop: 2, fontSize: 'var(--text-small)', color: 'var(--text-secondary)', lineHeight: 1.45 }}>{children}</div>}
        {action && <div style={{ marginTop: 10, display: 'flex', gap: 8 }}>{action}</div>}
      </div>
      {onClose && <IconButton icon="x" size="sm" label="Dismiss" onClick={onClose} />}
    </div>
  );
}
