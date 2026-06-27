/** Binary on/off toggle with a cyan track. Settings switches (watch folder, theme). */
export interface SwitchProps {
  checked?: boolean;
  onChange?: (next: boolean) => void;
  label?: React.ReactNode;
  disabled?: boolean;
  style?: React.CSSProperties;
}

export function Switch(props: SwitchProps): JSX.Element;
