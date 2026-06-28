import { useEffect, type ReactNode } from 'react';
import { Icon } from '@comichub/ui';
import { useReaderStore } from './store.js';
import type { FitMode, LayoutMode, ReaderBackground } from './types.js';
import type { AutoAdvance, SyncMode } from './prefs.js';

const LAYOUTS: { value: LayoutMode; label: string }[] = [
  { value: 'single', label: 'Single' },
  { value: 'double', label: 'Double' },
  { value: 'continuous', label: 'Continuous' },
];
const FITS: { value: FitMode; label: string }[] = [
  { value: 'screen', label: 'Screen' },
  { value: 'width', label: 'Width' },
  { value: 'height', label: 'Height' },
  { value: 'original', label: 'Original' },
  { value: 'smart', label: 'Smart' },
];
const BACKGROUNDS: ReaderBackground[] = ['black', 'gray', 'sepia', 'white'];

/** Reader settings: the reading preferences plus where per-book overrides are stored. */
export function SettingsPanel() {
  const settings = useReaderStore((s) => s.settings);
  const config = useReaderStore((s) => s.config);
  const mode = useReaderStore((s) => s.mode);
  const setLayout = useReaderStore((s) => s.setLayout);
  const setFit = useReaderStore((s) => s.setFit);
  const setDirection = useReaderStore((s) => s.setDirection);
  const setBackground = useReaderStore((s) => s.setBackground);
  const setCoverAlone = useReaderStore((s) => s.setCoverAlone);
  const setRememberPerBook = useReaderStore((s) => s.setRememberPerBook);
  const setSyncMode = useReaderStore((s) => s.setSyncMode);
  const setAutoAdvance = useReaderStore((s) => s.setAutoAdvance);
  const close = useReaderStore((s) => s.setSettingsOpen);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') close(false);
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [close]);

  return (
    <div
      className="settings-overlay"
      role="presentation"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) close(false);
      }}
    >
      <div className="settings-panel" role="dialog" aria-label="Reader settings">
        <div className="settings-head">
          <h2>Reader settings</h2>
          <button
            type="button"
            className="settings-x"
            aria-label="Close"
            onClick={() => close(false)}
          >
            <Icon name="x" size={16} />
          </button>
        </div>

        <Row label="Layout">
          <Seg options={LAYOUTS} value={settings.layout} onPick={setLayout} />
        </Row>
        <Row label="Fit">
          <Seg options={FITS} value={settings.fit} onPick={setFit} />
        </Row>
        <Row label="Direction">
          <Seg
            options={[
              { value: 'ltr', label: 'L → R' },
              { value: 'rtl', label: 'R → L' },
            ]}
            value={settings.direction}
            onPick={setDirection}
          />
        </Row>
        <Row label="Background">
          <div className="settings-swatches">
            {BACKGROUNDS.map((bg) => (
              <button
                key={bg}
                type="button"
                className={`swatch swatch--${bg}${settings.background === bg ? ' is-active' : ''}`}
                aria-label={bg}
                aria-pressed={settings.background === bg}
                onClick={() => setBackground(bg)}
              />
            ))}
          </div>
        </Row>
        <Row label="Cover alone" hint="Show the first page as a lone cover in double mode">
          <Toggle on={settings.coverAlone} onChange={setCoverAlone} />
        </Row>

        <div className="settings-divider" />

        <Row
          label="When finished"
          hint={
            mode === 'connected'
              ? 'Auto-advance to the next issue'
              : 'Auto-advance needs a connected library'
          }
        >
          <Seg
            options={[
              { value: 'off', label: 'Off' },
              { value: 'series', label: 'Series' },
              { value: 'readingList', label: 'List' },
            ]}
            value={config.autoAdvance}
            onPick={(m: AutoAdvance) => setAutoAdvance(m)}
          />
        </Row>

        <Row label="Remember per book" hint="Restore this book's layout next time">
          <Toggle on={config.rememberPerBook} onChange={setRememberPerBook} />
        </Row>
        <Row
          label="Store settings"
          hint={mode === 'connected' ? undefined : 'Server sync needs a connected library'}
        >
          <Seg
            options={[
              { value: 'local', label: 'This device' },
              { value: 'server', label: 'Server' },
            ]}
            value={config.syncMode}
            onPick={(m: SyncMode) => setSyncMode(m)}
            disabledValue={mode === 'connected' ? undefined : 'server'}
          />
        </Row>
      </div>
    </div>
  );
}

function Row({ label, hint, children }: { label: string; hint?: string; children: ReactNode }) {
  return (
    <div className="settings-row">
      <div className="settings-row__label">
        <span>{label}</span>
        {hint && <span className="settings-row__hint">{hint}</span>}
      </div>
      <div className="settings-row__control">{children}</div>
    </div>
  );
}

function Seg<T extends string>({
  options,
  value,
  onPick,
  disabledValue,
}: {
  options: { value: T; label: string }[];
  value: T;
  onPick: (v: T) => void;
  disabledValue?: T;
}) {
  return (
    <div className="seg" role="group">
      {options.map((o) => {
        const disabled = o.value === disabledValue;
        return (
          <button
            key={o.value}
            type="button"
            className={`seg__btn${o.value === value ? ' is-active' : ''}`}
            aria-pressed={o.value === value}
            disabled={disabled}
            onClick={() => onPick(o.value)}
          >
            {o.label}
          </button>
        );
      })}
    </div>
  );
}

function Toggle({ on, onChange }: { on: boolean; onChange: (on: boolean) => void }) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={on}
      className={`toggle${on ? ' is-on' : ''}`}
      onClick={() => onChange(!on)}
    >
      <span className="toggle__knob" />
    </button>
  );
}
