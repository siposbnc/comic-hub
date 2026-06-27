/** Square icon-only button for toolbars, card overflow, and close affordances. */
export interface IconButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  /** Icon name (see Icon). */
  icon: string;
  variant?: 'ghost' | 'solid' | 'accent';
  size?: 'sm' | 'md' | 'lg';
  /** Active/selected — tints the icon cyan. */
  active?: boolean;
  /** Accessible label (also the tooltip title). */
  label?: string;
  disabled?: boolean;
}

export function IconButton(props: IconButtonProps): JSX.Element;
