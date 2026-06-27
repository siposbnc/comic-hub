import React from 'react';
import { Icon } from './Icon';

/**
 * ComicHub Button. Cyan-filled primary action; quiet ink secondary/ghost so it
 * never competes with cover art. Press = darken + 1px nudge, no bounce.
 */
const SIZES = {
  sm: { height: 30, padding: '0 12px', font: 'var(--text-small)', gap: 6, icon: 15 },
  md: { height: 38, padding: '0 16px', font: 'var(--text-body)', gap: 8, icon: 18 },
  lg: { height: 46, padding: '0 22px', font: 'var(--text-body)', gap: 9, icon: 20 },
};

export function Button({
  children,
  variant = 'primary',
  size = 'md',
  icon,
  iconRight,
  disabled = false,
  fullWidth = false,
  type = 'button',
  style,
  ...rest
}) {
  const s = SIZES[size] || SIZES.md;

  const base = {
    display: 'inline-flex',
    alignItems: 'center',
    justifyContent: 'center',
    gap: s.gap,
    height: s.height,
    padding: s.padding,
    width: fullWidth ? '100%' : undefined,
    fontFamily: 'var(--font-body)',
    fontSize: s.font,
    fontWeight: 'var(--weight-semibold)',
    lineHeight: 1,
    letterSpacing: '0.005em',
    border: '1px solid transparent',
    borderRadius: 'var(--radius-md)',
    cursor: disabled ? 'not-allowed' : 'pointer',
    opacity: disabled ? 0.45 : 1,
    transition: 'background var(--dur-fast) var(--ease-standard), border-color var(--dur-fast) var(--ease-standard), transform var(--dur-fast) var(--ease-standard)',
    userSelect: 'none',
    whiteSpace: 'nowrap',
  };

  const variants = {
    primary: { background: 'var(--accent)', color: 'var(--text-on-accent)' },
    secondary: { background: 'var(--surface-card)', color: 'var(--text-primary)', borderColor: 'var(--border-strong)' },
    ghost: { background: 'transparent', color: 'var(--text-secondary)' },
    danger: { background: 'var(--danger)', color: '#fff' },
  };

  const onDown = (e) => { if (!disabled) e.currentTarget.style.transform = 'translateY(1px)'; };
  const onUp = (e) => { e.currentTarget.style.transform = 'translateY(0)'; };
  const onEnter = (e) => {
    if (disabled) return;
    const el = e.currentTarget;
    if (variant === 'primary') el.style.background = 'var(--accent-press)';
    else if (variant === 'danger') el.style.background = 'var(--danger-500)', el.style.filter = 'brightness(1.08)';
    else if (variant === 'secondary') el.style.borderColor = 'var(--accent)';
    else el.style.background = 'var(--surface-card)', el.style.color = 'var(--text-primary)';
  };
  const onLeave = (e) => {
    const el = e.currentTarget;
    el.style.filter = '';
    Object.assign(el.style, {
      background: variants[variant].background,
      color: variants[variant].color,
      borderColor: variants[variant].borderColor || 'transparent',
    });
  };

  return (
    <button
      type={type}
      disabled={disabled}
      style={{ ...base, ...variants[variant], ...style }}
      onMouseDown={onDown}
      onMouseUp={onUp}
      onMouseEnter={onEnter}
      onMouseLeave={onLeave}
      {...rest}
    >
      {icon && <Icon name={icon} size={s.icon} />}
      {children}
      {iconRight && <Icon name={iconRight} size={s.icon} />}
    </button>
  );
}
