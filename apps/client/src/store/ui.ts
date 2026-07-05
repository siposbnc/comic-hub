import { create } from 'zustand';

export type Theme = 'dark' | 'light';
export type Accent = 'cyan' | 'magenta' | 'amber';
/** Library cover grid density: small or medium covers. */
export type Density = 's' | 'm';

export interface ToastEntry {
  id: string;
  tone: 'info' | 'success' | 'warning' | 'danger';
  title: string;
  message?: string;
}

/** A live job as tracked from the WS `jobs` topic, shown in the top-bar JobIndicator. */
export interface TrackedJob {
  id: string;
  type: string;
  state: 'queued' | 'running' | 'done' | 'failed' | 'canceled';
  progress: number;
  total: number;
  done: number;
  error?: string;
}

interface UiState {
  theme: Theme;
  accent: Accent;
  density: Density;
  /** Whether reading lists reveal already-read issues (off = collapsed to the current spot). */
  showReadInLists: boolean;
  search: string;
  jobs: Record<string, TrackedJob>;
  toasts: ToastEntry[];
  setTheme: (theme: Theme) => void;
  toggleTheme: () => void;
  setAccent: (accent: Accent) => void;
  setDensity: (density: Density) => void;
  setShowReadInLists: (show: boolean) => void;
  setSearch: (q: string) => void;
  upsertJob: (job: TrackedJob) => void;
  clearFinishedJobs: () => void;
  addToast: (toast: Omit<ToastEntry, 'id'>) => string;
  dismissToast: (id: string) => void;
}

const THEME_KEY = 'comichub.theme';
const ACCENT_KEY = 'comichub.accent';
const DENSITY_KEY = 'comichub.density';
const SHOW_READ_KEY = 'comichub.showReadInLists';

function initialTheme(): Theme {
  if (typeof localStorage !== 'undefined') {
    const saved = localStorage.getItem(THEME_KEY);
    if (saved === 'light' || saved === 'dark') return saved;
  }
  return 'dark';
}

function initialAccent(): Accent {
  if (typeof localStorage !== 'undefined') {
    const saved = localStorage.getItem(ACCENT_KEY);
    if (saved === 'cyan' || saved === 'magenta' || saved === 'amber') return saved;
  }
  return 'cyan';
}

function initialDensity(): Density {
  if (typeof localStorage !== 'undefined') {
    const saved = localStorage.getItem(DENSITY_KEY);
    if (saved === 's' || saved === 'm') return saved;
  }
  return 'm';
}

function initialShowReadInLists(): boolean {
  if (typeof localStorage !== 'undefined') {
    return localStorage.getItem(SHOW_READ_KEY) === '1';
  }
  return false;
}

/** Reflects the theme onto the document so the design tokens (`[data-theme]`) switch. */
export function applyTheme(theme: Theme): void {
  if (typeof document !== 'undefined') {
    document.documentElement.setAttribute('data-theme', theme);
  }
}

/** Reflects the accent onto the document (`[data-accent]`); cyan is the default (no attr). */
export function applyAccent(accent: Accent): void {
  if (typeof document !== 'undefined') {
    if (accent === 'cyan') document.documentElement.removeAttribute('data-accent');
    else document.documentElement.setAttribute('data-accent', accent);
  }
}

let toastSeq = 0;

export const useUiStore = create<UiState>((set) => ({
  theme: initialTheme(),
  accent: initialAccent(),
  density: initialDensity(),
  showReadInLists: initialShowReadInLists(),
  search: '',
  jobs: {},
  toasts: [],
  setTheme: (theme) => {
    applyTheme(theme);
    if (typeof localStorage !== 'undefined') localStorage.setItem(THEME_KEY, theme);
    set({ theme });
  },
  toggleTheme: () =>
    set((s) => {
      const next: Theme = s.theme === 'dark' ? 'light' : 'dark';
      applyTheme(next);
      if (typeof localStorage !== 'undefined') localStorage.setItem(THEME_KEY, next);
      return { theme: next };
    }),
  setAccent: (accent) => {
    applyAccent(accent);
    if (typeof localStorage !== 'undefined') localStorage.setItem(ACCENT_KEY, accent);
    set({ accent });
  },
  setDensity: (density) => {
    if (typeof localStorage !== 'undefined') localStorage.setItem(DENSITY_KEY, density);
    set({ density });
  },
  setShowReadInLists: (show) => {
    if (typeof localStorage !== 'undefined') localStorage.setItem(SHOW_READ_KEY, show ? '1' : '0');
    set({ showReadInLists: show });
  },
  setSearch: (search) => set({ search }),
  upsertJob: (job) => set((s) => ({ jobs: { ...s.jobs, [job.id]: job } })),
  clearFinishedJobs: () =>
    set((s) => {
      const jobs: Record<string, TrackedJob> = {};
      for (const [id, j] of Object.entries(s.jobs)) {
        if (j.state === 'queued' || j.state === 'running') jobs[id] = j;
      }
      return { jobs };
    }),
  addToast: (toast) => {
    const id = `t${++toastSeq}`;
    set((s) => ({ toasts: [...s.toasts, { ...toast, id }] }));
    return id;
  },
  dismissToast: (id) => set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) })),
}));
