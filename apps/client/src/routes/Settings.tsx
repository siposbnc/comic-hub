import { useQuery } from '@tanstack/react-query';
import { useClient, useConnection } from '../lib/client.js';
import { useUiStore, type Accent, type Theme } from '../store/ui.js';
import { Page, LoadingState } from '../components/Page.js';

const ACCENTS: { value: Accent; label: string; swatch: string }[] = [
  { value: 'cyan', label: 'Cyan', swatch: 'var(--cyan-500)' },
  { value: 'magenta', label: 'Magenta', swatch: 'var(--magenta-500)' },
  { value: 'amber', label: 'Amber', swatch: 'var(--yellow-400)' },
];

/** Preferences plus a read-only view of the server this client is bound to. */
export function Settings() {
  const client = useClient();
  const connection = useConnection();
  const theme = useUiStore((s) => s.theme);
  const setTheme = useUiStore((s) => s.setTheme);
  const accent = useUiStore((s) => s.accent);
  const setAccent = useUiStore((s) => s.setAccent);

  const info = useQuery({ queryKey: ['server', 'info'], queryFn: () => client.serverInfo() });
  const stats = useQuery({ queryKey: ['server', 'stats'], queryFn: () => client.serverStats() });

  return (
    <Page title="Settings">
      <div style={{ display: 'flex', flexDirection: 'column', gap: 32, maxWidth: 640 }}>
        <SettingsCard title="Appearance">
          <Row label="Theme">
            <div style={{ display: 'flex', gap: 8 }}>
              {(['dark', 'light'] as Theme[]).map((t) => (
                <Choice key={t} active={theme === t} onClick={() => setTheme(t)}>
                  {t}
                </Choice>
              ))}
            </div>
          </Row>
          <Row label="Accent">
            <div style={{ display: 'flex', gap: 8 }}>
              {ACCENTS.map((a) => (
                <Choice
                  key={a.value}
                  active={accent === a.value}
                  onClick={() => setAccent(a.value)}
                >
                  <span
                    aria-hidden
                    style={{
                      width: 12,
                      height: 12,
                      borderRadius: '50%',
                      background: a.swatch,
                      display: 'inline-block',
                    }}
                  />
                  {a.label}
                </Choice>
              ))}
            </div>
          </Row>
        </SettingsCard>

        <SettingsCard title="Server">
          {info.isLoading || stats.isLoading ? (
            <LoadingState />
          ) : (
            <dl
              style={{
                display: 'grid',
                gridTemplateColumns: '140px 1fr',
                gap: '10px 16px',
                margin: 0,
                fontFamily: 'var(--font-mono)',
                fontSize: 'var(--text-small)',
              }}
            >
              <Data label="address" value={connection.baseUrl} />
              <Data label="mode" value={info.data?.mode ?? '—'} />
              <Data label="version" value={info.data?.version ?? '—'} />
              <Data label="libraries" value={String(stats.data?.libraries ?? '—')} />
              <Data label="series" value={String(stats.data?.series ?? '—')} />
              <Data label="books" value={String(stats.data?.books ?? '—')} />
            </dl>
          )}
        </SettingsCard>

        <SettingsCard title="About">
          <p
            style={{
              margin: 0,
              color: 'var(--text-secondary)',
              fontSize: 'var(--text-small)',
              lineHeight: 1.6,
            }}
          >
            ComicHub keeps your comics in one calm gallery and syncs your place across every device.
            Covers are the loud thing on screen; everything else gets out of the way.
          </p>
        </SettingsCard>
      </div>
    </Page>
  );
}

function SettingsCard({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section
      style={{
        background: 'var(--surface-raised)',
        border: '1px solid var(--border-hairline)',
        borderRadius: 'var(--radius-lg)',
        padding: 22,
      }}
    >
      <h2
        className="ch-mono"
        style={{
          margin: '0 0 18px',
          fontSize: 'var(--text-label)',
          textTransform: 'uppercase',
          letterSpacing: 'var(--tracking-label)',
          color: 'var(--text-tertiary)',
        }}
      >
        {title}
      </h2>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>{children}</div>
    </section>
  );
}

function Row({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div
      style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 16 }}
    >
      <span style={{ color: 'var(--text-secondary)', fontSize: 'var(--text-small)' }}>{label}</span>
      {children}
    </div>
  );
}

function Choice({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      aria-pressed={active}
      onClick={onClick}
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 7,
        height: 32,
        padding: '0 14px',
        textTransform: 'capitalize',
        background: active ? 'var(--accent-soft)' : 'var(--surface-card)',
        color: active ? 'var(--accent)' : 'var(--text-secondary)',
        border: `1px solid ${active ? 'var(--accent)' : 'var(--border-hairline)'}`,
        borderRadius: 'var(--radius-md)',
        cursor: 'pointer',
        fontFamily: 'var(--font-body)',
        fontSize: 'var(--text-small)',
        fontWeight: active ? 600 : 500,
      }}
    >
      {children}
    </button>
  );
}

function Data({ label, value }: { label: string; value: string }) {
  return (
    <>
      <dt
        style={{
          color: 'var(--text-tertiary)',
          textTransform: 'uppercase',
          letterSpacing: '0.04em',
        }}
      >
        {label}
      </dt>
      <dd style={{ margin: 0, color: 'var(--text-primary)', wordBreak: 'break-all' }}>{value}</dd>
    </>
  );
}
