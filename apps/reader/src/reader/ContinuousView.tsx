import { useEffect, useLayoutEffect, useRef, useState } from 'react';
import { Icon } from '@comichub/ui';
import { useReaderStore } from './store.js';
import { usePageSnapshot } from './usePageSnapshot.js';
import type { FitMode } from './types.js';

// Cap the page column on wide screens so webtoon pages don't blow up to full width.
const MAX_WIDTH = 1000;

/** Rendered column width (CSS px) for a page in continuous mode, honoring the fit mode and
 *  the zoom multiplier. Fit decides the base size from the viewport + page aspect; zoom
 *  scales it (so zooming in a webtoon enlarges pages and scrolls through them). */
function columnWidth(
  fit: FitMode,
  aspect: number,
  viewW: number,
  viewH: number,
  nativeW: number,
  zoom: number,
): number {
  const byWidth = viewW;
  const byHeight = viewH * aspect; // width such that the page height equals the viewport
  let base: number;
  switch (fit) {
    case 'width':
      base = byWidth;
      break;
    case 'height':
      base = byHeight;
      break;
    case 'original':
      base = nativeW > 0 ? nativeW : byWidth;
      break;
    case 'smart':
      // Contain the whole page, upscaling small pages to fill the column (matches paged).
      base = Math.min(byWidth, byHeight);
      break;
    case 'screen':
    default:
      // Contain, but never enlarge a page beyond its native width (the `,1` cap of paged
      // mode): a small scan stays at its real size instead of being blown up.
      base = Math.min(byWidth, byHeight);
      if (nativeW > 0) base = Math.min(base, nativeW);
      break;
  }
  // Cap width-driven fits on ultra-wide screens; native/height keep their intrinsic size.
  if (fit !== 'original' && fit !== 'height') base = Math.min(base, MAX_WIDTH);
  return Math.max(1, base * zoom);
}

/** One page in the continuous column. The slot's height is fixed from the manifest aspect
 *  so the scrollbar is stable; the decoded image swaps in once the cache window covers it. */
function ContinuousPage({ idx }: { idx: number }) {
  const snap = usePageSnapshot(idx);
  if (snap.status === 'ready' && snap.url) {
    return (
      <img
        className="continuous-img"
        src={snap.url}
        alt={`Page ${idx + 1}`}
        draggable={false}
        decoding="async"
      />
    );
  }
  if (snap.status === 'error') {
    return (
      <div className="continuous-ph" role="img" aria-label={`Page ${idx + 1} unavailable`}>
        <Icon name="alert-triangle" size={28} />
      </div>
    );
  }
  return (
    <div className="continuous-ph" aria-busy="true">
      <span className="spinner" />
    </div>
  );
}

/**
 * Continuous (webtoon) reading: a vertical scroll of pages sized by the fit mode and zoom.
 * The page nearest the top drives `currentPage` (progress + cache window); external page
 * changes (keyboard/scrubber/resume) and the initial open scroll the matching page into
 * view. Auto-scroll advances the column at a configurable speed.
 */
export function ContinuousView() {
  const containerRef = useRef<HTMLDivElement>(null);
  const slotRefs = useRef<Array<HTMLDivElement | null>>([]);
  const [view, setView] = useState({ w: 0, h: 0 });

  const manifest = useReaderStore((s) => s.manifest);
  const currentPage = useReaderStore((s) => s.currentPage);
  const fit = useReaderStore((s) => s.settings.fit);
  const zoom = useReaderStore((s) => s.zoom);
  const autoScroll = useReaderStore((s) => s.autoScroll);
  const goToPage = useReaderStore((s) => s.goToPage);
  const toggleChrome = useReaderStore((s) => s.toggleChrome);

  const total = manifest?.pageCount ?? 0;

  // The last page we reported to / synced from the store, to break the scroll↔state loop.
  const syncedRef = useRef(currentPage);
  const suppressRef = useRef(false);
  const didInitRef = useRef(false);

  useLayoutEffect(() => {
    const el = containerRef.current;
    if (!el) return;
    const update = () => setView({ w: el.clientWidth, h: el.clientHeight });
    update();
    const ro = new ResizeObserver(update);
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  // On first layout (and when toggling into continuous), scroll to the current page rather
  // than snapping to the top — otherwise the view jumps to page 1 while progress stays put.
  useEffect(() => {
    if (didInitRef.current || !total || view.w === 0) return;
    if (currentPage > 0) {
      const slot = slotRefs.current[currentPage];
      if (slot) {
        suppressRef.current = true;
        slot.scrollIntoView({ block: 'start' });
        syncedRef.current = currentPage;
        window.setTimeout(() => {
          suppressRef.current = false;
        }, 250);
      }
    }
    didInitRef.current = true;
  }, [total, view.w, currentPage]);

  // External page change (keyboard, scrubber, resume): scroll that page to the top.
  useEffect(() => {
    if (currentPage === syncedRef.current) return; // came from our own scroll handler
    const slot = slotRefs.current[currentPage];
    if (!slot) return;
    suppressRef.current = true;
    slot.scrollIntoView({ block: 'start' });
    syncedRef.current = currentPage;
    const t = window.setTimeout(() => {
      suppressRef.current = false;
    }, 250);
    return () => window.clearTimeout(t);
  }, [currentPage]);

  // Scroll: the page occupying the top quarter becomes current (drives progress + window).
  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;
    let raf = 0;
    const onScroll = () => {
      if (raf) return;
      raf = requestAnimationFrame(() => {
        raf = 0;
        if (suppressRef.current) return;
        const mark = el.scrollTop + el.clientHeight * 0.25;
        let page = 0;
        for (let i = 0; i < slotRefs.current.length; i++) {
          const slot = slotRefs.current[i];
          if (slot && slot.offsetTop <= mark) page = i;
          else if (slot) break;
        }
        if (page !== syncedRef.current) {
          syncedRef.current = page;
          goToPage(page);
        }
      });
    };
    el.addEventListener('scroll', onScroll, { passive: true });
    return () => {
      el.removeEventListener('scroll', onScroll);
      if (raf) cancelAnimationFrame(raf);
    };
  }, [goToPage]);

  // Auto-scroll: advance the column at the configured speed, pausing when a key is held.
  // Speed/pause are read live from the store so changes apply mid-scroll without re-binding.
  useEffect(() => {
    if (!autoScroll) return;
    const el = containerRef.current;
    if (!el) return;
    let raf = 0;
    let last = performance.now();
    const step = (now: number) => {
      const dt = (now - last) / 1000;
      last = now;
      const st = useReaderStore.getState();
      if (!st.autoScroll) return;
      if (!st.autoScrollPaused) {
        el.scrollTop += st.config.autoScrollSpeed * dt;
        if (el.scrollTop + el.clientHeight >= el.scrollHeight - 1) {
          useReaderStore.setState({ autoScroll: false }); // reached the end
          return;
        }
      }
      raf = requestAnimationFrame(step);
    };
    raf = requestAnimationFrame(step);
    return () => cancelAnimationFrame(raf);
  }, [autoScroll]);

  if (!manifest) return null;

  return (
    <div ref={containerRef} className="continuous-area" onClick={toggleChrome}>
      {Array.from({ length: total }, (_, idx) => {
        const meta = manifest.pages.find((p) => p.idx === idx);
        const aspect = meta && meta.w > 0 && meta.h > 0 ? meta.w / meta.h : 0.66;
        const nativeW = meta?.w ?? 0;
        const width = columnWidth(fit, aspect, view.w || MAX_WIDTH, view.h, nativeW, zoom);
        const height = width / aspect;
        return (
          <div
            key={idx}
            ref={(el) => {
              slotRefs.current[idx] = el;
            }}
            className="continuous-page"
            style={{ width, height }}
          >
            <ContinuousPage idx={idx} />
          </div>
        );
      })}
    </div>
  );
}
