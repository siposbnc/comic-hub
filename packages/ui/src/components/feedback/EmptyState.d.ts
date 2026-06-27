/** Empty state — the one place the halftone print texture appears. Always invites action. */
export interface EmptyStateProps {
  title?: React.ReactNode;
  children?: React.ReactNode;
  /** Primary action(s) — usually a single <Button>. */
  action?: React.ReactNode;
  style?: React.CSSProperties;
}

export function EmptyState(props: EmptyStateProps): JSX.Element;
