import { useEffect, useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Avatar, Badge, Button, IconButton, Input, Select, Icon } from '@comichub/ui';
import { ApiError, type UserAccount, type UserRole } from '@comichub/api-client';
import { useClient } from '../lib/client.js';
import { useAuthStore } from '../lib/auth.js';
import { useUiStore } from '../store/ui.js';
import { Glyph } from './Glyph.js';

const RATINGS = ['Everyone', 'Everyone 10+', 'Teen', 'Mature 17+', 'Adults Only 18+'];
const ROLES: { value: UserRole; label: string; desc: string }[] = [
  { value: 'owner', label: 'Owner', desc: 'Full control, including server settings.' },
  { value: 'admin', label: 'Admin', desc: 'Manages libraries, metadata, and other users.' },
  { value: 'member', label: 'Member', desc: 'Reads and tracks progress. No admin access.' },
  {
    value: 'restricted',
    label: 'Restricted',
    desc: 'Only sees content at or below a rating ceiling.',
  },
];
const ROLE_TONE: Record<UserRole, 'accent' | 'neutral' | 'warning'> = {
  owner: 'accent',
  admin: 'accent',
  member: 'neutral',
  restricted: 'warning',
};
const roleLabel = (r: UserRole) => ROLES.find((x) => x.value === r)?.label ?? r;

type Modal =
  | { mode: 'create' }
  | { mode: 'edit'; user: UserAccount }
  | { mode: 'delete'; user: UserAccount }
  | null;

/** C4 — admin user management, a Settings card. Only rendered for owners/admins. */
export function UsersCard() {
  const client = useClient();
  const me = useAuthStore((s) => s.user);
  const q = useQuery({ queryKey: ['users'], queryFn: () => client.listUsers() });
  const [modal, setModal] = useState<Modal>(null);

  return (
    <section
      style={{
        background: 'var(--surface-raised)',
        border: '1px solid var(--border-hairline)',
        borderRadius: 'var(--radius-lg)',
        overflow: 'hidden',
      }}
    >
      <div
        style={{
          display: 'flex',
          alignItems: 'flex-start',
          gap: 16,
          padding: '18px 20px',
          borderBottom: '1px solid var(--border-hairline)',
        }}
      >
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 9 }}>
            <Glyph name="shield" size={17} color="var(--accent)" />
            <h2
              style={{
                margin: 0,
                fontWeight: 600,
                fontSize: '1.05rem',
                color: 'var(--text-primary)',
              }}
            >
              Users
            </h2>
          </div>
          <p
            style={{
              margin: '7px 0 0',
              fontSize: '0.84rem',
              color: 'var(--text-secondary)',
              lineHeight: 1.45,
            }}
          >
            People who can sign in to this server. Accounts are created here — there&apos;s no
            public sign-up.
          </p>
        </div>
        <Button
          variant="primary"
          size="sm"
          icon="plus"
          onClick={() => setModal({ mode: 'create' })}
        >
          Add user
        </Button>
      </div>

      <div>
        {q.data?.map((u, i) => (
          <UserRow
            key={u.id}
            u={u}
            isYou={u.id === me?.id}
            last={i === (q.data?.length ?? 0) - 1}
            onEdit={() => setModal({ mode: 'edit', user: u })}
            onDelete={() => setModal({ mode: 'delete', user: u })}
          />
        ))}
        {q.isLoading && (
          <div style={{ padding: '16px 20px', color: 'var(--text-tertiary)' }}>Loading…</div>
        )}
      </div>

      <p
        style={{
          margin: '14px 2px 0',
          fontFamily: 'var(--font-mono)',
          fontSize: '0.66rem',
          color: 'var(--text-tertiary)',
          display: 'flex',
          alignItems: 'center',
          gap: 7,
          padding: '0 20px 16px',
        }}
      >
        <Icon name="info" size={13} color="var(--text-tertiary)" /> Restricted users only see issues
        rated at or below their ceiling — over-rated content is hidden, never teased.
      </p>

      {modal?.mode === 'create' && <UserDialog onClose={() => setModal(null)} />}
      {modal?.mode === 'edit' && <UserDialog user={modal.user} onClose={() => setModal(null)} />}
      {modal?.mode === 'delete' && (
        <DeleteDialog user={modal.user} onClose={() => setModal(null)} />
      )}
    </section>
  );
}

function UserRow({
  u,
  isYou,
  last,
  onEdit,
  onDelete,
}: {
  u: UserAccount;
  isYou: boolean;
  last: boolean;
  onEdit: () => void;
  onDelete: () => void;
}) {
  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 14,
        padding: '13px 20px',
        borderBottom: last ? 'none' : '1px solid var(--border-hairline)',
      }}
    >
      <Avatar name={u.displayName} size="md" />
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <span style={{ fontWeight: 600, fontSize: '0.92rem', color: 'var(--text-primary)' }}>
            {u.displayName}
          </span>
          {isYou && (
            <span
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: '0.58rem',
                letterSpacing: '0.1em',
                textTransform: 'uppercase',
                color: 'var(--text-tertiary)',
                padding: '2px 6px',
                border: '1px solid var(--border-hairline)',
                borderRadius: 'var(--radius-sm)',
              }}
            >
              You
            </span>
          )}
        </div>
        <div
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: '0.72rem',
            color: 'var(--text-tertiary)',
            marginTop: 3,
          }}
        >
          @{u.username}
        </div>
      </div>
      {u.role === 'restricted' && u.ageRatingMax && (
        <span
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: '0.68rem',
            color: 'var(--warning)',
            padding: '3px 9px',
            borderRadius: 'var(--radius-pill)',
            background: 'color-mix(in oklab, var(--warning) 14%, transparent)',
          }}
        >
          ≤ {u.ageRatingMax}
        </span>
      )}
      <div style={{ width: 104, flex: 'none' }}>
        <Badge tone={ROLE_TONE[u.role]} mono dot>
          {roleLabel(u.role)}
        </Badge>
      </div>
      <div style={{ display: 'flex', gap: 2, flex: 'none' }}>
        <IconButton icon="edit" label="Edit user" onClick={onEdit} />
        <IconButton
          icon="trash"
          label="Delete user"
          disabled={u.role === 'owner'}
          onClick={onDelete}
        />
      </div>
    </div>
  );
}

function UserDialog({ user, onClose }: { user?: UserAccount; onClose: () => void }) {
  const client = useClient();
  const qc = useQueryClient();
  const addToast = useUiStore((s) => s.addToast);
  const isCreate = !user;

  const [username, setUsername] = useState(user?.username ?? '');
  const [displayName, setDisplayName] = useState(user?.displayName ?? '');
  const [role, setRole] = useState<UserRole>(user?.role ?? 'member');
  const [ceiling, setCeiling] = useState(user?.ageRatingMax || 'Teen');
  const [password, setPassword] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && !busy && onClose();
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [busy, onClose]);

  async function save() {
    setBusy(true);
    setError(null);
    const ageRatingMax = role === 'restricted' ? ceiling : '';
    try {
      if (isCreate) {
        await client.createUser({
          username: username.trim(),
          displayName: displayName.trim(),
          role,
          password,
          ageRatingMax,
        });
      } else {
        await client.updateUser(user.id, {
          displayName: displayName.trim(),
          role,
          ageRatingMax,
          ...(password ? { password } : {}),
        });
      }
      await qc.invalidateQueries({ queryKey: ['users'] });
      addToast({
        tone: 'success',
        title: isCreate ? 'User created' : 'User updated',
        message: `@${username.trim()}`,
      });
      onClose();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Could not save the user.');
    } finally {
      setBusy(false);
    }
  }

  const roleDesc = ROLES.find((r) => r.value === role)?.desc;

  return (
    <Modal
      title={isCreate ? 'Add user' : 'Edit user'}
      subtitle={isCreate ? 'New account on this server' : `@${user.username}`}
      onClose={onClose}
    >
      <div
        style={{ display: 'grid', gridTemplateColumns: 'minmax(0, 1fr) minmax(0, 1fr)', gap: 14 }}
      >
        <Labeled label="Username" hint={isCreate ? undefined : "Can't be changed"}>
          <Input
            style={{ width: '100%' }}
            value={username}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) => setUsername(e.target.value)}
            disabled={!isCreate}
            placeholder="lowercase"
          />
        </Labeled>
        <Labeled label="Display name">
          <Input
            style={{ width: '100%' }}
            value={displayName}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) => setDisplayName(e.target.value)}
            placeholder="Full name"
          />
        </Labeled>
      </div>

      <Labeled label="Role">
        <Select
          style={{ width: '100%' }}
          value={role}
          onChange={(e: React.ChangeEvent<HTMLSelectElement>) =>
            setRole(e.target.value as UserRole)
          }
        >
          {ROLES.map((r) => (
            <option key={r.value} value={r.value}>
              {r.label}
            </option>
          ))}
        </Select>
        {roleDesc && (
          <p style={{ margin: '8px 0 0', fontSize: '0.76rem', color: 'var(--text-tertiary)' }}>
            {roleDesc}
          </p>
        )}
      </Labeled>

      {role === 'restricted' && (
        <div
          style={{
            padding: '14px',
            borderRadius: 'var(--radius-md)',
            background: 'color-mix(in oklab, var(--warning) 9%, var(--surface-card))',
            border: '1px solid color-mix(in oklab, var(--warning) 34%, var(--border-hairline))',
          }}
        >
          <Labeled label="Content rating ceiling">
            <Select
              style={{ width: '100%' }}
              value={ceiling}
              onChange={(e: React.ChangeEvent<HTMLSelectElement>) => setCeiling(e.target.value)}
            >
              {RATINGS.map((r) => (
                <option key={r} value={r}>
                  {r}
                </option>
              ))}
            </Select>
          </Labeled>
          <p
            style={{
              margin: '9px 0 0',
              display: 'flex',
              gap: 7,
              fontSize: '0.76rem',
              color: 'var(--text-secondary)',
            }}
          >
            <Glyph name="shield" size={14} color="var(--warning)" /> This user only sees issues
            rated at or below this.
          </p>
        </div>
      )}

      <Labeled label={isCreate ? 'Password' : 'Set new password'}>
        <Input
          style={{ width: '100%' }}
          type="password"
          value={password}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => setPassword(e.target.value)}
          placeholder={isCreate ? 'At least 8 characters' : 'Leave blank to keep current'}
        />
      </Labeled>

      {error && (
        <div style={{ display: 'flex', alignItems: 'center', gap: 7, color: 'var(--danger)' }}>
          <Icon name="alert-triangle" size={14} color="var(--danger)" />
          <span style={{ fontSize: '0.8rem' }}>{error}</span>
        </div>
      )}

      <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 10, marginTop: 4 }}>
        <Button variant="ghost" onClick={onClose} disabled={busy}>
          Cancel
        </Button>
        <Button variant="primary" onClick={() => void save()} disabled={busy}>
          {isCreate ? 'Create user' : 'Save changes'}
        </Button>
      </div>
    </Modal>
  );
}

function DeleteDialog({ user, onClose }: { user: UserAccount; onClose: () => void }) {
  const client = useClient();
  const qc = useQueryClient();
  const addToast = useUiStore((s) => s.addToast);
  const [busy, setBusy] = useState(false);

  async function remove() {
    setBusy(true);
    try {
      await client.deleteUser(user.id);
      await qc.invalidateQueries({ queryKey: ['users'] });
      addToast({ tone: 'success', title: 'User deleted', message: `@${user.username}` });
      onClose();
    } catch (err) {
      addToast({
        tone: 'danger',
        title: 'Could not delete',
        message: err instanceof ApiError ? err.message : 'Unknown error.',
      });
      setBusy(false);
    }
  }

  return (
    <Modal title="Delete this user?" onClose={onClose}>
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 12,
          padding: '12px 14px',
          borderRadius: 'var(--radius-md)',
          background: 'var(--surface-card)',
          border: '1px solid var(--border-hairline)',
        }}
      >
        <Avatar name={user.displayName} size="md" />
        <div>
          <div style={{ fontWeight: 600, fontSize: '0.9rem', color: 'var(--text-primary)' }}>
            {user.displayName}
          </div>
          <div
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: '0.7rem',
              color: 'var(--text-tertiary)',
              marginTop: 2,
            }}
          >
            @{user.username} · {roleLabel(user.role)}
          </div>
        </div>
      </div>
      <p
        style={{ margin: 0, fontSize: '0.86rem', color: 'var(--text-secondary)', lineHeight: 1.5 }}
      >
        They&apos;ll be signed out and can no longer access this server. Their reading history is
        removed. Your comics stay on disk.
      </p>
      <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 10 }}>
        <Button variant="ghost" onClick={onClose} disabled={busy}>
          Cancel
        </Button>
        <Button variant="danger" icon="trash" onClick={() => void remove()} disabled={busy}>
          Delete user
        </Button>
      </div>
    </Modal>
  );
}

/** Shared modal shell (custom overlay, matching the app's dialog convention). */
function Modal({
  title,
  subtitle,
  onClose,
  children,
}: {
  title: string;
  subtitle?: string;
  onClose: () => void;
  children: React.ReactNode;
}) {
  return (
    <div
      role="presentation"
      onMouseDown={(e) => e.target === e.currentTarget && onClose()}
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 90,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        padding: 24,
        background: 'color-mix(in oklab, var(--ink-900, #000) 60%, transparent)',
        backdropFilter: 'blur(3px)',
      }}
    >
      <div
        role="dialog"
        aria-label={title}
        style={{
          width: 480,
          maxWidth: '100%',
          maxHeight: '90vh',
          overflowY: 'auto',
          background: 'var(--surface-raised)',
          border: '1px solid var(--border-strong)',
          borderRadius: 'var(--radius-lg)',
          boxShadow: 'var(--shadow-dialog)',
          padding: 20,
          display: 'flex',
          flexDirection: 'column',
          gap: 14,
        }}
      >
        <div
          style={{
            display: 'flex',
            alignItems: 'flex-start',
            justifyContent: 'space-between',
            gap: 16,
          }}
        >
          <div>
            <h2
              style={{
                margin: 0,
                fontWeight: 600,
                fontSize: '1.1rem',
                color: 'var(--text-primary)',
              }}
            >
              {title}
            </h2>
            {subtitle && (
              <div
                style={{
                  fontFamily: 'var(--font-mono)',
                  fontSize: '0.66rem',
                  color: 'var(--text-tertiary)',
                  marginTop: 4,
                }}
              >
                {subtitle}
              </div>
            )}
          </div>
          <IconButton icon="x" label="Close" onClick={onClose} />
        </div>
        {children}
      </div>
    </div>
  );
}

function Labeled({
  label,
  hint,
  children,
}: {
  label: string;
  hint?: string;
  children: React.ReactNode;
}) {
  return (
    <label style={{ display: 'block' }}>
      <span
        style={{
          display: 'block',
          fontFamily: 'var(--font-mono)',
          fontSize: '0.62rem',
          fontWeight: 600,
          letterSpacing: '0.14em',
          textTransform: 'uppercase',
          color: 'var(--text-tertiary)',
          marginBottom: 8,
        }}
      >
        {label}
      </span>
      {children}
      {hint && (
        <span
          style={{
            display: 'block',
            fontFamily: 'var(--font-mono)',
            fontSize: '0.62rem',
            color: 'var(--text-tertiary)',
            marginTop: 5,
          }}
        >
          {hint}
        </span>
      )}
    </label>
  );
}
