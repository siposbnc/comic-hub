import { useEffect, useRef, useState } from 'react';
import { Avatar, Badge, Icon } from '@comichub/ui';
import type { UserRole } from '@comichub/api-client';
import { useClient, useConnection } from '../lib/client.js';
import { tokenStore, useAuthStore } from '../lib/auth.js';
import { Glyph } from './Glyph.js';

const ROLE_TONE: Record<UserRole, 'accent' | 'neutral' | 'warning'> = {
  owner: 'accent',
  admin: 'accent',
  member: 'neutral',
  restricted: 'warning',
};
const ROLE_LABEL: Record<UserRole, string> = {
  owner: 'Owner',
  admin: 'Admin',
  member: 'Member',
  restricted: 'Restricted',
};

/** Top-bar identity. In auth mode (signed in to a remote server) it's a chip with a popover
 *  + Sign out; in embedded / auth-disabled mode it falls back to the plain avatar. */
export function AccountChip({ fallbackName }: { fallbackName: string }) {
  const client = useClient();
  const connection = useConnection();
  const user = useAuthStore((s) => s.user);
  const setDisconnected = useAuthStore((s) => s.setDisconnected);
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  const authMode = !!tokenStore.refresh();

  useEffect(() => {
    if (!open) return;
    const onDown = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener('mousedown', onDown);
    return () => document.removeEventListener('mousedown', onDown);
  }, [open]);

  if (!authMode || !user) {
    return <Avatar name={user?.displayName || fallbackName} />;
  }

  async function signOut() {
    const refresh = tokenStore.refresh();
    tokenStore.clearTokens();
    if (refresh) {
      try {
        await client.logout(refresh);
      } catch {
        // best-effort; tokens are already cleared locally
      }
    }
    setDisconnected(true); // App swaps to the login flow
  }

  const tone = ROLE_TONE[user.role];
  const host = hostOf(connection.baseUrl);

  return (
    <div ref={ref} style={{ position: 'relative', flex: 'none' }}>
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        style={{
          display: 'inline-flex',
          alignItems: 'center',
          gap: 9,
          height: 38,
          padding: '0 8px 0 7px',
          borderRadius: 'var(--radius-pill)',
          cursor: 'pointer',
          background: open ? 'var(--surface-card)' : 'transparent',
          border: `1px solid ${open ? 'var(--border-hairline)' : 'transparent'}`,
          color: 'var(--text-primary)',
        }}
      >
        <Avatar name={user.displayName} size="sm" />
        <span
          style={{
            fontWeight: 600,
            fontSize: '0.84rem',
            maxWidth: 110,
            whiteSpace: 'nowrap',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
          }}
        >
          {user.displayName}
        </span>
        <Icon name="chevron-down" size={15} color="var(--text-tertiary)" />
      </button>

      {open && (
        <div
          style={{
            position: 'absolute',
            top: 'calc(100% + 8px)',
            right: 0,
            width: 268,
            zIndex: 80,
            background: 'var(--surface-raised)',
            border: '1px solid var(--border-strong)',
            borderRadius: 'var(--radius-lg)',
            boxShadow: 'var(--shadow-popover)',
            overflow: 'hidden',
          }}
        >
          <div style={{ display: 'flex', gap: 12, padding: '16px 16px 14px' }}>
            <Avatar name={user.displayName} size="lg" />
            <div style={{ flex: 1, minWidth: 0 }}>
              <div
                style={{
                  fontWeight: 600,
                  fontSize: '0.95rem',
                  color: 'var(--text-primary)',
                  whiteSpace: 'nowrap',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                }}
              >
                {user.displayName}
              </div>
              <div
                style={{
                  fontFamily: 'var(--font-mono)',
                  fontSize: '0.72rem',
                  color: 'var(--text-tertiary)',
                  marginTop: 3,
                }}
              >
                @{user.username}
              </div>
              <div style={{ marginTop: 9 }}>
                <Badge tone={tone} mono dot>
                  {ROLE_LABEL[user.role]}
                </Badge>
              </div>
            </div>
          </div>
          <div style={{ padding: '0 16px 12px' }}>
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 7,
                fontFamily: 'var(--font-mono)',
                fontSize: '0.66rem',
                color: 'var(--text-tertiary)',
              }}
            >
              <Glyph name="server" size={13} color="var(--text-tertiary)" /> {host}
            </div>
          </div>
          <div style={{ borderTop: '1px solid var(--border-hairline)', padding: 8 }}>
            <button
              type="button"
              onClick={() => void signOut()}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 10,
                width: '100%',
                padding: '9px 10px',
                borderRadius: 'var(--radius-md)',
                border: 'none',
                background: 'transparent',
                cursor: 'pointer',
                textAlign: 'left',
                color: 'var(--text-secondary)',
                fontFamily: 'var(--font-body)',
                fontSize: '0.86rem',
                fontWeight: 500,
              }}
            >
              <Glyph name="log-out" size={16} color="var(--text-secondary)" /> Sign out
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

function hostOf(url: string): string {
  try {
    return new URL(url).host;
  } catch {
    return url.replace(/^https?:\/\//, '');
  }
}
