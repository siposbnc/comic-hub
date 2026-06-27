/** Horizontally-scrolling row of CoverCards under a mono section label. The Home/Discover block. */
export interface RailProps {
  /** Mono uppercase section label (e.g. "Continue Reading"). */
  label: React.ReactNode;
  /** Optional trailing link, e.g. { label: 'See all', onClick }. */
  action?: { label: React.ReactNode; onClick?: () => void };
  /** A row of <CoverCard>s. */
  children?: React.ReactNode;
  style?: React.CSSProperties;
}

export function Rail(props: RailProps): JSX.Element;
