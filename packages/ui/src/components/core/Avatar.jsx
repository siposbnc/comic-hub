import React from 'react';

/** Avatar — a user's initial(s) on a tinted ink chip. Multi-user household switcher. */
const SIZES = { sm: 24, md: 32, lg: 40 };

export function Avatar({ name = '?', src, size = 'md', accent, style, ...rest }) {
  const dim = SIZES[size] || SIZES.md;
  const initials = name.trim().split(/\s+/).slice(0, 2).map((w) => w[0]).join('').toUpperCase();
  const tint = accent || 'var(--accent)';

  return (
    <span
      title={name}
      style={{
        display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
        width: dim, height: dim, flex: 'none',
        borderRadius: 'var(--radius-pill)', overflow: 'hidden',
        fontFamily: 'var(--font-body)', fontSize: dim * 0.4, fontWeight: 600,
        color: 'var(--text-on-accent)',
        background: src ? 'var(--surface-card)' : `color-mix(in oklab, ${tint} 78%, #000)`,
        border: '1px solid var(--border-hairline)',
        ...style,
      }}
      {...rest}
    >
      {src ? <img src={src} alt={name} style={{ width: '100%', height: '100%', objectFit: 'cover' }} /> : initials}
    </span>
  );
}
