/** User avatar — initials on a tinted chip, or an image. For the household user switcher. */
export interface AvatarProps extends React.HTMLAttributes<HTMLSpanElement> {
  name?: string;
  /** Optional image URL; falls back to initials. */
  src?: string;
  size?: 'sm' | 'md' | 'lg';
  /** Override tint color (defaults to cyan accent). */
  accent?: string;
}

export function Avatar(props: AvatarProps): JSX.Element;
