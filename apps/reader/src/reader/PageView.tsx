import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  type PointerEvent as ReactPointerEvent,
  type WheelEvent as ReactWheelEvent,
} from 'react';
import { useReaderStore } from './store.js';
import { usePageSnapshot } from './usePageSnapshot.js';
import { spreadIndexOf } from './layout.js';
import type { FitMode } from './types.js';
import { Icon } from '../ui/Icon.js';

const PAGE_GAP = 8;
const MAX_ZOOM = 5;
const MIN_ZOOM = 1;

interface Size {
  w: number;
  h: number;
}

function baseScale(fit: FitMode, spread: Size, area: Size): number {
  if (spread.w <= 0 || spread.h <= 0 || area.w <= 0 || area.h <= 0) return 1;
  const sw = area.w / spread.w;
  const sh = area.h / spread.h;
  switch (fit) {
    case 'width':
      return sw;
    case 'height':
      return sh;
    case 'original':
      return 1;
    case 'smart':
      return Math.min(sw, sh);
    case 'screen':
    default:
      return Math.min(sw, sh, 1);
  }
}

/** One page image, subscribed to the cache so it swaps in the instant it is decoded. */
function PageImage({ idx, height }: { idx: number; height: number }) {
  const snap = usePageSnapshot(idx);
  const manifest = useReaderStore((s) => s.manifest);
  const meta = manifest?.pages.find((p) => p.idx === idx);
  const aspect = meta && meta.h > 0 ? meta.w / meta.h : 0.66;
  const width = height * aspect;

  if (snap.status === 'error') {
    return (
      <div className="page-img page-img--placeholder" style={{ width, height }} role="img" aria-label={`Page ${idx + 1} could not be loaded`}>
        <Icon name="alert" size={32} />
        <span>Page {idx + 1} unavailable</span>
      </div>
    );
  }

  if (snap.status !== 'ready' || !snap.url) {
    return (
      <div className="page-img page-img--loading" style={{ width, height }} aria-busy="true">
        <span className="spinner" />
      </div>
    );
  }

  return (
    <img
      className="page-img"
      src={snap.url}
      alt={`Page ${idx + 1}`}
      draggable={false}
      decoding="async"
      style={{ height, width: 'auto' }}
    />
  );
}

export function PageView() {
  const areaRef = useRef<HTMLDivElement>(null);
  const [area, setArea] = useState<Size>({ w: 0, h: 0 });

  const manifest = useReaderStore((s) => s.manifest);
  const spreads = useReaderStore((s) => s.spreads);
  const currentPage = useReaderStore((s) => s.currentPage);
  const fit = useReaderStore((s) => s.settings.fit);
  const direction = useReaderStore((s) => s.settings.direction);
  const zoom = useReaderStore((s) => s.zoom);
  const panX = useReaderStore((s) => s.panX);
  const panY = useReaderStore((s) => s.panY);
  const pages = useReaderStore((s) => s.pages);

  const next = useReaderStore((s) => s.next);
  const prev = useReaderStore((s) => s.prev);
  const setZoom = useReaderStore((s) => s.setZoom);
  const setPan = useReaderStore((s) => s.setPan);
  const toggleZoomFit = useReaderStore((s) => s.toggleZoomFit);
  const toggleChrome = useReaderStore((s) => s.toggleChrome);

  // Track reading-area size.
  useLayoutEffect(() => {
    const el = areaRef.current;
    if (!el) return;
    const update = () => setArea({ w: el.clientWidth, h: el.clientHeight });
    update();
    const ro = new ResizeObserver(update);
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  const spreadIdx = spreadIndexOf(spreads, currentPage);
  const spreadPages = spreads[spreadIdx] ?? [currentPage];
  const ordered = direction === 'rtl' ? [...spreadPages].reverse() : spreadPages;

  // Spread natural size from the manifest (drives fit math; stable across decode).
  let spreadW = 0;
  let spreadH = 1;
  const renderHeight = 1000; // virtual layout height; scaled to fit below
  for (const idx of spreadPages) {
    const meta = manifest?.pages.find((p) => p.idx === idx);
    const aspect = meta && meta.h > 0 ? meta.w / meta.h : 0.66;
    spreadW += renderHeight * aspect;
    spreadH = renderHeight;
  }
  if (spreadPages.length > 1) spreadW += PAGE_GAP;

  const scale = baseScale(fit, { w: spreadW, h: spreadH }, area) * zoom;
  const isZoomed = zoom > MIN_ZOOM;

  // Pointer interaction: drag to pan when zoomed, otherwise click zones to turn.
  const drag = useRef<{ active: boolean; moved: boolean; x: number; y: number; px: number; py: number }>(
    { active: false, moved: false, x: 0, y: 0, px: 0, py: 0 },
  );

  const handlePointerDown = useCallback(
    (e: ReactPointerEvent<HTMLDivElement>) => {
      if (e.button !== 0) return;
      drag.current = { active: true, moved: false, x: e.clientX, y: e.clientY, px: panX, py: panY };
      (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
    },
    [panX, panY],
  );

  const handlePointerMove = useCallback(
    (e: ReactPointerEvent<HTMLDivElement>) => {
      const d = drag.current;
      if (!d.active || !isZoomed) return;
      const dx = e.clientX - d.x;
      const dy = e.clientY - d.y;
      if (Math.abs(dx) > 3 || Math.abs(dy) > 3) d.moved = true;
      setPan(d.px + dx, d.py + dy);
    },
    [isZoomed, setPan],
  );

  const turnByZone = useCallback(
    (clientX: number, rect: DOMRect) => {
      const rel = (clientX - rect.left) / rect.width;
      if (rel < 0.35) {
        direction === 'rtl' ? next() : prev();
      } else if (rel > 0.65) {
        direction === 'rtl' ? prev() : next();
      } else {
        toggleChrome();
      }
    },
    [direction, next, prev, toggleChrome],
  );

  const handlePointerUp = useCallback(
    (e: ReactPointerEvent<HTMLDivElement>) => {
      const d = drag.current;
      d.active = false;
      if (isZoomed) return; // dragging panned; no turn
      if (d.moved) return;
      turnByZone(e.clientX, e.currentTarget.getBoundingClientRect());
    },
    [isZoomed, turnByZone],
  );

  const handleWheel = useCallback(
    (e: ReactWheelEvent<HTMLDivElement>) => {
      if (!e.ctrlKey && !e.metaKey && !isZoomed) return; // let plain scroll pass when fit
      e.preventDefault();
      const rect = e.currentTarget.getBoundingClientRect();
      const cx = e.clientX - rect.left - rect.width / 2;
      const cy = e.clientY - rect.top - rect.height / 2;
      const factor = e.deltaY < 0 ? 1.15 : 1 / 1.15;
      const nextZoom = Math.min(MAX_ZOOM, Math.max(MIN_ZOOM, zoom * factor));
      const ratio = nextZoom / zoom;
      // Keep the point under the cursor anchored as we scale.
      setPan(cx - (cx - panX) * ratio, cy - (cy - panY) * ratio);
      setZoom(nextZoom);
    },
    [isZoomed, zoom, panX, panY, setPan, setZoom],
  );

  // Ensure non-passive wheel so preventDefault works.
  useEffect(() => {
    const el = areaRef.current;
    if (!el) return;
    const handler = (e: WheelEvent) => {
      if (e.ctrlKey || e.metaKey || zoom > MIN_ZOOM) e.preventDefault();
    };
    el.addEventListener('wheel', handler, { passive: false });
    return () => el.removeEventListener('wheel', handler);
  }, [zoom]);

  // Warm thumbnails near the current page for the scrubber preview.
  useEffect(() => {
    pages?.get(currentPage);
  }, [pages, currentPage]);

  return (
    <div
      ref={areaRef}
      className={`page-area${isZoomed ? ' is-zoomed' : ''}`}
      onPointerDown={handlePointerDown}
      onPointerMove={handlePointerMove}
      onPointerUp={handlePointerUp}
      onWheel={handleWheel}
      onDoubleClick={toggleZoomFit}
    >
      <div
        className="page-stage"
        style={{
          gap: PAGE_GAP,
          transform: `translate3d(${panX}px, ${panY}px, 0) scale(${scale})`,
        }}
      >
        {ordered.map((idx) => (
          <PageImage key={idx} idx={idx} height={renderHeight} />
        ))}
      </div>
    </div>
  );
}
