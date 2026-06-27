import { useEffect, useState } from 'react';
import { useReaderStore } from './store.js';
import { IconButton } from '../ui/IconButton.js';
import { toggleFullscreen, isFullscreen } from './fullscreen.js';
import type { FitMode } from './types.js';

const FIT_LABEL: Record<FitMode, string> = {
  screen: 'Fit screen',
  width: 'Fit width',
  height: 'Fit height',
  original: 'Original size',
  smart: 'Smart fit',
};

function useFullscreenState(): boolean {
  const [full, setFull] = useState(isFullscreen);
  useEffect(() => {
    const onChange = () => setFull(isFullscreen());
    document.addEventListener('fullscreenchange', onChange);
    return () => document.removeEventListener('fullscreenchange', onChange);
  }, []);
  return full;
}

export function Toolbar() {
  const title = useReaderStore((s) => s.title);
  const mode = useReaderStore((s) => s.mode);
  const layout = useReaderStore((s) => s.settings.layout);
  const fit = useReaderStore((s) => s.settings.fit);
  const direction = useReaderStore((s) => s.settings.direction);
  const currentPage = useReaderStore((s) => s.currentPage);
  const pageCount = useReaderStore((s) => s.manifest?.pageCount ?? 0);

  const toggleLayout = useReaderStore((s) => s.toggleLayout);
  const cycleFit = useReaderStore((s) => s.cycleFit);
  const toggleDirection = useReaderStore((s) => s.toggleDirection);
  const zoomBy = useReaderStore((s) => s.zoomBy);

  const full = useFullscreenState();

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
        <IconButton icon="fit" label={FIT_LABEL[fit]} hint="cycle" onClick={cycleFit} />
        <IconButton
          icon="direction"
          label={direction === 'rtl' ? 'Right to left' : 'Left to right'}
          active={direction === 'rtl'}
          onClick={toggleDirection}
        />
        <span className="toolbar__divider" aria-hidden="true" />
        <IconButton icon="zoom-out" label="Zoom out" hint="-" onClick={() => zoomBy(-1)} />
        <IconButton icon="zoom-in" label="Zoom in" hint="+" onClick={() => zoomBy(1)} />
        <IconButton
          icon={full ? 'fullscreen-exit' : 'maximize'}
          label={full ? 'Exit fullscreen' : 'Fullscreen'}
          hint="F"
          active={full}
          onClick={() => void toggleFullscreen()}
        />
      </div>
    </header>
  );
}
