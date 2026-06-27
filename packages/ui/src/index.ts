// Barrel for the ComicHub design system. Components are synced verbatim from the
// Claude Design project (JSX + sibling .d.ts); see README.md. Apps import from here.

// core
export { Icon } from './components/core/Icon';
export type { IconProps } from './components/core/Icon';
export { Button } from './components/core/Button';
export type { ButtonProps } from './components/core/Button';
export { IconButton } from './components/core/IconButton';
export type { IconButtonProps } from './components/core/IconButton';
export { Badge } from './components/core/Badge';
export type { BadgeProps } from './components/core/Badge';
export { Tag } from './components/core/Tag';
export type { TagProps } from './components/core/Tag';

// comic
export { CoverCard } from './components/comic/CoverCard';
export type { CoverCardProps } from './components/comic/CoverCard';
export { Rail } from './components/comic/Rail';
export type { RailProps } from './components/comic/Rail';
export { ProgressBar } from './components/comic/ProgressBar';
export type { ProgressBarProps } from './components/comic/ProgressBar';
export { JobIndicator } from './components/comic/JobIndicator';
export type { JobIndicatorProps, JobItem } from './components/comic/JobIndicator';

// navigation
export { SidebarItem, SidebarSection } from './components/navigation/SidebarItem';
export type { SidebarItemProps, SidebarSectionProps } from './components/navigation/SidebarItem';
export { Tabs } from './components/navigation/Tabs';
export type { TabsProps, TabItem } from './components/navigation/Tabs';

// feedback
export { EmptyState } from './components/feedback/EmptyState';
export type { EmptyStateProps } from './components/feedback/EmptyState';
export { Toast } from './components/feedback/Toast';
export type { ToastProps } from './components/feedback/Toast';
export { Tooltip } from './components/feedback/Tooltip';
export type { TooltipProps } from './components/feedback/Tooltip';

// forms
export { Input } from './components/forms/Input';
export type { InputProps } from './components/forms/Input';
export { Slider } from './components/forms/Slider';
export type { SliderProps } from './components/forms/Slider';
