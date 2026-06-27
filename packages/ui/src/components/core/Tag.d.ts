/** Removable free-form label (genre, character, mood). */
export interface TagProps extends React.HTMLAttributes<HTMLSpanElement> {
  removable?: boolean;
  onRemove?: () => void;
  /** Cyan-outlined for a selected/active facet. */
  accent?: boolean;
  children?: React.ReactNode;
}

export function Tag(props: TagProps): JSX.Element;
