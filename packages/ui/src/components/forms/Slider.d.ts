/** Range slider with a cyan fill track. Cover density, brightness, prefetch depth. */
export interface SliderProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, 'onChange' | 'value'> {
  value?: number;
  min?: number;
  max?: number;
  step?: number;
  onChange?: (value: number) => void;
  disabled?: boolean;
}

export function Slider(props: SliderProps): JSX.Element;
