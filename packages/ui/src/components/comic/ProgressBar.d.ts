/** Cyan fill on an ink track. Series progress, scan/job progress, match confidence. */
export interface ProgressBarProps {
  value?: number;
  max?: number;
  /** Optional left-aligned label. */
  label?: React.ReactNode;
  tone?: 'accent' | 'unread' | 'success';
  height?: number;
  /** Show a mono "value / max" count on the right. */
  showCount?: boolean;
  style?: React.CSSProperties;
}

export function ProgressBar(props: ProgressBarProps): JSX.Element;
