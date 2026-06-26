import type { ButtonHTMLAttributes } from 'react';

type Variant = 'primary' | 'ghost';

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant;
}

/**
 * Button — the first design-system primitive, wired to the Longbox tokens. Inline
 * styles here keep the package dependency-free for Phase 0; Claude Design will expand
 * this into the full component set (CoverCard, Rail, FilterBar, …).
 */
export function Button({ variant = 'primary', style, ...props }: ButtonProps) {
  const base = {
    font: '500 var(--ch-text-body)/1 var(--ch-font-body)',
    padding: '0.5rem 0.9rem',
    borderRadius: 'var(--ch-radius)',
    cursor: 'pointer',
    border: '1px solid transparent',
    transition: 'background 120ms ease, border-color 120ms ease',
  } as const;

  const variants: Record<Variant, React.CSSProperties> = {
    primary: {
      background: 'var(--ch-accent)',
      color: 'var(--ch-ink-900)',
    },
    ghost: {
      background: 'transparent',
      color: 'var(--ch-text)',
      borderColor: 'var(--ch-border)',
    },
  };

  return <button style={{ ...base, ...variants[variant], ...style }} {...props} />;
}
