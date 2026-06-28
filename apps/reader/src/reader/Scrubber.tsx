import { useCallback, useRef, useState, type PointerEvent as ReactPointerEvent } from 'react';
import { useReaderStore } from './store.js';
import { useThumbUrl } from './usePageSnapshot.js';

function ThumbPreview({ idx }: { idx: number }) {
  const thumbs = useReaderStore((s) => s.thumbs);
  const url = useThumbUrl(idx);
  thumbs?.ensure(idx);
  return (
    <div className="scrubber__preview-img">
      {url ? <img src={url} alt="" draggable={false} /> : <span className="spinner" />}
      <span className="scrubber__preview-num">{idx + 1}</span>
    </div>
  );
}

export function Scrubber() {
  const trackRef = useRef<HTMLDivElement>(null);
  const manifest = useReaderStore((s) => s.manifest);
  const currentPage = useReaderStore((s) => s.currentPage);
  const direction = useReaderStore((s) => s.settings.direction);
  const goToPage = useReaderStore((s) => s.goToPage);
  const bookmarks = useReaderStore((s) => s.bookmarks);

  const [hover, setHover] = useState<{ page: number; x: number } | null>(null);
  const dragging = useRef(false);

  const count = manifest?.pageCount ?? 0;
  if (count <= 1) return null;

  const rtl = direction === 'rtl';

  const pageFromEvent = (clientX: number): number => {
    const el = trackRef.current;
    if (!el) return currentPage;
    const rect = el.getBoundingClientRect();
    let frac = (clientX - rect.left) / rect.width;
    frac = Math.min(1, Math.max(0, frac));
    if (rtl) frac = 1 - frac;
    return Math.round(frac * (count - 1));
  };

  const handleMove = useCallback(
    (e: ReactPointerEvent<HTMLDivElement>) => {
      const page = pageFromEvent(e.clientX);
      const rect = trackRef.current?.getBoundingClientRect();
      setHover({ page, x: rect ? e.clientX - rect.left : 0 });
      if (dragging.current) goToPage(page);
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [count, rtl, goToPage],
  );

  const handleDown = (e: ReactPointerEvent<HTMLDivElement>) => {
    dragging.current = true;
    (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
    goToPage(pageFromEvent(e.clientX));
  };

  const handleUp = () => {
    dragging.current = false;
  };

  const frac = count > 1 ? currentPage / (count - 1) : 0;
  // Fill is the proportion read; in RTL it's anchored to the right edge in CSS, so the
  // width is the same `frac` either way (it grows from the right as you progress).
  const fillPct = frac * 100;

  return (
    <footer className="scrubber" aria-label="Page navigation">
      {hover && (
        <div className="scrubber__preview" style={{ left: hover.x }}>
          <ThumbPreview idx={hover.page} />
        </div>
      )}
      <div className="scrubber__row">
        <span className="scrubber__count" aria-hidden="true">
          {currentPage + 1}
        </span>
        <div
          ref={trackRef}
          className={`scrubber__track${rtl ? ' is-rtl' : ''}`}
          role="slider"
          tabIndex={0}
          aria-label="Page"
          aria-valuemin={1}
          aria-valuemax={count}
          aria-valuenow={currentPage + 1}
          aria-valuetext={`Page ${currentPage + 1} of ${count}`}
          onPointerDown={handleDown}
          onPointerMove={handleMove}
          onPointerUp={handleUp}
          onPointerLeave={() => setHover(null)}
        >
          <div className="scrubber__fill" style={{ width: `${fillPct}%` }} />
          {bookmarks.map((bm) => {
            const f = count > 1 ? bm.page / (count - 1) : 0;
            const pos = (rtl ? 1 - f : f) * 100;
            return (
              <span
                key={bm.id}
                className={`scrubber__mark${bm.page === currentPage ? ' is-current' : ''}`}
                style={{ left: `${pos}%` }}
                title={`p.${bm.page + 1}${bm.note ? ` · ${bm.note}` : ''}`}
                onPointerDown={(e) => {
                  e.stopPropagation();
                  goToPage(bm.page);
                }}
              />
            );
          })}
          <div
            className="scrubber__handle"
            style={rtl ? { right: `${frac * 100}%` } : { left: `${frac * 100}%` }}
          />
        </div>
        <span className="scrubber__count scrubber__count--total" aria-hidden="true">
          {count}
        </span>
      </div>
    </footer>
  );
}
