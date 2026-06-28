/**
 * CoverCard — the atom of the whole ComicHub UI. A square-cornered comic cover with
 * the print signatures: registration ticks on hover, a mono spine tab (issue number +
 * read state), a progress underline, and optional multiselect.
 */
export interface CoverCardProps {
  /** Cover image URL. Falls back to a masthead of the title. */
  cover?: string;
  title?: string;
  /** Secondary line (series, publisher) — rendered in the mono data voice. */
  subtitle?: string;
  /** Issue/volume number shown in the spine tab, e.g. "#012". */
  number?: string;
  /** Read state — drives the spine-tab color. */
  status?: 'unread' | 'reading' | 'read';
  /** 0..1 reading progress; shows the underline when status is 'reading'. */
  progress?: number;
  size?: 's' | 'm' | 'l';
  selectable?: boolean;
  selected?: boolean;
  onSelect?: (next: boolean) => void;
  onClick?: () => void;
  style?: React.CSSProperties;
}

/**
 * @startingPoint section="Comic" subtitle="The cover atom — spine tab, ticks, progress" viewport="240x420"
 */
export function CoverCard(props: CoverCardProps): JSX.Element;
