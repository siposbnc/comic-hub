import { useCallback, useEffect, useRef, useState } from 'react';
import { useReaderStore } from './store.js';
import { useKeyboard } from './useKeyboard.js';
import { Toolbar } from './Toolbar.js';
import { Scrubber } from './Scrubber.js';
import { PageView } from './PageView.js';
import { ContinuousView } from './ContinuousView.js';
import { SettingsPanel } from './SettingsPanel.js';
import { BookmarksPanel } from './BookmarksPanel.js';
import { Button } from '@comichub/ui';
import { Icon } from '@comichub/ui';

const IDLE_HIDE_MS = 2800;
const RESUME_TOAST_MS = 6000;

/** The reading surface: pages, chrome (auto-hiding), resume + completion affordances. */
export function Reader() {
  useKeyboard();

  const background = useReaderStore((s) => s.settings.background);
  const continuous = useReaderStore((s) => s.settings.layout === 'continuous');
  const chromeVisible = useReaderStore((s) => s.chromeVisible);
  const showChrome = useReaderStore((s) => s.showChrome);
  const hideChrome = useReaderStore((s) => s.hideChrome);
  const finished = useReaderStore((s) => s.finished);
  const settingsOpen = useReaderStore((s) => s.settingsOpen);
  const bookmarksOpen = useReaderStore((s) => s.bookmarksOpen);
  const bmToast = useReaderStore((s) => s.bmToast);
  const resumePage = useReaderStore((s) => s.resumePage);
  const nextBook = useReaderStore((s) => s.nextBook);
  const dismissFinished = useReaderStore((s) => s.dismissFinished);
  const startOver = useReaderStore((s) => s.startOver);
  const dismissResume = useReaderStore((s) => s.dismissResume);
  const flushProgress = useReaderStore((s) => s.flushProgress);
  const fetchNext = useReaderStore((s) => s.fetchNext);
  const clearNext = useReaderStore((s) => s.clearNext);

  const idleTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const bumpActivity = useCallback(() => {
    showChrome();
    if (idleTimer.current) clearTimeout(idleTimer.current);
    idleTimer.current = setTimeout(() => hideChrome(), IDLE_HIDE_MS);
  }, [showChrome, hideChrome]);

  useEffect(() => {
    bumpActivity();
    return () => {
      if (idleTimer.current) clearTimeout(idleTimer.current);
    };
  }, [bumpActivity]);

  // Persist progress on blur / tab hide so a place is never lost.
  useEffect(() => {
    const onHide = () => flushProgress();
    window.addEventListener('blur', onHide);
    document.addEventListener('visibilitychange', onHide);
    return () => {
      window.removeEventListener('blur', onHide);
      document.removeEventListener('visibilitychange', onHide);
    };
  }, [flushProgress]);

  // Auto-dismiss the resume toast.
  useEffect(() => {
    if (resumePage == null) return;
    const t = setTimeout(() => dismissResume(), RESUME_TOAST_MS);
    return () => clearTimeout(t);
  }, [resumePage, dismissResume]);

  // On reaching the end, resolve the next issue (when auto-advance is enabled).
  useEffect(() => {
    if (finished) void fetchNext();
    else clearNext();
  }, [finished, fetchNext, clearNext]);

  return (
    <div
      className={`reader-shell bg-${background}${chromeVisible ? ' chrome-on' : ' chrome-off'}`}
      onPointerMove={bumpActivity}
    >
      <div className="chrome chrome--top" onPointerEnter={showChrome}>
        <Toolbar />
      </div>

      {continuous ? <ContinuousView /> : <PageView />}

      <div className="chrome chrome--bottom" onPointerEnter={showChrome}>
        <Scrubber />
      </div>

      <ZoomIndicator />

      {settingsOpen && <SettingsPanel />}
      {bookmarksOpen && <BookmarksPanel />}

      {bmToast && (
        <div className="toast bm-confirm" role="status">
          <Icon name="bookmark" size={14} />
          <span>{bmToast}</span>
        </div>
      )}

      {resumePage != null && (
        <div className="toast" role="status">
          <span>
            Resumed from page <strong>{resumePage + 1}</strong>
          </span>
          <button type="button" className="toast__link" onClick={startOver}>
            Start over
          </button>
          <button
            type="button"
            className="toast__close"
            aria-label="Dismiss"
            onClick={dismissResume}
          >
            <Icon name="x" size={16} />
          </button>
        </div>
      )}

      {finished && (
        <div className="endcard" role="dialog" aria-label="End of book">
          <div className="endcard__inner">
            <Icon name="check" size={40} />
            <h2>You&apos;ve reached the end</h2>
            {nextBook ? (
              <AutoNext label={nextBook.label} />
            ) : (
              <FinishedActions onStartOver={startOver} onKeep={dismissFinished} />
            )}
          </div>
        </div>
      )}
    </div>
  );
}

const AUTO_NEXT_SECONDS = 5;

/** End-of-book offer to advance to the next issue, with a short auto-load countdown. */
function AutoNext({ label }: { label: string }) {
  const loadNext = useReaderStore((s) => s.loadNext);
  const dismiss = useReaderStore((s) => s.dismissFinished);
  const [secs, setSecs] = useState(AUTO_NEXT_SECONDS);

  useEffect(() => {
    if (secs <= 0) {
      loadNext();
      return;
    }
    const t = setTimeout(() => setSecs((s) => s - 1), 1000);
    return () => clearTimeout(t);
  }, [secs, loadNext]);

  return (
    <>
      <p>
        Up next: <strong>{label}</strong>
      </p>
      <div className="endcard__actions">
        <Button onClick={loadNext}>Read next now</Button>
        <Button variant="ghost" onClick={dismiss}>
          Stay ({secs})
        </Button>
      </div>
    </>
  );
}

const ZOOM_FADE_MS = 1400;

/** Transient zoom readout: shows the current zoom % with a Reset button whenever the zoom
 *  level changes, then fades away. (The toolbar carries the always-visible readout.) */
function ZoomIndicator() {
  const zoom = useReaderStore((s) => s.zoom);
  const resetZoom = useReaderStore((s) => s.resetZoom);
  const [visible, setVisible] = useState(false);
  const mounted = useRef(false);

  useEffect(() => {
    // Don't flash on first render — only on actual zoom changes.
    if (!mounted.current) {
      mounted.current = true;
      return;
    }
    setVisible(true);
    const t = setTimeout(() => setVisible(false), ZOOM_FADE_MS);
    return () => clearTimeout(t);
  }, [zoom]);

  return (
    <div className={`zoom-indicator${visible ? ' is-visible' : ''}`} aria-hidden={!visible}>
      <span className="zoom-indicator__pct">{Math.round(zoom * 100)}%</span>
      <button type="button" className="zoom-indicator__reset" onClick={resetZoom}>
        Reset
      </button>
    </div>
  );
}

function FinishedActions({ onStartOver, onKeep }: { onStartOver: () => void; onKeep: () => void }) {
  return (
    <>
      <p>Marked as read.</p>
      <div className="endcard__actions">
        <Button onClick={onStartOver}>Start over</Button>
        <Button variant="ghost" onClick={onKeep}>
          Keep reading
        </Button>
      </div>
    </>
  );
}
