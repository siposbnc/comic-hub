import { create } from './createStore.js';
import {
  DEFAULT_PREFETCH_AHEAD,
  DEFAULT_PREFETCH_BEHIND,
  type Manifest,
  type PageProvider,
  type ReadingDirection,
} from '@comichub/reader-core';
import { PageCache } from './PageCache.js';
import { ThumbCache } from './ThumbCache.js';
import { buildSpreads, spreadIndexOf, type Spread } from './layout.js';
import {
  DEFAULT_SETTINGS,
  type FitMode,
  type LayoutMode,
  type ReaderBackground,
  type ReaderSettings,
} from './types.js';
import { resolveLaunch } from '../providers/launch.js';

const FIT_CYCLE: FitMode[] = ['screen', 'width', 'height', 'original', 'smart'];
const MIN_ZOOM = 1;
const MAX_ZOOM = 5;
const ZOOM_STEP = 0.25;
const PAGE_CACHE_CAPACITY = 16;
const SAVE_DEBOUNCE_MS = 1500;

type Status = 'idle' | 'loading' | 'ready' | 'error';
type Mode = 'connected' | 'standalone' | 'empty';

export interface ReaderState {
  status: Status;
  error?: string;
  mode: Mode;
  title?: string;

  manifest: Manifest | null;
  spreads: Spread[];
  currentPage: number;
  finished: boolean;
  resumePage: number | null;

  settings: ReaderSettings;

  zoom: number;
  panX: number;
  panY: number;

  chromeVisible: boolean;

  provider: PageProvider | null;
  pages: PageCache | null;
  thumbs: ThumbCache | null;

  init: () => Promise<void>;
  retry: () => void;
  goToPage: (page: number) => void;
  next: () => void;
  prev: () => void;
  resume: () => void;
  startOver: () => void;
  dismissResume: () => void;
  dismissFinished: () => void;

  setLayout: (layout: LayoutMode) => void;
  toggleLayout: () => void;
  setFit: (fit: FitMode) => void;
  cycleFit: () => void;
  setDirection: (dir: ReadingDirection) => void;
  toggleDirection: () => void;
  setBackground: (bg: ReaderBackground) => void;

  setZoom: (zoom: number) => void;
  zoomBy: (delta: number) => void;
  resetZoom: () => void;
  setPan: (x: number, y: number) => void;
  toggleZoomFit: () => void;

  showChrome: () => void;
  hideChrome: () => void;
  toggleChrome: () => void;

  flushProgress: () => void;
  dispose: () => void;
}

export const useReaderStore = create<ReaderState>((set, get) => {
  let saveTimer: ReturnType<typeof setTimeout> | null = null;

  function clamp(value: number, min: number, max: number): number {
    return Math.min(max, Math.max(min, value));
  }

  function refreshWindow(): void {
    const { pages, currentPage, manifest, settings } = get();
    if (!pages || !manifest) return;
    const ahead = settings.layout === 'double' ? DEFAULT_PREFETCH_AHEAD * 2 : DEFAULT_PREFETCH_AHEAD;
    pages.setWindow(currentPage, ahead, DEFAULT_PREFETCH_BEHIND, manifest.pageCount);
  }

  function scheduleSave(): void {
    if (saveTimer) clearTimeout(saveTimer);
    saveTimer = setTimeout(() => {
      saveTimer = null;
      get().flushProgress();
    }, SAVE_DEBOUNCE_MS);
  }

  function recomputeSpreads(): void {
    const { manifest, settings } = get();
    if (!manifest) return;
    set({ spreads: buildSpreads(manifest, settings.layout, settings.coverAlone) });
  }

  /** Moves to a page, snapping to the start of its spread; resets zoom/pan. */
  function navigateTo(page: number): void {
    const { manifest, spreads } = get();
    if (!manifest) return;
    const clamped = clamp(page, 0, manifest.pageCount - 1);
    const spreadIdx = spreadIndexOf(spreads, clamped);
    const start = spreads[spreadIdx]?.[0] ?? clamped;
    const lastSpread = spreadIdx === spreads.length - 1;
    set({
      currentPage: start,
      finished: lastSpread,
      zoom: MIN_ZOOM,
      panX: 0,
      panY: 0,
    });
    refreshWindow();
    scheduleSave();
  }

  return {
    status: 'idle',
    mode: 'empty',
    manifest: null,
    spreads: [],
    currentPage: 0,
    finished: false,
    resumePage: null,
    settings: { ...DEFAULT_SETTINGS },
    zoom: MIN_ZOOM,
    panX: 0,
    panY: 0,
    chromeVisible: true,
    provider: null,
    pages: null,
    thumbs: null,

    init: async () => {
      set({ status: 'loading' });
      try {
        const launch = await resolveLaunch();
        if (launch.kind === 'empty') {
          set({ status: 'idle', mode: 'empty' });
          return;
        }
        if (launch.kind === 'error') {
          set({ status: 'error', mode: 'empty', error: launch.message });
          return;
        }

        const provider = launch.provider;
        const manifest =
          launch.kind === 'standalone' ? launch.manifest : await provider.manifest();

        const pages = new PageCache(provider, PAGE_CACHE_CAPACITY);
        const thumbs = new ThumbCache(provider);
        const direction = manifest.readingDir ?? DEFAULT_SETTINGS.direction;
        const settings: ReaderSettings = { ...DEFAULT_SETTINGS, direction };
        const spreads = buildSpreads(manifest, settings.layout, settings.coverAlone);

        // Decide the entry page: deep-link page wins, else restored progress.
        let startPage = launch.kind === 'connected' ? (launch.startPage ?? 0) : 0;
        let resumePage: number | null = null;
        const restored = await provider.restoreProgress();
        if ((launch.kind !== 'connected' || launch.startPage == null) && restored) {
          if (restored.page > 0 && restored.page < manifest.pageCount) {
            startPage = restored.page;
            resumePage = restored.page;
          }
        }

        set({
          status: 'ready',
          mode: launch.kind,
          title: launch.title,
          manifest,
          spreads,
          settings,
          provider,
          pages,
          thumbs,
          resumePage,
        });
        navigateTo(startPage);
      } catch (err) {
        set({
          status: 'error',
          error: err instanceof Error ? err.message : 'Failed to open this comic.',
        });
      }
    },

    retry: () => {
      void get().init();
    },

    goToPage: (page) => navigateTo(page),

    next: () => {
      const { spreads, currentPage, manifest } = get();
      if (!manifest) return;
      const idx = spreadIndexOf(spreads, currentPage);
      const target = spreads[idx + 1];
      if (idx >= spreads.length - 1 || !target) {
        set({ finished: true });
        get().flushProgress();
        return;
      }
      navigateTo(target[0] ?? currentPage);
    },

    prev: () => {
      const { spreads, currentPage } = get();
      const idx = spreadIndexOf(spreads, currentPage);
      if (idx <= 0) return;
      const target = spreads[idx - 1];
      if (target) navigateTo(target[0] ?? currentPage);
    },

    resume: () => {
      const { resumePage } = get();
      set({ resumePage: null });
      if (resumePage != null) navigateTo(resumePage);
    },

    startOver: () => {
      set({ resumePage: null });
      navigateTo(0);
    },

    dismissResume: () => set({ resumePage: null }),

    dismissFinished: () => set({ finished: false }),

    setLayout: (layout) => {
      set((s) => ({ settings: { ...s.settings, layout } }));
      recomputeSpreads();
      navigateTo(get().currentPage);
    },

    toggleLayout: () => {
      const next = get().settings.layout === 'single' ? 'double' : 'single';
      get().setLayout(next);
    },

    setFit: (fit) => set((s) => ({ settings: { ...s.settings, fit }, zoom: MIN_ZOOM, panX: 0, panY: 0 })),

    cycleFit: () => {
      const current = get().settings.fit;
      const idx = FIT_CYCLE.indexOf(current);
      get().setFit(FIT_CYCLE[(idx + 1) % FIT_CYCLE.length] ?? 'screen');
    },

    setDirection: (direction) => set((s) => ({ settings: { ...s.settings, direction } })),

    toggleDirection: () => {
      const next = get().settings.direction === 'ltr' ? 'rtl' : 'ltr';
      get().setDirection(next);
    },

    setBackground: (background) => set((s) => ({ settings: { ...s.settings, background } })),

    setZoom: (zoom) => {
      const z = clamp(zoom, MIN_ZOOM, MAX_ZOOM);
      set({ zoom: z, ...(z === MIN_ZOOM ? { panX: 0, panY: 0 } : {}) });
    },

    zoomBy: (delta) => get().setZoom(get().zoom + delta * ZOOM_STEP),

    resetZoom: () => set({ zoom: MIN_ZOOM, panX: 0, panY: 0 }),

    setPan: (x, y) => set({ panX: x, panY: y }),

    toggleZoomFit: () => {
      const { zoom } = get();
      if (zoom > MIN_ZOOM) {
        set({ zoom: MIN_ZOOM, panX: 0, panY: 0 });
      } else {
        set({ zoom: 2 });
      }
    },

    showChrome: () => set({ chromeVisible: true }),
    hideChrome: () => {
      if (!get().resumePage) set({ chromeVisible: false });
    },
    toggleChrome: () => set((s) => ({ chromeVisible: !s.chromeVisible })),

    flushProgress: () => {
      const { provider, manifest, currentPage, finished } = get();
      if (!provider || !manifest) return;
      provider.saveProgress({
        bookId: manifest.bookId,
        page: currentPage,
        status: finished ? 'read' : 'in_progress',
        updatedAt: Date.now(),
      });
    },

    dispose: () => {
      if (saveTimer) {
        clearTimeout(saveTimer);
        saveTimer = null;
      }
      get().flushProgress();
      get().pages?.dispose();
      get().thumbs?.dispose();
      set({ provider: null, pages: null, thumbs: null, status: 'idle' });
    },
  };
});
