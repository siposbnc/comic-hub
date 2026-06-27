/** Native select styled to match Input — ink fill, chevron, cyan focus ring. */
export interface SelectProps extends Omit<React.SelectHTMLAttributes<HTMLSelectElement>, 'size'> {
  size?: 'sm' | 'md' | 'lg';
  children?: React.ReactNode;
}

export function Select(props: SelectProps): JSX.Element;
