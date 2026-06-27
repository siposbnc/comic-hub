/** Top-bar scan/job status pill with a per-job progress popover. Non-blocking. */
export interface JobItem {
  name: string;
  /** 0..1 progress. */
  progress: number;
}

export interface JobIndicatorProps {
  status?: 'idle' | 'scanning' | 'error';
  /** Override the pill label. */
  label?: React.ReactNode;
  /** Per-job rows shown in the popover. */
  jobs?: JobItem[];
  style?: React.CSSProperties;
}

export function JobIndicator(props: JobIndicatorProps): JSX.Element;
