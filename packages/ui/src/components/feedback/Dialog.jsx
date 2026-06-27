import React from 'react';
import { IconButton } from '../core/IconButton';

/** Modal dialog — ink-raised card floating above a scrim. The only place real elevation appears. */
export function Dialog({ open = true, title, children, footer, onClose, width = 440 }) {
  if (!open) return null;
  return (
    <div
      onClick={onClose}
      style={{
        position: 'fixed', inset: 0, zIndex: 1000,
        display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 24,
        background: 'color-mix(in oklab, var(--ink-900) 72%, transparent)',
        backdropFilter: 'blur(3px)', WebkitBackdropFilter: 'blur(3px)',
      }}
    >
      <div
        role="dialog"
        aria-modal="true"
        onClick={(e) => e.stopPropagation()}
        style={{
          width, maxWidth: '100%',
          background: 'var(--surface-raised)',
          border: '1px solid var(--border-strong)',
          borderRadius: 'var(--radius-lg)',
          boxShadow: 'var(--shadow-dialog)',
          overflow: 'hidden',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 16, padding: '18px 20px 0' }}>
          <h2 style={{ margin: 0, fontFamily: 'var(--font-body)', fontWeight: 600, fontSize: 'var(--text-heading)', color: 'var(--text-primary)' }}>{title}</h2>
          {onClose && <IconButton icon="x" label="Close" onClick={onClose} />}
        </div>
        <div style={{ padding: '12px 20px 4px', color: 'var(--text-secondary)', fontSize: 'var(--text-body)', lineHeight: 'var(--leading-body)' }}>
          {children}
        </div>
        {footer && (
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 10, padding: '16px 20px 20px' }}>
            {footer}
          </div>
        )}
      </div>
    </div>
  );
}
