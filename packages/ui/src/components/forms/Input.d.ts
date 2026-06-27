/** Text input on an ink fill with a cyan focus ring. */
export interface InputProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, 'size'> {
  /** Leading icon name (see Icon) — e.g. 'search'. */
  icon?: string;
  size?: 'sm' | 'md' | 'lg';
  /** Red border for validation errors. */
  invalid?: boolean;
}

export function Input(props: InputProps): JSX.Element;
