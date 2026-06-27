/** A sidebar nav row — icon + label, optional mono count, cyan when active. */
export interface SidebarItemProps {
  icon?: string;
  label: React.ReactNode;
  /** Optional mono count on the right (e.g. unread issues). */
  count?: number | string;
  active?: boolean;
  onClick?: () => void;
  style?: React.CSSProperties;
}

export function SidebarItem(props: SidebarItemProps): JSX.Element;

/** A mono section label that separates sidebar groups (e.g. "Libraries"). */
export interface SidebarSectionProps {
  children?: React.ReactNode;
  style?: React.CSSProperties;
}

export function SidebarSection(props: SidebarSectionProps): JSX.Element;
