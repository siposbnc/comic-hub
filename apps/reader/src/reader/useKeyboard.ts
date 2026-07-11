import { useEffect } from 'react';
import { useReaderStore } from './store.js';
import { closeWindow } from './fullscreen.js';

/** Keys that pause auto-scroll while held (resumed on release). */
const AUTOSCROLL_PAUSE_KEYS = new Set([' ', 'ArrowDown', 'ArrowUp', 'PageDown', 'PageUp']);

/** Reading-order intent of a navigation key, honoring RTL and Shift; null for other keys. */
function navDirection(e: KeyboardEvent, rtl: boolean): 'next' | 'prev' | null {
  switch (e.key) {
    case 'ArrowRight':
      return rtl ? 'prev' : 'next';
    case 'ArrowLeft':
      return rtl ? 'next' : 'prev';
    case ' ':
      return e.shiftKey ? 'prev' : 'next';
    case 'Enter':
    case 'ArrowDown':
    case 'PageDown':
      return 'next';
    case 'ArrowUp':
    case 'PageUp':
      return 'prev';
    default:
      return null;
  }
}

/**
 * Global keyboard navigation (docs/06-reader.md §3.4). Physical arrows map to reading order
 * by direction; Space/Shift-Space always advance/retreat in reading order.
 */
export function useKeyboard(): void {
  useEffect(() => {
    function onKey(e: KeyboardEvent): void {
      // Ignore when typing in a field.
      const target = e.target as HTMLElement | null;
      if (target && (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA')) return;

      const s = useReaderStore.getState();
      const rtl = s.settings.direction === 'rtl';

      // At the end of an issue with a next one offered, a forward gesture loads it (so the
      // reader continues without reaching for the mouse) while a back gesture retreats into
      // the current issue — which clears `finished` and cancels the auto-advance countdown.
      // So "go back a page" during the countdown does exactly that, never loads next.
      if (s.finished && s.nextBook) {
        const dir = navDirection(e, rtl);
        if (dir === 'prev') {
          e.preventDefault();
          s.prev();
          return;
        }
        if (dir === 'next') {
          e.preventDefault();
          s.loadNext();
          return;
        }
        // Non-navigation keys (Home/End/f/…) fall through to normal handling below.
      }

      // While auto-scrolling, navigation keys momentarily pause the scroll (held, not toggled)
      // instead of paging — released in the keyup handler below.
      if (s.autoScroll && AUTOSCROLL_PAUSE_KEYS.has(e.key)) {
        e.preventDefault();
        s.setAutoScrollPaused(true);
        return;
      }

      switch (e.key) {
        case 'ArrowRight':
          e.preventDefault();
          rtl ? s.prev() : s.next();
          break;
        case 'ArrowLeft':
          e.preventDefault();
          rtl ? s.next() : s.prev();
          break;
        case ' ':
          e.preventDefault();
          e.shiftKey ? s.prev() : s.next();
          break;
        case 'PageDown':
          e.preventDefault();
          s.next();
          break;
        case 'PageUp':
          e.preventDefault();
          s.prev();
          break;
        case 'Home':
          e.preventDefault();
          s.goToPage(0);
          break;
        case 'End':
          e.preventDefault();
          if (s.manifest) s.goToPage(s.manifest.pageCount - 1);
          break;
        case 'f':
        case 'F':
          e.preventDefault();
          s.toggleFullscreen();
          break;
        case 'd':
        case 'D':
          e.preventDefault();
          s.toggleLayout();
          break;
        case 'c':
        case 'C':
          e.preventDefault();
          s.toggleContinuous();
          break;
        case 'a':
        case 'A':
          e.preventDefault();
          s.toggleAutoScroll();
          break;
        case 'b':
        case 'B':
          // Bookmarks are server-backed (connected mode only); no-ops otherwise.
          // Ctrl/⌘+B opens the list; plain B adds the current page (never removes).
          e.preventDefault();
          if (e.ctrlKey || e.metaKey) {
            s.setBookmarksOpen(!s.bookmarksOpen);
          } else {
            void s.bookmarkCurrentPage();
          }
          break;
        case '+':
        case '=':
          e.preventDefault();
          s.zoomBy(1);
          break;
        case '-':
        case '_':
          e.preventDefault();
          s.zoomBy(-1);
          break;
        case '0':
          e.preventDefault();
          s.resetZoom();
          break;
        case 'Escape':
          // Escape peels off the topmost layer: an open panel, then a zoom, then the
          // reader itself — so a plain Escape while reading closes the window.
          e.preventDefault();
          if (s.settingsOpen) {
            s.setSettingsOpen(false);
          } else if (s.bookmarksOpen) {
            s.setBookmarksOpen(false);
          } else if (s.zoom > 1) {
            s.resetZoom();
          } else {
            void closeWindow();
          }
          break;
        default:
          break;
      }
    }

    function onKeyUp(e: KeyboardEvent): void {
      const s = useReaderStore.getState();
      if (s.autoScroll && s.autoScrollPaused && AUTOSCROLL_PAUSE_KEYS.has(e.key)) {
        s.setAutoScrollPaused(false);
      }
    }

    window.addEventListener('keydown', onKey);
    window.addEventListener('keyup', onKeyUp);
    return () => {
      window.removeEventListener('keydown', onKey);
      window.removeEventListener('keyup', onKeyUp);
    };
  }, []);
}
