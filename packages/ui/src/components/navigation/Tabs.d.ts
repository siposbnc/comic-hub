/** Underline-style tab nav. Active tab gets a cyan underline. Optional mono count per tab. */
export interface TabItem {
  value: string;
  label: React.ReactNode;
  /** Optional mono count badge (e.g. unread issues). */
  count?: number;
}

export interface TabsProps {
  /** Tab list — strings or {value,label,count} objects. */
  tabs: (string | TabItem)[];
  value: string;
  onChange?: (value: string) => void;
  style?: React.CSSProperties;
}

export function Tabs(props: TabsProps): JSX.Element;
