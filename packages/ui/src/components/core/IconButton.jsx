import React from 'react';
import { Icon } from './Icon';

/** Square icon-only button — toolbar actions, card overflow, close. Quiet by default. */
const SIZES = { sm: 28, md: 34, lg: 40 };
const ICON = { sm: 16, md: 18, lg: 20 };

export function IconButton({
  icon,
  variant = 'ghost',
  size = 'md',
  active = false,
  disabled = false,
  label,
  style,
  ...rest
}) {
  const dim = SIZES[size] || SIZES.md;

  const variants = {
    ghost: { background: 'transparent', color: active ? 'var(--accent)' : 'var(--text-secondary)', borderColor: 'transparent' },
    solid: { background: 'var(--surface-card)', color: active ? 'var(--accent)' : 'var(--text-primary)', borderColor: 'var(--border-hairline)' },
    accent: { background: 'var(--accent)', color: 'var(--text-on-accent)', borderColor: 'transparent' },
  };

  const onEnter = (e) => {
    if (disabled) return;
    const el = e.currentTarget;
    if (variant === 'accent') el.style.background = 'var(--accent-press)';
    else { el.style.background = 'var(--surface-hover)'; el.style.color = active ? 'var(--accent)' : 'var(--text-primary)'; }
  };
  const onLeave = (e) => Object.assign(e.currentTarget.style, variants[variant]);

  return (
    <button
      type="button"
      disabled={disabled}
      aria-label={label}
      title={label}
      onMouseEnter={onEnter}
      onMouseLeave={onLeave}
      style={{
        display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
        width: dim, height: dim, flex: 'none',
        border: '1px solid', borderRadius: 'var(--radius-md)',
        cursor: disabled ? 'not-allowed' : 'pointer', opacity: disabled ? 0.45 : 1,
        transition: 'background var(--dur-fast) var(--ease-standard), color var(--dur-fast) var(--ease-standard)',
        ...variants[variant], ...style,
      }}
      {...rest}
    >
      <Icon name={icon} size={ICON[size]} />
    </button>
  );
}
