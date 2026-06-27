/** Checkbox — ink box, cyan fill + check when on. Multiselect and field-lock toggles. */
export interface CheckboxProps {
  checked?: boolean;
  onChange?: (next: boolean) => void;
  label?: React.ReactNode;
  /** Mixed state (e.g. partial selection) — shows a dash. */
  indeterminate?: boolean;
  disabled?: boolean;
  style?: React.CSSProperties;
}

export function Checkbox(props: CheckboxProps): JSX.Element;
