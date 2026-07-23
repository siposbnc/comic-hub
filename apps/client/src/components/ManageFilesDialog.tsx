import { useState } from 'react';
import { Dialog, Button, Badge } from '@comichub/ui';
import type { BookKind, FileRow } from '@comichub/api-client';
import { useClient } from '../lib/client.js';
import { useUiStore } from '../store/ui.js';
import { useSeriesFiles, useEditBook } from '../lib/queries.js';
import { LoadingState, ErrorState } from './Page.js';
import { EditBookDialog } from './EditBookDialog.js';
import { issueLabel } from '../lib/format.js';

/** Human label per kind for the row chip. */
const KIND_LABEL: Record<BookKind, string> = {
  issue: 'Issue',
  annual: 'Annual',
  special: 'Special',
  'one-shot': 'One-shot',
  tpb: 'Collected',
  gn: 'Graphic novel',
  variant: 'Variant',
  cover: 'Cover art',
};

/**
 * Manage files: the one screen that lists EVERY scanned file of a series — including files
 * the catalog otherwise hides (variant/cover extras, ignored files, and specials pulled out
 * of the numbered run). This is where a mis-scanned issue that vanished from the series page
 * (parsed to the wrong number, filed as the wrong kind, or ignored) can be found and fixed.
 */
export function ManageFilesDialog({
  seriesId,
  seriesName,
  onClose,
}: {
  seriesId: string;
  seriesName: string;
  onClose: () => void;
}) {
  const files = useSeriesFiles(seriesId);
  const addToast = useUiStore((s) => s.addToast);
  const quick = useEditBook();
  const [editing, setEditing] = useState<FileRow | null>(null);

  const toggleIgnore = (file: FileRow) =>
    quick.mutate(
      { bookId: file.id, patch: { ignored: !file.ignored } },
      {
        onSuccess: () =>
          addToast({
            tone: 'success',
            title: file.ignored ? 'File restored' : 'File hidden',
          }),
        onError: (e) =>
          addToast({
            tone: 'danger',
            title: 'Could not update',
            message: e instanceof Error ? e.message : 'Unexpected error.',
          }),
      },
    );

  const rows = files.data?.files ?? [];
  const hiddenCount = rows.filter((f) => f.ignored).length;
  const extraCount = rows.filter((f) => f.kind === 'variant' || f.kind === 'cover').length;
  const summary = [
    `${rows.length} ${rows.length === 1 ? 'file' : 'files'}`,
    hiddenCount ? `${hiddenCount} hidden` : undefined,
    extraCount ? `${extraCount} extra art` : undefined,
  ]
    .filter(Boolean)
    .join(' · ');

  return (
    <Dialog
      title="Manage files"
      width={720}
      onClose={onClose}
      footer={
        <Button variant="primary" onClick={onClose}>
          Done
        </Button>
      }
    >
      <p
        style={{
          margin: '0 0 16px',
          color: 'var(--text-secondary)',
          fontSize: 'var(--text-body)',
          lineHeight: 1.55,
        }}
      >
        Every file scanned for <span style={{ color: 'var(--text-primary)' }}>{seriesName}</span> —
        including hidden files and cover/variant extras that don't show on the series page. If an
        issue is missing, it's usually here under the wrong number or type. <strong>Edit</strong> a
        file to fix its number, title, or type; corrections are locked so rescans keep them.
      </p>

      {files.isLoading ? (
        <LoadingState />
      ) : files.isError ? (
        <ErrorState
          message={
            files.error instanceof Error ? files.error.message : 'Could not load the file list.'
          }
          onRetry={() => files.refetch()}
        />
      ) : rows.length === 0 ? (
        <p style={{ color: 'var(--text-tertiary)' }}>This series has no scanned files.</p>
      ) : (
        <>
          <div
            className="ch-mono"
            style={{
              fontSize: '0.66rem',
              letterSpacing: '0.04em',
              textTransform: 'uppercase',
              color: 'var(--text-tertiary)',
              marginBottom: 10,
            }}
          >
            {summary}
          </div>
          <div
            style={{
              display: 'flex',
              flexDirection: 'column',
              gap: 8,
              maxHeight: 460,
              overflowY: 'auto',
            }}
          >
            {rows.map((file) => (
              <FileRowItem
                key={file.id}
                file={file}
                busy={quick.isPending}
                onEdit={() => setEditing(file)}
                onToggleIgnore={() => toggleIgnore(file)}
              />
            ))}
          </div>
        </>
      )}

      {editing && <EditBookDialog detail={editing} onClose={() => setEditing(null)} />}
    </Dialog>
  );
}

function FileRowItem({
  file,
  busy,
  onEdit,
  onToggleIgnore,
}: {
  file: FileRow;
  busy: boolean;
  onEdit: () => void;
  onToggleIgnore: () => void;
}) {
  const client = useClient();
  const number = issueLabel(file.number);
  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 12,
        padding: '8px 10px',
        background: 'var(--surface-card)',
        border: '1px solid var(--border-hairline)',
        borderRadius: 'var(--radius-md)',
        opacity: file.ignored ? 0.6 : 1,
      }}
    >
      <div
        aria-hidden
        style={{
          width: 34,
          height: 50,
          flex: 'none',
          background: 'var(--surface-cover)',
          backgroundImage: `url(${client.coverUrl(file.id, 100)})`,
          backgroundSize: 'cover',
          backgroundPosition: 'center',
          borderRadius: 3,
        }}
      />
      <div style={{ flex: 1, minWidth: 0 }}>
        <div
          className="ch-mono"
          title={file.filePath ?? file.fileName}
          style={{
            fontSize: '0.72rem',
            color: 'var(--text-primary)',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
          }}
        >
          {file.fileName}
        </div>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 8,
            marginTop: 5,
            flexWrap: 'wrap',
          }}
        >
          <Badge mono>{number ? `#${file.number}` : 'no number'}</Badge>
          <Badge mono>{KIND_LABEL[file.kind]}</Badge>
          <span className="ch-mono" style={{ fontSize: '0.64rem', color: 'var(--text-tertiary)' }}>
            {file.pageCount} pp
          </span>
          {file.ignored && (
            <Badge tone="warning" mono>
              hidden
            </Badge>
          )}
          {file.isCorrupt && (
            <Badge tone="danger" mono>
              corrupt
            </Badge>
          )}
        </div>
      </div>
      <div style={{ display: 'flex', alignItems: 'center', gap: 6, flex: 'none' }}>
        {file.ignored ? (
          <Button size="sm" variant="ghost" icon="refresh" disabled={busy} onClick={onToggleIgnore}>
            Restore
          </Button>
        ) : (
          <Button size="sm" variant="ghost" disabled={busy} onClick={onToggleIgnore}>
            Hide
          </Button>
        )}
        <Button size="sm" variant="secondary" icon="edit" onClick={onEdit}>
          Edit
        </Button>
      </div>
    </div>
  );
}
