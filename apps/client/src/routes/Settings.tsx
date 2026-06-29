import { useEffect, useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Button, Badge } from '@comichub/ui';
import type { ProviderSettingsUpdate } from '@comichub/api-client';
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

        <ProvidersCard />

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

/** Metadata-provider credentials. Secrets are write-only: the server reports only whether
 *  each is set, so saved keys/passwords are never shown back. */
function ProvidersCard() {
  const client = useClient();
  const qc = useQueryClient();
  const addToast = useUiStore((s) => s.addToast);
  const q = useQuery({
    queryKey: ['providerSettings'],
    queryFn: () => client.getProviderSettings(),
  });

  const [cvKey, setCvKey] = useState('');
  const [mUser, setMUser] = useState('');
  const [mPass, setMPass] = useState('');

  useEffect(() => {
    if (q.data) setMUser(q.data.metron.username);
  }, [q.data]);

  const save = useMutation({
    mutationFn: () => {
      const update: ProviderSettingsUpdate = { metronUsername: mUser.trim() };
      if (cvKey.trim()) update.comicVineApiKey = cvKey.trim();
      if (mPass) update.metronPassword = mPass;
      return client.updateProviderSettings(update);
    },
    onSuccess: (data) => {
      qc.setQueryData(['providerSettings'], data);
      qc.invalidateQueries({ queryKey: ['providers'] });
      setCvKey('');
      setMPass('');
      addToast({ tone: 'success', title: 'Saved', message: 'Metadata providers updated.' });
    },
    onError: (e) =>
      addToast({
        tone: 'danger',
        title: 'Could not save',
        message: e instanceof Error ? e.message : 'Unknown error.',
      }),
  });

  const cv = q.data?.comicvine.configured ?? false;
  const metron = q.data?.metron.configured ?? false;

  return (
    <SettingsCard title="Metadata providers">
      {q.isLoading ? (
        <LoadingState />
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
          <p
            style={{
              margin: 0,
              color: 'var(--text-secondary)',
              fontSize: 'var(--text-small)',
              lineHeight: 1.6,
            }}
          >
            Used to auto-match series and issues on scan, and in the manual match picker. Matching
            searches every configured provider. Credentials stay on the server.
          </p>

          <ProviderBlock label="Comic Vine" configured={cv} hint="comicvine.gamespot.com/api">
            <Field
              label="API key"
              type="password"
              value={cvKey}
              onChange={setCvKey}
              placeholder={
                cv ? '•••••••• (saved — leave blank to keep)' : 'Enter your Comic Vine API key'
              }
            />
          </ProviderBlock>

          <ProviderBlock label="Metron" configured={metron} hint="metron.cloud">
            <Field
              label="Username"
              value={mUser}
              onChange={setMUser}
              placeholder="Metron username"
            />
            <Field
              label="Password"
              type="password"
              value={mPass}
              onChange={setMPass}
              placeholder={metron ? '•••••••• (saved — leave blank to keep)' : 'Metron password'}
            />
          </ProviderBlock>

          <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
            <Button icon="check" disabled={save.isPending} onClick={() => save.mutate()}>
              {save.isPending ? 'Saving…' : 'Save providers'}
            </Button>
          </div>
        </div>
      )}
    </SettingsCard>
  );
}

function ProviderBlock({
  label,
  configured,
  hint,
  children,
}: {
  label: string;
  configured: boolean;
  hint: string;
  children: React.ReactNode;
}) {
  return (
    <div
      style={{
        border: '1px solid var(--border-hairline)',
        borderRadius: 'var(--radius-md)',
        padding: 16,
        background: 'var(--surface-card)',
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 14 }}>
        <span style={{ fontWeight: 600, color: 'var(--text-primary)' }}>{label}</span>
        {configured ? (
          <Badge tone="success" mono dot>
            connected
          </Badge>
        ) : (
          <Badge tone="neutral" mono>
            not set
          </Badge>
        )}
        <span style={{ flex: 1 }} />
        <span className="ch-mono" style={{ fontSize: '0.66rem', color: 'var(--text-tertiary)' }}>
          {hint}
        </span>
      </div>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>{children}</div>
    </div>
  );
}

function Field({
  label,
  value,
  onChange,
  placeholder,
  type = 'text',
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
  type?: 'text' | 'password';
}) {
  return (
    <label style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
      <span
        className="ch-label"
        style={{ color: 'var(--text-tertiary)', fontSize: 'var(--text-label)' }}
      >
        {label}
      </span>
      <input
        type={type}
        value={value}
        placeholder={placeholder}
        onChange={(e) => onChange(e.target.value)}
        autoComplete="off"
        spellCheck={false}
        style={{
          height: 38,
          padding: '0 12px',
          background: 'var(--surface-raised)',
          border: '1px solid var(--border-hairline)',
          borderRadius: 'var(--radius-md)',
          color: 'var(--text-primary)',
          fontFamily: 'var(--font-body)',
          fontSize: 'var(--text-body)',
          outline: 'none',
        }}
      />
    </label>
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
