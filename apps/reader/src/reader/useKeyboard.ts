import { useEffect } from 'react';
import { useReaderStore } from './store.js';
import { toggleFullscreen, exitFullscreen, isFullscreen } from './fullscreen.js';

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

    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, []);
}
