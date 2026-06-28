import { useCallback, useEffect, useRef } from 'react';
import { useReaderStore } from './store.js';
import { useKeyboard } from './useKeyboard.js';
import { Toolbar } from './Toolbar.js';
import { Scrubber } from './Scrubber.js';
import { PageView } from './PageView.js';
import { ContinuousView } from './ContinuousView.js';
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
  const resumePage = useReaderStore((s) => s.resumePage);
  const dismissFinished = useReaderStore((s) => s.dismissFinished);
  const startOver = useReaderStore((s) => s.startOver);
  const dismissResume = useReaderStore((s) => s.dismissResume);
  const flushProgress = useReaderStore((s) => s.flushProgress);

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
            <p>Marked as read.</p>
            <div className="endcard__actions">
              <Button onClick={startOver}>Start over</Button>
              <Button variant="ghost" onClick={dismissFinished}>
                Keep reading
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
