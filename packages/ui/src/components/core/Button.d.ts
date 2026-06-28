/** ComicHub's primary action button — cyan fill for primary, quiet ink for the rest. */
export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  /** Visual weight. `primary` is the cyan spot color; use it once per view. */
  variant?: 'primary' | 'secondary' | 'ghost' | 'danger';
  size?: 'sm' | 'md' | 'lg';
  /** Leading icon name (see Icon). */
  icon?: string;
  /** Trailing icon name. */
  iconRight?: string;
  fullWidth?: boolean;
  disabled?: boolean;
  children?: React.ReactNode;
}

/**
 * @startingPoint section="Core" subtitle="Buttons — primary, secondary, ghost, danger" viewport="700x200"
 */
export function Button(props: ButtonProps): JSX.Element;
