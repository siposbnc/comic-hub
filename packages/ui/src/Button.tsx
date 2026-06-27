import type { ButtonHTMLAttributes } from 'react';

type Variant = 'primary' | 'ghost';

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant;
}

/**
 * Button — a thin design-system primitive wired to the Longbox tokens (synced from
 * the ComicHub Design System). The full component set (CoverCard, Rail, IconButton,
 * …) is pulled in incrementally as screens consume it; see packages/ui/README.md.
 */
export function Button({ variant = 'primary', style, ...props }: ButtonProps) {
  const base = {
    font: '500 var(--text-body)/1 var(--font-body)',
    padding: '0.5rem 0.9rem',
    borderRadius: 'var(--radius-md)',
    cursor: 'pointer',
    border: '1px solid transparent',
    transition:
      'background var(--dur-fast) var(--ease-standard), border-color var(--dur-fast) var(--ease-standard)',
  } as const;

  const variants: Record<Variant, React.CSSProperties> = {
    primary: {
      background: 'var(--accent)',
      color: 'var(--text-on-accent)',
    },
    ghost: {
      background: 'transparent',
      color: 'var(--text-primary)',
      borderColor: 'var(--border-hairline)',
    },
  };

  return <button style={{ ...base, ...variants[variant], ...style }} {...props} />;
}
