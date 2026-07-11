import { Icon } from '@comichub/ui';
import { useReaderStore } from './store.js';
import { IconButton } from '../ui/IconButton.js';
import type { FitMode } from './types.js';

const FIT_LABEL: Record<FitMode, string> = {
  screen: 'Fit screen',
  width: 'Fit width',
  height: 'Fit height',
  original: 'Original size',
  smart: 'Smart fit',
};

export function Toolbar() {
  const title = useReaderStore((s) => s.title);
  const mode = useReaderStore((s) => s.mode);
  const layout = useReaderStore((s) => s.settings.layout);
  const fit = useReaderStore((s) => s.settings.fit);
  const direction = useReaderStore((s) => s.settings.direction);
  const zoom = useReaderStore((s) => s.zoom);
  const autoScroll = useReaderStore((s) => s.autoScroll);
  const currentPage = useReaderStore((s) => s.currentPage);
  const pageCount = useReaderStore((s) => s.manifest?.pageCount ?? 0);
  const bookmarks = useReaderStore((s) => s.bookmarks);
  const bookmarksOpen = useReaderStore((s) => s.bookmarksOpen);

  const bookmarkCurrentPage = useReaderStore((s) => s.bookmarkCurrentPage);
  const setBookmarksOpen = useReaderStore((s) => s.setBookmarksOpen);
  const toggleLayout = useReaderStore((s) => s.toggleLayout);
  const toggleContinuous = useReaderStore((s) => s.toggleContinuous);
  const toggleAutoScroll = useReaderStore((s) => s.toggleAutoScroll);
  const cycleFit = useReaderStore((s) => s.cycleFit);
  const toggleDirection = useReaderStore((s) => s.toggleDirection);
  const zoomBy = useReaderStore((s) => s.zoomBy);
  const resetZoom = useReaderStore((s) => s.resetZoom);
  const openSettings = useReaderStore((s) => s.setSettingsOpen);
  const fullscreen = useReaderStore((s) => s.fullscreen);
  const toggleFullscreen = useReaderStore((s) => s.toggleFullscreen);

  return (
    <header className="toolbar" role="toolbar" aria-label="Reader controls">
      <div className="toolbar__group toolbar__title">
        <span className="toolbar__name" title={title}>
          {title ?? 'ComicHub Reader'}
        </span>
        <span className={`toolbar__badge toolbar__badge--${mode}`}>
          {mode === 'connected' ? 'Connected' : 'Standalone'}
        </span>
      </div>

      <div className="toolbar__group toolbar__page" aria-live="polite">
        <span className="page-counter">
          <span className="page-counter__current">{pageCount ? currentPage + 1 : 0}</span>
          <span className="page-counter__sep">/</span>
          <span className="page-counter__total">{pageCount}</span>
        </span>
      </div>

      <div className="toolbar__group toolbar__actions">
        <IconButton
          icon={layout === 'double' ? 'double-page' : 'single-page'}
          label={layout === 'double' ? 'Double page' : 'Single page'}
          hint="D"
          active={layout === 'double'}
          onClick={toggleLayout}
        />
        <IconButton
          icon="list"
          label="Continuous scroll"
          hint="C"
          active={layout === 'continuous'}
          onClick={toggleContinuous}
        />
        <IconButton
          icon="chevron-down"
          label={autoScroll ? 'Stop auto-scroll' : 'Auto-scroll'}
          hint="A"
          active={autoScroll}
          onClick={toggleAutoScroll}
        />
        <IconButton icon="fit" label={FIT_LABEL[fit]} hint="cycle" onClick={cycleFit} />
        <IconButton
          icon="direction"
          label={direction === 'rtl' ? 'Right to left' : 'Left to right'}
          active={direction === 'rtl'}
          className={direction === 'rtl' ? 'is-flip' : undefined}
          onClick={toggleDirection}
        />
        <IconButton icon="settings" label="Reader settings" onClick={() => openSettings(true)} />
        {mode === 'connected' && (
          <span className="toolbar__bm">
            <IconButton
              icon="bookmark"
              label="Bookmark page"
              hint="B"
              active={bookmarks.some((b) => b.page === currentPage)}
              onClick={() => void bookmarkCurrentPage()}
            />
            <button
              type="button"
              className={`bm-count${bookmarksOpen ? ' is-open' : ''}`}
              aria-label="Bookmarks list"
              aria-expanded={bookmarksOpen}
              title="Bookmarks list"
              onClick={() => setBookmarksOpen(!bookmarksOpen)}
            >
              <span className="ch-mono">{bookmarks.length}</span>
              <Icon name="chevron-down" size={13} />
            </button>
          </span>
        )}
        <span className="toolbar__divider" aria-hidden="true" />
        <IconButton icon="zoom-out" label="Zoom out" hint="-" onClick={() => zoomBy(-1)} />
        <button
          type="button"
          className="zoom-pct"
          title="Reset zoom"
          aria-label={`Zoom ${Math.round(zoom * 100)}% — click to reset`}
          onClick={resetZoom}
        >
          {Math.round(zoom * 100)}%
        </button>
        <IconButton icon="zoom-in" label="Zoom in" hint="+" onClick={() => zoomBy(1)} />
        <IconButton
          icon={fullscreen ? 'fullscreen-exit' : 'maximize'}
          label={fullscreen ? 'Exit fullscreen' : 'Fullscreen'}
          hint="F"
          active={fullscreen}
          onClick={toggleFullscreen}
        />
      </div>
    </header>
  );
}
