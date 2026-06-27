/** Compact status / count pill. Mono tone for issue counts; semantic tones for status. */
export interface BadgeProps extends React.HTMLAttributes<HTMLSpanElement> {
  tone?: 'neutral' | 'accent' | 'unread' | 'success' | 'warning' | 'danger';
  /** Use the mono data voice (issue counts, page totals). */
  mono?: boolean;
  /** Show a leading status dot. */
  dot?: boolean;
  children?: React.ReactNode;
}

export function Badge(props: BadgeProps): JSX.Element;
