import { useEffect } from 'react';
import { useReaderStore } from './store.js';
import { toggleFullscreen, exitFullscreen, isFullscreen } from './fullscreen.js';

/** Keys that pause auto-scroll while held (resumed on release). */
const AUTOSCROLL_PAUSE_KEYS = new Set([' ', 'ArrowDown', 'ArrowUp', 'PageDown', 'PageUp']);

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
          void toggleFullscreen();
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
          if (isFullscreen()) {
            void exitFullscreen();
          } else if (s.zoom > 1) {
            s.resetZoom();
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
