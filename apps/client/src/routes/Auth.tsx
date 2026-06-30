import { useState } from 'react';
import { Icon } from '@comichub/ui';
import { ComicHubClient, type Connection, type User } from '@comichub/api-client';
import { Glyph } from '../components/Glyph.js';
import { tokenStore, useAuthStore, wireRefresh } from '../lib/auth.js';

/** Result handed back to the App when sign-in completes. */
export interface AuthResult {
  client: ComicHubClient;
  connection: Connection;
  user: User;
}

function hostOf(url: string): string {
  try {
    return new URL(url).host;
  } catch {
    return url.replace(/^https?:\/\//, '');
  }
}

/**
 * C1 + C2 — the connect → login flow shown when a server requires auth (or the user wants to
 * point at a different server). On success it persists the server + tokens, wires the
 * refresh hook, and resolves with a ready client.
 */
export function AuthFlow({
  initialBaseUrl,
  startAtLogin,
  onAuthenticated,
}: {
  initialBaseUrl: string;
  startAtLogin: boolean;
  onAuthenticated: (r: AuthResult) => void;
}) {
  const [mode, setMode] = useState<'connect' | 'login'>(startAtLogin ? 'login' : 'connect');
  const [baseUrl, setBaseUrl] = useState(initialBaseUrl);

  if (mode === 'login') {
    return (
      <LoginCard
        baseUrl={baseUrl}
        onBack={() => setMode('connect')}
        onAuthenticated={onAuthenticated}
      />
    );
  }
  return (
    <ConnectCard
      initial={baseUrl}
      onConnected={(url) => {
        setBaseUrl(url);
        setMode('login');
      }}
    />
  );
}

/* ── C1 — Connect to a server ─────────────────────────────────────────────── */

function ConnectCard({
  initial,
  onConnected,
}: {
  initial: string;
  onConnected: (baseUrl: string) => void;
}) {
  const [url, setUrl] = useState(initial);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState(false);

  async function connect() {
    const trimmed = url.trim().replace(/\/$/, '');
    if (!trimmed) return;
    setBusy(true);
    setError(false);
    try {
      // Reachability check against the entered server.
      await new ComicHubClient({ baseUrl: trimmed, token: '' }).health();
      tokenStore.setServerUrl(trimmed);
      onConnected(trimmed);
    } catch {
      setError(true);
    } finally {
      setBusy(false);
    }
  }

  return (
    <AuthScaffold subtitle="Connect to a server">
      <label style={labelStyle}>Server URL</label>
      <div style={inputShell(error ? 'var(--danger)' : 'var(--border-strong)', busy)}>
        <Glyph name="server" size={17} color="var(--text-tertiary)" />
        <input
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          placeholder="http://host:port"
          disabled={busy}
          onKeyDown={(e) => e.key === 'Enter' && void connect()}
          style={{ ...inputEl, fontFamily: 'var(--font-mono)' }}
        />
      </div>
      {error && (
        <InlineError text="Couldn't reach that server. Check the address and that it's running." />
      )}
      <PrimaryButton busy={busy} busyLabel="Connecting…" onClick={() => void connect()}>
        Connect <Icon name="chevron-right" size={17} color="var(--text-on-accent)" />
      </PrimaryButton>
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          marginTop: 16,
          color: 'var(--text-tertiary)',
        }}
      >
        <Icon name="info" size={14} color="var(--text-tertiary)" />
        <span style={{ fontSize: '0.78rem', lineHeight: 1.4 }}>
          Running ComicHub on this device instead? It starts automatically.
        </span>
      </div>
    </AuthScaffold>
  );
}

/* ── C2 — Login ───────────────────────────────────────────────────────────── */

function LoginCard({
  baseUrl,
  onBack,
  onAuthenticated,
}: {
  baseUrl: string;
  onBack: () => void;
  onAuthenticated: (r: AuthResult) => void;
}) {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState(false);

  async function submit() {
    if (!username.trim() || !password) return;
    setBusy(true);
    setError(false);
    const client = new ComicHubClient({ baseUrl, token: '' });
    try {
      const t = await client.login(username.trim(), password);
      tokenStore.setServerUrl(baseUrl);
      tokenStore.setTokens(t.access, t.refresh);
      client.setAccessToken(t.access);
      wireRefresh(client);
      useAuthStore.getState().setUser(t.user);
      onAuthenticated({ client, connection: { baseUrl, token: t.access }, user: t.user });
    } catch {
      // Any failure (incl. the single 401 for either wrong field) surfaces one message —
      // never reveal which field was wrong.
      setError(true);
    } finally {
      setBusy(false);
    }
  }

  return (
    <AuthScaffold
      subtitle={
        <>
          Sign in to <span style={{ color: 'var(--accent)' }}>{hostOf(baseUrl)}</span>
        </>
      }
    >
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        <AuthField
          label="Username"
          icon="user"
          mono
          value={username}
          onChange={setUsername}
          disabled={busy}
          onEnter={() => void submit()}
        />
        <AuthField
          label="Password"
          type="password"
          value={password}
          onChange={setPassword}
          disabled={busy}
          invalid={error}
          onEnter={() => void submit()}
        />
      </div>
      {error && <InlineError text="Incorrect username or password." />}
      <PrimaryButton busy={busy} busyLabel="Signing in…" onClick={() => void submit()}>
        Sign in
      </PrimaryButton>
      <div style={{ display: 'flex', justifyContent: 'center', marginTop: 16 }}>
        <button type="button" onClick={onBack} style={linkButton}>
          <Icon name="chevron-left" size={14} color="var(--text-tertiary)" /> Use a different server
        </button>
      </div>
    </AuthScaffold>
  );
}

/* ── shared pieces (match Phase3Preview AuthScaffold / Field / Spinner) ──────── */

function AuthScaffold({
  subtitle,
  children,
}: {
  subtitle: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <div
      style={{
        position: 'absolute',
        inset: 0,
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        background: 'var(--bg-app)',
        color: 'var(--text-primary)',
        overflow: 'hidden',
      }}
    >
      <div
        className="ch-halftone-duo"
        style={{
          position: 'absolute',
          top: -120,
          right: -120,
          width: 420,
          height: 420,
          borderRadius: '50%',
          opacity: 0.16,
        }}
      />
      <div style={{ position: 'relative', width: 408, maxWidth: '90%' }}>
        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            marginBottom: 24,
          }}
        >
          <div
            style={{
              fontFamily: 'var(--font-display)',
              fontWeight: 800,
              fontSize: '1.9rem',
              letterSpacing: '-0.02em',
            }}
          >
            Comic<span style={{ color: 'var(--accent)' }}>Hub</span>
          </div>
          <div style={{ fontSize: '0.95rem', color: 'var(--text-secondary)', marginTop: 6 }}>
            {subtitle}
          </div>
        </div>
        <div
          style={{
            background: 'var(--surface-raised)',
            border: '1px solid var(--border-hairline)',
            borderRadius: 'var(--radius-lg)',
            padding: '24px 24px 26px',
            boxShadow: 'var(--shadow-dialog)',
          }}
        >
          {children}
        </div>
      </div>
    </div>
  );
}

function AuthField({
  label,
  type = 'text',
  icon,
  mono,
  value,
  onChange,
  disabled,
  invalid,
  onEnter,
}: {
  label: string;
  type?: 'text' | 'password';
  icon?: 'user';
  mono?: boolean;
  value: string;
  onChange: (v: string) => void;
  disabled?: boolean;
  invalid?: boolean;
  onEnter?: () => void;
}) {
  return (
    <div style={{ opacity: disabled ? 0.6 : 1 }}>
      <label style={labelStyle}>{label}</label>
      <div style={inputShell(invalid ? 'var(--danger)' : 'var(--border-strong)', false)}>
        {icon && <Icon name={icon} size={16} color="var(--text-tertiary)" />}
        <input
          type={type}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          disabled={disabled}
          autoComplete={type === 'password' ? 'current-password' : 'username'}
          onKeyDown={(e) => e.key === 'Enter' && onEnter?.()}
          style={{ ...inputEl, fontFamily: mono ? 'var(--font-mono)' : 'var(--font-body)' }}
        />
      </div>
    </div>
  );
}

function PrimaryButton({
  busy,
  busyLabel,
  onClick,
  children,
}: {
  busy: boolean;
  busyLabel: string;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      disabled={busy}
      onClick={onClick}
      style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        gap: 9,
        width: '100%',
        height: 44,
        marginTop: 16,
        borderRadius: 'var(--radius-md)',
        border: 'none',
        cursor: busy ? 'default' : 'pointer',
        background: 'var(--accent)',
        color: 'var(--text-on-accent)',
        fontFamily: 'var(--font-body)',
        fontWeight: 600,
        fontSize: '0.92rem',
        opacity: busy ? 0.85 : 1,
      }}
    >
      {busy ? (
        <>
          <Spinner /> {busyLabel}
        </>
      ) : (
        children
      )}
    </button>
  );
}

/** A small spinner using the DS `ch-spin` keyframe (base.css). */
function Spinner() {
  return (
    <span
      style={{
        display: 'inline-block',
        width: 16,
        height: 16,
        borderRadius: '50%',
        border: '2px solid color-mix(in oklab, currentColor 28%, transparent)',
        borderTopColor: 'currentColor',
        animation: 'ch-spin 0.7s linear infinite',
      }}
    />
  );
}

function InlineError({ text }: { text: string }) {
  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 7,
        marginTop: 10,
        color: 'var(--danger)',
      }}
    >
      <Icon name="alert-triangle" size={14} color="var(--danger)" />
      <span style={{ fontSize: '0.8rem' }}>{text}</span>
    </div>
  );
}

const labelStyle: React.CSSProperties = {
  display: 'block',
  fontFamily: 'var(--font-mono)',
  fontSize: '0.64rem',
  fontWeight: 600,
  letterSpacing: '0.14em',
  textTransform: 'uppercase',
  color: 'var(--text-tertiary)',
  marginBottom: 8,
};

function inputShell(border: string, busy: boolean): React.CSSProperties {
  return {
    display: 'flex',
    alignItems: 'center',
    gap: 9,
    height: 44,
    padding: '0 13px',
    background: 'var(--surface-card)',
    borderRadius: 'var(--radius-md)',
    border: `1px solid ${border}`,
    opacity: busy ? 0.6 : 1,
  };
}

const inputEl: React.CSSProperties = {
  flex: 1,
  minWidth: 0,
  height: '100%',
  background: 'transparent',
  border: 'none',
  outline: 'none',
  color: 'var(--text-primary)',
  fontSize: '0.92rem',
};

const linkButton: React.CSSProperties = {
  display: 'inline-flex',
  alignItems: 'center',
  gap: 6,
  background: 'none',
  border: 'none',
  cursor: 'pointer',
  color: 'var(--text-tertiary)',
  fontFamily: 'var(--font-body)',
  fontSize: '0.8rem',
};
