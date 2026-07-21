import { useState } from 'react';
import { Dialog, Button, Input, Select, Switch, Icon } from '@comichub/ui';
import type { BookDetail, BookKind } from '@comichub/api-client';
import { useEditBook } from '../lib/queries.js';
import { useUiStore } from '../store/ui.js';

/** The kinds a user may assign by hand, with human labels. Mirrors the server's
 *  validEditKinds; "extra art" (variant/cover) drops the file out of the numbered run. */
const KIND_OPTIONS: { value: BookKind; label: string }[] = [
  { value: 'issue', label: 'Issue' },
  { value: 'annual', label: 'Annual' },
  { value: 'special', label: 'Special' },
  { value: 'one-shot', label: 'One-shot' },
  { value: 'tpb', label: 'Collected / TPB' },
  { value: 'gn', label: 'Graphic novel' },
  { value: 'variant', label: 'Variant cover (extra)' },
  { value: 'cover', label: 'Cover / extra art' },
];

/**
 * Manual correction for a single book, for when the scanner mis-read a file that isn't a
 * duplicate: fix the issue number, title, or type (including marking it as extra cover art),
 * or hide the file from the library entirely. Corrections are locked server-side so a rescan
 * or a later provider match keeps them.
 */
export function EditBookDialog({ detail, onClose }: { detail: BookDetail; onClose: () => void }) {
  const edit = useEditBook();
  const addToast = useUiStore((s) => s.addToast);

  const [number, setNumber] = useState(detail.number ?? '');
  const [title, setTitle] = useState(detail.title ?? '');
  const [kind, setKind] = useState<BookKind>(detail.kind ?? 'issue');
  const [ignored, setIgnored] = useState(Boolean(detail.ignored));

  const origNumber = detail.number ?? '';
  const origTitle = detail.title ?? '';
  const origKind = detail.kind ?? 'issue';
  const origIgnored = Boolean(detail.ignored);

  const patch: { number?: string; title?: string; kind?: BookKind; ignored?: boolean } = {};
  if (number.trim() !== origNumber) patch.number = number.trim();
  if (title !== origTitle) patch.title = title;
  if (kind !== origKind) patch.kind = kind;
  if (ignored !== origIgnored) patch.ignored = ignored;
  const dirty = Object.keys(patch).length > 0;
  const numberInvalid = number.trim() === '' && origNumber !== '';

  const submit = () => {
    if (!dirty || numberInvalid) return;
    edit.mutate(
      { bookId: detail.id, patch },
      {
        onSuccess: () => {
          addToast({
            tone: 'success',
            title: 'Issue updated',
            message: patch.ignored
              ? 'Hidden from the library — restore it any time from Library health.'
              : 'Your correction is locked, so rescans keep it.',
          });
          onClose();
        },
        onError: (e) =>
          addToast({
            tone: 'danger',
            title: 'Could not save',
            message: e instanceof Error ? e.message : 'Unexpected error.',
          }),
      },
    );
  };

  return (
    <Dialog
      title="Edit issue"
      width={520}
      onClose={onClose}
      footer={
        <>
          <Button variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button
            variant="primary"
            icon="check"
            disabled={!dirty || numberInvalid || edit.isPending}
            onClick={submit}
          >
            {edit.isPending ? 'Saving…' : 'Save changes'}
          </Button>
        </>
      }
    >
      <p
        style={{
          margin: '0 0 18px',
          color: 'var(--text-secondary)',
          fontSize: 'var(--text-body)',
          lineHeight: 1.5,
        }}
      >
        Correct anything the scanner read wrong. Changes are locked, so a rescan or a metadata match
        won’t overwrite them.
      </p>

      <Field label="Issue number">
        <Input
          value={number}
          onChange={(e) => setNumber(e.target.value)}
          invalid={numberInvalid}
          placeholder="e.g. 1, 23.1, Futures End 1"
        />
      </Field>

      <Field label="Title">
        <Input
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="Optional — the issue's own title"
        />
      </Field>

      <Field
        label="Type"
        hint="Specials sort after the numbered run; variant/cover art is kept out of the run entirely."
      >
        <Select value={kind} onChange={(e) => setKind(e.target.value as BookKind)}>
          {KIND_OPTIONS.map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </Select>
      </Field>

      <div
        style={{
          marginTop: 8,
          padding: '12px 14px',
          display: 'flex',
          alignItems: 'flex-start',
          gap: 12,
          background: ignored
            ? 'color-mix(in oklab, var(--warning) 12%, transparent)'
            : 'var(--surface-card)',
          border: `1px solid ${
            ignored
              ? 'color-mix(in oklab, var(--warning) 35%, transparent)'
              : 'var(--border-hairline)'
          }`,
          borderRadius: 'var(--radius-md)',
        }}
      >
        <Icon
          name={ignored ? 'alert-triangle' : 'info'}
          size={17}
          color={ignored ? 'var(--warning)' : 'var(--text-tertiary)'}
        />
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ fontWeight: 600, fontSize: '0.9rem', color: 'var(--text-primary)' }}>
            Hide this file
          </div>
          <div style={{ fontSize: '0.8rem', color: 'var(--text-secondary)', marginTop: 2 }}>
            Removes it from every view without deleting it on disk. Find it again under Library
            health to restore.
          </div>
        </div>
        <Switch checked={ignored} onChange={setIgnored} />
      </div>
    </Dialog>
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
    <div style={{ marginBottom: 16 }}>
      <div
        className="ch-label"
        style={{ color: 'var(--text-tertiary)', marginBottom: 6, fontSize: 'var(--text-label)' }}
      >
        {label}
      </div>
      {children}
      {hint && (
        <div style={{ marginTop: 5, fontSize: '0.76rem', color: 'var(--text-tertiary)' }}>
          {hint}
        </div>
      )}
    </div>
  );
}
