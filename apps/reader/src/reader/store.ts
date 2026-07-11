import { create } from 'zustand';
import {
  DEFAULT_PREFETCH_AHEAD,
  DEFAULT_PREFETCH_BEHIND,
  type Manifest,
  type PageProvider,
  type ReadingDirection,
} from '@comichub/reader-core';
import { PageCache } from './PageCache.js';
import { ThumbCache } from './ThumbCache.js';
import { isFullscreen, setFullscreen } from './fullscreen.js';
import { buildSpreads, spreadIndexOf, type Spread } from './layout.js';
import {
  DEFAULT_SETTINGS,
  type FitMode,
  type LayoutMode,
  type ReaderBackground,
  type ReaderSettings,
} from './types.js';
import { invoke } from '@tauri-apps/api/core';
import type { Bookmark, ComicHubClient } from '@comichub/api-client';
import {
  resolveLaunch,
  resolveLaunchFromPath,
  connectedLaunch,
  type LaunchResult,
} from '../providers/launch.js';
import {
  loadConfig,
  saveConfig,
  localPrefs,
  type AutoAdvance,
  type PrefsBackend,
  type ReaderConfig,
  type SyncMode,
} from './prefs.js';

const PREFS_DEBOUNCE_MS = 600;

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
  /** App-level reader config (persisted in localStorage). */
  config: ReaderConfig;
  /** Server-backed per-book settings store, when in connected mode. */
  prefsServer: PrefsBackend | null;
  /** The connected client, used to resolve/open the next issue (connected mode only). */
  serverClient: ComicHubClient | null;
  /** The next issue offered at the end of this one (when auto-advance is on). */
  nextBook: { id: string; label: string } | null;

  zoom: number;
  panX: number;
  panY: number;

  chromeVisible: boolean;
  settingsOpen: boolean;
  /** Whether the OS window is fullscreen. The window-state plugin persists this across
   *  launches; it is mirrored here so the toolbar/keyboard can reflect and toggle it. */
  fullscreen: boolean;

  /** Continuous auto-scroll running (transient; forced off outside continuous mode). */
  autoScroll: boolean;
  /** Auto-scroll momentarily paused while a navigation key is held. */
  autoScrollPaused: boolean;

  /** Bookmarks for the open book (connected mode only); ordered by page ascending. */
  bookmarks: Bookmark[];
  bookmarksOpen: boolean;
  /** Transient confirmation after a bookmark toggle (auto-clears). */
  bmToast: string | null;

  provider: PageProvider | null;
  pages: PageCache | null;
  thumbs: ThumbCache | null;

  init: (url?: string) => Promise<void>;
  /** Re-open a book from a deep-link URL while the reader is already running. */
  openUrl: (url: string) => void;
  /** Prompt for a comic file on disk and open it (standalone mode). */
  openFile: () => Promise<void>;
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
  /** Switch between continuous (webtoon) scroll and paged single mode. */
  toggleContinuous: () => void;
  setFit: (fit: FitMode) => void;
  cycleFit: () => void;
  setDirection: (dir: ReadingDirection) => void;
  toggleDirection: () => void;
  setBackground: (bg: ReaderBackground) => void;
  setCoverAlone: (on: boolean) => void;
  /** Toggle remembering settings per book; persisted to config. */
  setRememberPerBook: (on: boolean) => void;
  /** Choose where per-book overrides are stored (local vs server). */
  setSyncMode: (mode: SyncMode) => void;
  /** Set auto-advance behavior on completion. */
  setAutoAdvance: (mode: AutoAdvance) => void;
  /** Resolve the next issue to offer at the end (connected + auto-advance on). */
  fetchNext: () => Promise<void>;
  /** Open the offered next issue in place. */
  loadNext: () => void;
  /** Forget any offered next issue. */
  clearNext: () => void;

  setZoom: (zoom: number) => void;
  zoomBy: (delta: number) => void;
  resetZoom: () => void;
  setPan: (x: number, y: number) => void;
  toggleZoomFit: () => void;

  showChrome: () => void;
  hideChrome: () => void;
  toggleChrome: () => void;
  setSettingsOpen: (open: boolean) => void;

  /** Toggle OS-window fullscreen (persisted across launches by the window-state plugin). */
  toggleFullscreen: () => void;
  /** Read the window's actual fullscreen state into the store (e.g. after the window-state
   *  plugin restored a fullscreen launch), so the toolbar icon matches reality. */
  syncFullscreen: () => void;

  /** Toggle continuous auto-scroll (switches to continuous layout if needed). */
  toggleAutoScroll: () => void;
  /** Momentarily pause/resume auto-scroll (held navigation key). */
  setAutoScrollPaused: (paused: boolean) => void;
  /** Set the auto-scroll speed (CSS px/sec), persisted to config. */
  setAutoScrollSpeed: (pxPerSec: number) => void;

  /** Load the open book's bookmarks from the server (no-op outside connected mode). */
  loadBookmarks: () => Promise<void>;
  /** Bookmark the current page (add-only; removal is explicit, from the list). */
  bookmarkCurrentPage: () => Promise<void>;
  /** Replace a bookmark's note. */
  updateBookmarkNote: (id: string, note: string) => Promise<void>;
  /** Remove a bookmark by id. */
  removeBookmark: (id: string) => Promise<void>;
  setBookmarksOpen: (open: boolean) => void;

  flushProgress: () => void;
  dispose: () => void;
}

const BM_TOAST_MS = 1900;

/** Display page label (store pages are 0-indexed; the UI shows 1-indexed). */
function pageLabel(page: number): string {
  return `p.${String(page + 1).padStart(2, '0')}`;
}

export const useReaderStore = create<ReaderState>()((set, get) => {
  let saveTimer: ReturnType<typeof setTimeout> | null = null;
  let prefsTimer: ReturnType<typeof setTimeout> | null = null;
  let bmToastTimer: ReturnType<typeof setTimeout> | null = null;

  /** Show a transient bookmark confirmation toast. */
  function flashBookmark(message: string): void {
    if (bmToastTimer) clearTimeout(bmToastTimer);
    set({ bmToast: message });
    bmToastTimer = setTimeout(() => {
      bmToastTimer = null;
      set({ bmToast: null });
    }, BM_TOAST_MS);
  }

  /** The active per-book settings backend per the chosen sync mode. */
  function activePrefs(): PrefsBackend {
    const { config, prefsServer } = get();
    return config.syncMode === 'server' && prefsServer ? prefsServer : localPrefs;
  }

  /** Persist the current settings for the open book (debounced), when remembering is on. */
  function persistPrefs(): void {
    if (!get().config.rememberPerBook) return;
    if (prefsTimer) clearTimeout(prefsTimer);
    prefsTimer = setTimeout(() => {
      prefsTimer = null;
      const { manifest, settings } = get();
      if (manifest) void activePrefs().save(manifest.bookId, settings);
    }, PREFS_DEBOUNCE_MS);
  }

  /** Capture the current settings as the global defaults, so a chosen fit/layout/etc. sticks
   *  across issues even when per-book remembering is off. */
  function saveDefaults(): void {
    const config = { ...get().config, defaults: { ...get().settings } };
    set({ config });
    saveConfig(config);
  }

  function clamp(value: number, min: number, max: number): number {
    return Math.min(max, Math.max(min, value));
  }

  function refreshWindow(): void {
    const { pages, currentPage, manifest, settings } = get();
    if (!pages || !manifest) return;
    const ahead =
      settings.layout === 'double' ? DEFAULT_PREFETCH_AHEAD * 2 : DEFAULT_PREFETCH_AHEAD;
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

  /** True once the reader has a place worth recording: past the first page, or finished
   *  (a single-page book opens already finished). Opening and closing on the first page
   *  therefore never marks a book "in progress". */
  function persistsPlace(): boolean {
    const { currentPage, finished } = get();
    return currentPage > 0 || finished;
  }

  /** Saves the current book's place and tears down its caches before swapping books. */
  function teardownCurrent(): void {
    const { provider, manifest, currentPage, finished } = get();
    if (provider && manifest && persistsPlace()) {
      provider.saveProgress({
        bookId: manifest.bookId,
        page: currentPage,
        status: finished ? 'read' : 'in_progress',
        updatedAt: Date.now(),
      });
    }
    if (saveTimer) {
      clearTimeout(saveTimer);
      saveTimer = null;
    }
    get().pages?.dispose();
    get().thumbs?.dispose();
    set({
      provider: null,
      pages: null,
      thumbs: null,
      finished: false,
      resumePage: null,
      nextBook: null,
      bookmarks: [],
      bookmarksOpen: false,
      autoScroll: false,
      autoScrollPaused: false,
    });
  }

  /** Applies a resolved launch to the store: builds caches, restores progress, and shows
   *  the reader — or surfaces an empty/error state. Shared by every open path. */
  async function applyLaunch(launch: LaunchResult): Promise<void> {
    try {
      if (launch.kind === 'empty') {
        set({ status: 'idle', mode: 'empty' });
        return;
      }
      if (launch.kind === 'error') {
        set({ status: 'error', mode: 'empty', error: launch.message });
        return;
      }

      const provider = launch.provider;
      const manifest = launch.kind === 'standalone' ? launch.manifest : await provider.manifest();

      // Connected mode brings a server-backed prefs store + client; local mode falls back.
      const prefsServer = launch.kind === 'connected' ? (launch.prefsServer ?? null) : null;
      const serverClient = launch.kind === 'connected' ? launch.client : null;
      set({ prefsServer, serverClient, nextBook: null });

      const pages = new PageCache(provider, PAGE_CACHE_CAPACITY);
      const thumbs = new ThumbCache(provider);
      // Base every book on the global defaults (so a chosen fit/layout persists across
      // issues), but let the manifest's natural reading direction win for direction.
      const config = get().config;
      const direction = manifest.readingDir ?? config.defaults.direction;
      let settings: ReaderSettings = { ...DEFAULT_SETTINGS, ...config.defaults, direction };

      // Restore this book's saved overrides (if remembering is enabled) before laying out.
      if (config.rememberPerBook) {
        const backend = config.syncMode === 'server' && prefsServer ? prefsServer : localPrefs;
        const saved = await backend.load(manifest.bookId);
        if (saved) settings = { ...settings, ...saved };
      }
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
      void get().loadBookmarks();

      // Connected mode only knows the bookId up front; fetch the book to show a readable
      // "Series #Issue" title (standalone already uses the filename as the title).
      if (launch.kind === 'connected') {
        void (async () => {
          try {
            const d = await launch.client.bookDetail(manifest.bookId);
            if (get().manifest?.bookId !== manifest.bookId) return; // book changed meanwhile
            const label = d.number
              ? `${d.seriesName} #${d.number}`
              : d.title || d.seriesName || get().title;
            set({ title: label });
          } catch {
            // keep whatever title we have
          }
        })();
      }
    } catch (err) {
      set({
        status: 'error',
        error: err instanceof Error ? err.message : 'Failed to open this comic.',
      });
    }
  }

  /** Moves to a page, snapping to the start of its spread. Resets zoom/pan in paged modes;
   *  in continuous mode zoom is a page-size multiplier independent of scroll position, so it
   *  is preserved (scroll-driven page changes must not keep resetting it). */
  function navigateTo(page: number): void {
    const { manifest, spreads, settings } = get();
    if (!manifest) return;
    const clamped = clamp(page, 0, manifest.pageCount - 1);
    const spreadIdx = spreadIndexOf(spreads, clamped);
    const start = spreads[spreadIdx]?.[0] ?? clamped;
    const lastSpread = spreadIdx === spreads.length - 1;
    set({
      currentPage: start,
      finished: lastSpread,
      ...(settings.layout === 'continuous' ? {} : { zoom: MIN_ZOOM, panX: 0, panY: 0 }),
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
    config: loadConfig(),
    prefsServer: null,
    serverClient: null,
    nextBook: null,
    zoom: MIN_ZOOM,
    panX: 0,
    panY: 0,
    chromeVisible: true,
    settingsOpen: false,
    fullscreen: false,
    autoScroll: false,
    autoScrollPaused: false,
    bookmarks: [],
    bookmarksOpen: false,
    bmToast: null,
    provider: null,
    pages: null,
    thumbs: null,

    init: async (url?: string) => {
      set({ status: 'loading' });
      await applyLaunch(await resolveLaunch(url));
    },

    retry: () => {
      void get().init();
    },

    openUrl: (url) => {
      teardownCurrent();
      void get().init(url);
    },

    openFile: async () => {
      if (!('__TAURI_INTERNALS__' in window)) return;
      let path: string | null;
      try {
        path = await invoke<string | null>('pick_comic_file');
      } catch (err) {
        set({
          status: 'error',
          error: err instanceof Error ? err.message : 'Could not open the file picker.',
        });
        return;
      }
      if (!path) return; // user cancelled
      teardownCurrent();
      set({ status: 'loading' });
      await applyLaunch(await resolveLaunchFromPath(path));
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
      // Auto-scroll only applies to continuous; leaving it stops auto-scroll.
      set((s) => ({
        settings: { ...s.settings, layout },
        ...(layout === 'continuous' ? {} : { autoScroll: false, autoScrollPaused: false }),
      }));
      recomputeSpreads();
      navigateTo(get().currentPage);
      persistPrefs();
      saveDefaults();
    },

    toggleLayout: () => {
      const next = get().settings.layout === 'single' ? 'double' : 'single';
      get().setLayout(next);
    },

    toggleContinuous: () => {
      const next = get().settings.layout === 'continuous' ? 'single' : 'continuous';
      get().setLayout(next);
    },

    setFit: (fit) => {
      set((s) => ({ settings: { ...s.settings, fit }, zoom: MIN_ZOOM, panX: 0, panY: 0 }));
      persistPrefs();
      saveDefaults();
    },

    cycleFit: () => {
      const current = get().settings.fit;
      const idx = FIT_CYCLE.indexOf(current);
      get().setFit(FIT_CYCLE[(idx + 1) % FIT_CYCLE.length] ?? 'screen');
    },

    setDirection: (direction) => {
      set((s) => ({ settings: { ...s.settings, direction } }));
      persistPrefs();
      saveDefaults();
    },

    toggleDirection: () => {
      const next = get().settings.direction === 'ltr' ? 'rtl' : 'ltr';
      get().setDirection(next);
    },

    setBackground: (background) => {
      set((s) => ({ settings: { ...s.settings, background } }));
      persistPrefs();
      saveDefaults();
    },

    setCoverAlone: (on) => {
      set((s) => ({ settings: { ...s.settings, coverAlone: on } }));
      recomputeSpreads();
      navigateTo(get().currentPage);
      persistPrefs();
      saveDefaults();
    },

    setRememberPerBook: (on) => {
      const config = { ...get().config, rememberPerBook: on };
      set({ config });
      saveConfig(config);
      if (on) persistPrefs(); // capture the current book's settings immediately
    },

    setSyncMode: (mode) => {
      const config = { ...get().config, syncMode: mode };
      set({ config });
      saveConfig(config);
      persistPrefs(); // mirror current settings into the newly-selected store
    },

    setAutoAdvance: (mode) => {
      const config = { ...get().config, autoAdvance: mode };
      set({ config });
      saveConfig(config);
    },

    fetchNext: async () => {
      const { config, serverClient, manifest } = get();
      if (config.autoAdvance === 'off' || !serverClient || !manifest) return;
      try {
        const nb = await serverClient.nextBook(manifest.bookId, config.autoAdvance);
        if (nb) {
          const label = nb.title || (nb.number ? `Issue ${nb.number}` : 'Next issue');
          set({ nextBook: { id: nb.id, label } });
        } else {
          set({ nextBook: null });
        }
      } catch {
        set({ nextBook: null });
      }
    },

    loadNext: () => {
      const { serverClient, nextBook } = get();
      if (!serverClient || !nextBook) return;
      teardownCurrent();
      set({ status: 'loading' });
      void applyLaunch(connectedLaunch(serverClient, nextBook.id));
    },

    clearNext: () => set({ nextBook: null }),

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

    setSettingsOpen: (open) => set({ settingsOpen: open }),

    toggleFullscreen: () => {
      const next = !get().fullscreen;
      set({ fullscreen: next }); // optimistic; the window call is fire-and-forget
      void setFullscreen(next);
    },

    syncFullscreen: () => {
      void isFullscreen().then((on) => set({ fullscreen: on }));
    },

    toggleAutoScroll: () => {
      const on = !get().autoScroll;
      // Auto-scroll is a continuous-mode behavior; turning it on switches layout if needed.
      if (on && get().settings.layout !== 'continuous') {
        get().setLayout('continuous');
      }
      set({ autoScroll: on, autoScrollPaused: false });
    },

    setAutoScrollPaused: (paused) => set({ autoScrollPaused: paused }),

    setAutoScrollSpeed: (pxPerSec) => {
      const config = { ...get().config, autoScrollSpeed: pxPerSec };
      set({ config });
      saveConfig(config);
    },

    loadBookmarks: async () => {
      const { serverClient, manifest } = get();
      if (!serverClient || !manifest) {
        set({ bookmarks: [] });
        return;
      }
      try {
        const list = await serverClient.listBookmarks(manifest.bookId);
        // Only apply if we're still on the same book (open is async).
        if (get().manifest?.bookId === manifest.bookId) set({ bookmarks: list });
      } catch {
        // best-effort; leave whatever we have
      }
    },

    bookmarkCurrentPage: async () => {
      const { serverClient, manifest, currentPage, bookmarks } = get();
      if (!serverClient || !manifest) return;
      const bookId = manifest.bookId;
      // Add-only: never remove here (and never re-POST an existing page, which would
      // wipe its note). Removal is explicit, from the list.
      if (bookmarks.some((b) => b.page === currentPage)) {
        flashBookmark(`Already bookmarked · ${pageLabel(currentPage)}`);
        return;
      }
      flashBookmark(`Bookmarked ${pageLabel(currentPage)}`);
      try {
        const bm = await serverClient.addBookmark(bookId, currentPage);
        if (get().manifest?.bookId === bookId) {
          const next = [...get().bookmarks.filter((b) => b.id !== bm.id), bm].sort(
            (a, b) => a.page - b.page,
          );
          set({ bookmarks: next });
        }
      } catch {
        void get().loadBookmarks();
      }
    },

    updateBookmarkNote: async (id, note) => {
      const { serverClient, manifest } = get();
      if (!serverClient || !manifest) return;
      try {
        const updated = await serverClient.updateBookmark(manifest.bookId, id, note);
        set({ bookmarks: get().bookmarks.map((b) => (b.id === id ? updated : b)) });
      } catch {
        void get().loadBookmarks();
      }
    },

    removeBookmark: async (id) => {
      const { serverClient, manifest, bookmarks } = get();
      if (!serverClient || !manifest) return;
      set({ bookmarks: bookmarks.filter((b) => b.id !== id) }); // optimistic
      try {
        await serverClient.removeBookmark(manifest.bookId, id);
      } catch {
        void get().loadBookmarks();
      }
    },

    setBookmarksOpen: (open) => set({ bookmarksOpen: open }),

    flushProgress: () => {
      const { provider, manifest, currentPage, finished } = get();
      if (!provider || !manifest) return;
      // Merely opening a book shouldn't mark it "in progress": skip persisting while still
      // on the first page and not finished. (A single-page book opens already finished, so
      // it is still recorded as read.) See persistsPlace().
      if (!persistsPlace()) return;
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
      if (prefsTimer) {
        clearTimeout(prefsTimer);
        prefsTimer = null;
      }
      if (bmToastTimer) {
        clearTimeout(bmToastTimer);
        bmToastTimer = null;
      }
      get().flushProgress();
      get().pages?.dispose();
      get().thumbs?.dispose();
      set({ provider: null, pages: null, thumbs: null, status: 'idle' });
    },
  };
});
