import { useEffect, useLayoutEffect, useRef, useState } from 'react';
import { Icon } from '@comichub/ui';
import { useReaderStore } from './store.js';
import { usePageSnapshot } from './usePageSnapshot.js';

// Cap the page column on wide screens so webtoon pages don't blow up to full width.
const MAX_WIDTH = 1000;

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
 * Continuous (webtoon) reading: a single vertical scroll of fit-to-width pages. The page
 * nearest the top drives `currentPage` (progress + cache window via the store's goToPage);
 * external page changes (keyboard/scrubber/resume) scroll the matching page into view.
 */
export function ContinuousView() {
  const containerRef = useRef<HTMLDivElement>(null);
  const slotRefs = useRef<Array<HTMLDivElement | null>>([]);
  const [contentW, setContentW] = useState(0);

  const manifest = useReaderStore((s) => s.manifest);
  const currentPage = useReaderStore((s) => s.currentPage);
  const goToPage = useReaderStore((s) => s.goToPage);
  const toggleChrome = useReaderStore((s) => s.toggleChrome);

  const total = manifest?.pageCount ?? 0;
  const renderedW = Math.min(contentW || MAX_WIDTH, MAX_WIDTH);

  // The last page we reported to / synced from the store, to break the scroll↔state loop.
  const syncedRef = useRef(currentPage);
  const suppressRef = useRef(false);

  useLayoutEffect(() => {
    const el = containerRef.current;
    if (!el) return;
    const update = () => setContentW(el.clientWidth);
    update();
    const ro = new ResizeObserver(update);
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

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

  if (!manifest) return null;

  return (
    <div ref={containerRef} className="continuous-area" onClick={toggleChrome}>
      {Array.from({ length: total }, (_, idx) => {
        const meta = manifest.pages.find((p) => p.idx === idx);
        const aspect = meta && meta.w > 0 && meta.h > 0 ? meta.w / meta.h : 0.66;
        const height = renderedW / aspect;
        return (
          <div
            key={idx}
            ref={(el) => {
              slotRefs.current[idx] = el;
            }}
            className="continuous-page"
            style={{ width: renderedW, height }}
          >
            <ContinuousPage idx={idx} />
          </div>
        );
      })}
    </div>
  );
}
