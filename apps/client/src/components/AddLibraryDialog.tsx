import { useEffect, useState } from 'react';
import { useNavigate } from '@tanstack/react-router';
import { useQueryClient } from '@tanstack/react-query';
import { Button, Input, Icon } from '@comichub/ui';
import type { LibraryKind } from '@comichub/api-client';
import { useClient } from '../lib/client.js';
import { qk } from '../lib/queries.js';
import { useUiStore } from '../store/ui.js';
import { pickFolder } from '../lib/tauri.js';
import { isTauri } from '../connection.js';

/** Modal: name a library, point it at a folder, then create + kick off a full scan. */
export function AddLibraryDialog({ onClose }: { onClose: () => void }) {
  const client = useClient();
  const qc = useQueryClient();
  const navigate = useNavigate();
  const addToast = useUiStore((s) => s.addToast);

  const [name, setName] = useState('');
  const [root, setRoot] = useState('');
  const [kind, setKind] = useState<LibraryKind>('comic');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !busy) onClose();
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [busy, onClose]);

  const handleBrowse = async () => {
    const picked = await pickFolder();
    if (picked) {
      setRoot(picked);
      if (!name) setName(picked.split(/[\\/]/).filter(Boolean).pop() ?? '');
    }
  };

  const canSubmit = name.trim().length > 0 && root.trim().length > 0 && !busy;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!canSubmit) return;
    setBusy(true);
    setError(null);
    try {
      const library = await client.createLibrary({
        name: name.trim(),
        kind,
        roots: [root.trim()],
      });
      await qc.invalidateQueries({ queryKey: qk.libraries });
      try {
        await client.scanLibrary(library.id, 'full');
        addToast({
          tone: 'info',
          title: 'Scanning library',
          message: `Indexing "${library.name}". Progress shows in the top bar.`,
        });
      } catch {
        addToast({
          tone: 'warning',
          title: 'Library added',
          message: 'Created, but the scan could not start. Try scanning from the library page.',
        });
      }
      onClose();
      navigate({ to: '/library/$id', params: { id: library.id } });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not create the library.');
      setBusy(false);
    }
  };

  return (
    <div
      role="presentation"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget && !busy) onClose();
      }}
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 1100,
        display: 'grid',
        placeItems: 'center',
        background: 'color-mix(in oklab, var(--ink-900) 70%, transparent)',
        padding: 24,
      }}
    >
      <form
        role="dialog"
        aria-modal="true"
        aria-labelledby="add-library-title"
        onSubmit={handleSubmit}
        style={{
          width: 'min(480px, 100%)',
          background: 'var(--surface-raised)',
          border: '1px solid var(--border-strong)',
          borderRadius: 'var(--radius-lg)',
          boxShadow: 'var(--shadow-popover)',
          padding: 24,
          display: 'flex',
          flexDirection: 'column',
          gap: 16,
        }}
      >
        <h2
          id="add-library-title"
          style={{
            margin: 0,
            fontFamily: 'var(--font-display)',
            fontSize: 'var(--text-heading)',
            fontWeight: 700,
            color: 'var(--text-primary)',
          }}
        >
          Add a library
        </h2>

        <Field label="Name">
          <Input
            autoFocus
            placeholder="DC Comics"
            value={name}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) => setName(e.target.value)}
          />
        </Field>

        <Field
          label="Folder"
          hint={
            isTauri()
              ? 'Pick the folder that holds your .cbz / .cbr files.'
              : 'Type the absolute path to the folder of comics on the server.'
          }
        >
          <div style={{ display: 'flex', gap: 8 }}>
            <div style={{ flex: 1, minWidth: 0 }}>
              <Input
                placeholder="C:\\Comics\\DC"
                value={root}
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => setRoot(e.target.value)}
              />
            </div>
            {isTauri() && (
              <Button type="button" variant="secondary" icon="folder" onClick={handleBrowse}>
                Browse
              </Button>
            )}
          </div>
        </Field>

        <Field label="Kind">
          <div role="radiogroup" aria-label="Library kind" style={{ display: 'flex', gap: 8 }}>
            {(['comic', 'manga'] as const).map((k) => (
              <KindChip key={k} value={k} active={kind === k} onSelect={() => setKind(k)} />
            ))}
          </div>
        </Field>

        {error && (
          <div
            role="alert"
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 8,
              color: 'var(--danger)',
              fontSize: 'var(--text-small)',
            }}
          >
            <Icon name="alert-triangle" size={15} color="var(--danger)" />
            {error}
          </div>
        )}

        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 10, marginTop: 4 }}>
          <Button type="button" variant="ghost" onClick={onClose} disabled={busy}>
            Cancel
          </Button>
          <Button
            type="submit"
            variant="primary"
            disabled={!canSubmit}
            icon={busy ? 'refresh' : 'plus'}
          >
            {busy ? 'Creating…' : 'Add & scan'}
          </Button>
        </div>
      </form>
    </div>
  );
}

function Field({
  label,
  hint,
  children,
}: {
  label: string;
  hint?: string;
  children: React.ReactNode;
}) {
  return (
    <label style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
      <span
        className="ch-mono"
        style={{
          fontSize: 'var(--text-label)',
          textTransform: 'uppercase',
          letterSpacing: 'var(--tracking-label)',
          color: 'var(--text-tertiary)',
        }}
      >
        {label}
      </span>
      {children}
      {hint && (
        <span style={{ fontSize: 'var(--text-small)', color: 'var(--text-tertiary)' }}>{hint}</span>
      )}
    </label>
  );
}

function KindChip({
  value,
  active,
  onSelect,
}: {
  value: string;
  active: boolean;
  onSelect: () => void;
}) {
  return (
    <button
      type="button"
      role="radio"
      aria-checked={active}
      onClick={onSelect}
      style={{
        flex: 1,
        height: 38,
        textTransform: 'capitalize',
        background: active ? 'var(--accent-soft)' : 'var(--surface-card)',
        color: active ? 'var(--accent)' : 'var(--text-secondary)',
        border: `1px solid ${active ? 'var(--accent)' : 'var(--border-hairline)'}`,
        borderRadius: 'var(--radius-md)',
        cursor: 'pointer',
        fontFamily: 'var(--font-body)',
        fontSize: 'var(--text-body)',
        fontWeight: active ? 600 : 500,
      }}
    >
      {value}
    </button>
  );
}
