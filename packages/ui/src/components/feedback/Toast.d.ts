/** Transient notification with a colored status edge. Errors stay concrete and actionable. */
export interface ToastProps {
  tone?: 'info' | 'success' | 'warning' | 'danger';
  title?: React.ReactNode;
  children?: React.ReactNode;
  /** Action buttons (e.g. Skip / Show details). */
  action?: React.ReactNode;
  onClose?: () => void;
  style?: React.CSSProperties;
}

export function Toast(props: ToastProps): JSX.Element;
