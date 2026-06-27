import { forwardRef, type ButtonHTMLAttributes } from 'react';
import { Icon, type IconName } from './Icon.js';

export interface IconButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  icon: IconName;
  /** Required for accessibility — also surfaced as the tooltip text. */
  label: string;
  /** Optional keyboard hint shown in the tooltip (e.g. "D"). */
  hint?: string;
  size?: number;
  active?: boolean;
}

/**
 * IconButton — toolbar control with an accessible label, hover/focus/active states, and a
 * built-in tooltip (title-driven via the .tooltip-host CSS). Local primitive; ideally lives
 * in @comichub/ui once that package gains icon controls.
 */
export const IconButton = forwardRef<HTMLButtonElement, IconButtonProps>(function IconButton(
  { icon, label, hint, size = 20, active = false, className, ...props },
  ref,
) {
  const tip = hint ? `${label} · ${hint}` : label;
  return (
    <button
      ref={ref}
      type="button"
      className={`icon-btn${active ? ' is-active' : ''}${className ? ` ${className}` : ''}`}
      aria-label={label}
      aria-pressed={active || undefined}
      data-tip={tip}
      {...props}
    >
      <Icon name={icon} size={size} />
    </button>
  );
});
