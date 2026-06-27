import { useEffect, useRef, useState, type ReactNode } from 'react';
import { useVirtualizer } from '@tanstack/react-virtual';

interface CoverGridProps<T> {
  items: T[];
  /** Cover cell width in px (matches the CoverCard size token). */
  cardWidth: number;
  /** Estimated full cell height (cover + label block) for row virtualization. */
  rowHeight: number;
  gap?: number;
  renderItem: (item: T, index: number) => ReactNode;
  getKey: (item: T, index: number) => string;
}

/**
 * A windowed cover grid: only the visible rows render, so a library of thousands of
 * covers scrolls at 60fps. Column count tracks the container width via ResizeObserver.
 */
export function CoverGrid<T>({
  items,
  cardWidth,
  rowHeight,
  gap = 16,
  renderItem,
  getKey,
}: CoverGridProps<T>) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const [columns, setColumns] = useState(1);

  useEffect(() => {
    const el = scrollRef.current;
    if (!el) return;
    const measure = () => {
      const width = el.clientWidth;
      const cols = Math.max(1, Math.floor((width + gap) / (cardWidth + gap)));
      setColumns(cols);
    };
    measure();
    const ro = new ResizeObserver(measure);
    ro.observe(el);
    return () => ro.disconnect();
  }, [cardWidth, gap]);

  const rowCount = Math.ceil(items.length / columns);
  const virtualizer = useVirtualizer({
    count: rowCount,
    getScrollElement: () => scrollRef.current,
    estimateSize: () => rowHeight + gap,
    overscan: 3,
  });

  return (
    <div
      ref={scrollRef}
      style={{ height: '100%', overflowY: 'auto', padding: 'var(--pad-screen, 32px)' }}
    >
      <div style={{ height: virtualizer.getTotalSize(), position: 'relative', width: '100%' }}>
        {virtualizer.getVirtualItems().map((row) => {
          const start = row.index * columns;
          const cells = items.slice(start, start + columns);
          return (
            <div
              key={row.key}
              style={{
                position: 'absolute',
                top: 0,
                left: 0,
                width: '100%',
                transform: `translateY(${row.start}px)`,
                display: 'grid',
                gridTemplateColumns: `repeat(${columns}, ${cardWidth}px)`,
                gap,
                justifyContent: 'start',
              }}
            >
              {cells.map((item, i) => (
                <div key={getKey(item, start + i)}>{renderItem(item, start + i)}</div>
              ))}
            </div>
          );
        })}
      </div>
    </div>
  );
}
