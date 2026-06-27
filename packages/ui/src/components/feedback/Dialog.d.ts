/** Modal dialog — an ink-raised card over a blurred scrim. Confirmations, edit forms, match picker. */
export interface DialogProps {
  open?: boolean;
  title?: React.ReactNode;
  children?: React.ReactNode;
  /** Footer actions, typically a pair of <Button>s. */
  footer?: React.ReactNode;
  onClose?: () => void;
  width?: number;
}

export function Dialog(props: DialogProps): JSX.Element | null;
