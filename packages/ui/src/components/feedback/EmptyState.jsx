import React from 'react';

/**
 * EmptyState — the one place the halftone print texture appears. Invites action;
 * never a dead end. Cyan→magenta Ben-Day dot field behind a tight message.
 */
export function EmptyState({ title, children, action, style }) {
  return (
    <div
      style={{
        position: 'relative', overflow: 'hidden',
        display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center',
        textAlign: 'center', padding: '56px 32px', minHeight: 260,
        background: 'var(--surface-raised)',
        border: '1px solid var(--border-hairline)',
        borderRadius: 'var(--radius-lg)',
        ...style,
      }}
    >
      {/* Signature halftone field — used once, here. */}
      <div
        className="ch-halftone-duo"
        style={{
          position: 'absolute', inset: 0,
          maskImage: 'radial-gradient(120% 90% at 50% 0%, #000 0%, transparent 62%)',
          WebkitMaskImage: 'radial-gradient(120% 90% at 50% 0%, #000 0%, transparent 62%)',
          opacity: 0.5, pointerEvents: 'none',
        }}
      />
      <div style={{ position: 'relative' }}>
        <h3 style={{ margin: 0, fontFamily: 'var(--font-display)', fontWeight: 700, fontSize: 'var(--text-title)', color: 'var(--text-primary)' }}>{title}</h3>
        {children && <p style={{ margin: '10px auto 0', maxWidth: 380, color: 'var(--text-secondary)', fontSize: 'var(--text-body)', lineHeight: 1.55 }}>{children}</p>}
        {action && <div style={{ marginTop: 22, display: 'flex', gap: 10, justifyContent: 'center' }}>{action}</div>}
      </div>
    </div>
  );
}
