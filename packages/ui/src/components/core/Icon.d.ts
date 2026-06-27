/** A Lucide-style line icon, 1.5px stroke, square-ish. ComicHub's only icon system. */
export interface IconProps {
  /** Icon name from the registry (e.g. 'home', 'search', 'book-open', 'settings'). */
  name:
    | 'home' | 'library' | 'list' | 'layers' | 'collection' | 'bookmark' | 'stats' | 'settings'
    | 'search' | 'x' | 'check' | 'plus' | 'minus' | 'chevron-right' | 'chevron-left' | 'chevron-down'
    | 'more-horizontal' | 'book-open' | 'filter' | 'sort' | 'grid' | 'columns' | 'sun' | 'moon'
    | 'alert-triangle' | 'info' | 'trash' | 'edit' | 'star' | 'folder' | 'refresh' | 'user'
    | 'clock' | 'download' | 'link' | 'maximize';
  /** Pixel size (width = height). Default 18. */
  size?: number;
  /** Stroke color. Default currentColor — inherits text color. */
  color?: string;
  /** Stroke weight. Default 1.5 (brand standard). */
  strokeWidth?: number;
  style?: React.CSSProperties;
}

export function Icon(props: IconProps): JSX.Element | null;
