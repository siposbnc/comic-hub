import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Dialog, Button, Icon } from '@comichub/ui';
import type { BookCard } from '@comichub/api-client';
import { useClient } from '../lib/client.js';
import { useUiStore } from '../store/ui.js';
import { qk } from '../lib/queries.js';
import { issueLabel } from '../lib/format.js';

/** A group of books whose parsed issue numbers collide. */
export interface DuplicateGroup {
  number: string;
  books: BookCard[];
}

/**
 * Resolver for duplicate issue numbers: each colliding file gets an editable number,
 * pre-filled with a subtitle guess from its file name where one is detectable. A labeled
 * number ("Futures End 1") files the book as a Special past the numbered run; the server
 * locks the corrected field so rescans and matches keep it.
 */
export function ResolveDuplicatesDialog({
  seriesId,
  groups,
  onClose,
}: {
  seriesId: string;
  groups: DuplicateGroup[];
  onClose: () => void;
}) {
  const client = useClient();
  const qc = useQueryClient();
  const addToast = useUiStore((s) => s.addToast);

  const [values, setValues] = useState<Record<string, string>>(() => {
    const init: Record<string, string> = {};
    for (const g of groups) {
      for (const b of g.books) {
        init[b.id] = suggestNumber(b) ?? b.number ?? '';
      }
    }
    return init;
  });

  const edits = groups
    .flatMap((g) => g.books)
    .filter((b) => {
      const v = values[b.id]?.trim();
      return v && v !== (b.number ?? '');
    });

  const apply = useMutation({
    mutationFn: async () => {
      for (const b of edits) {
        await client.setBookNumber(b.id, values[b.id]!.trim());
      }
      return edits.length;
    },
    onSuccess: (n) => {
      qc.invalidateQueries({ queryKey: qk.seriesDetail(seriesId) });
      qc.invalidateQueries({ queryKey: ['series'] });
      addToast({
        tone: 'success',
        title: 'Issue numbers updated',
        message: `${n} ${n === 1 ? 'book' : 'books'} corrected. The fields are locked, so rescans keep them.`,
      });
      onClose();
    },
    onError: (e) =>
      addToast({
        tone: 'danger',
        title: 'Could not update',
        message: e instanceof Error ? e.message : 'Unexpected error.',
      }),
  });

  return (
    <Dialog
      title="Resolve duplicate issues"
      width={640}
      onClose={onClose}
      footer={
        <>
          <Button variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button
            variant="primary"
            icon="check"
            disabled={edits.length === 0 || apply.isPending}
            onClick={() => apply.mutate()}
          >
            {apply.isPending
              ? 'Applying…'
              : `Apply ${edits.length > 0 ? edits.length : ''} ${edits.length === 1 ? 'change' : 'changes'}`.replace(
                  '  ',
                  ' ',
                )}
          </Button>
        </>
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
        These files parsed to the same issue number — usually a special or tie-in read as a regular
        issue. Give each file its real number: a labeled number like{' '}
        <span className="ch-mono" style={{ color: 'var(--text-primary)' }}>
          Futures End 1
        </span>{' '}
        files it under Specials, out of the numbered run. Corrections are locked, so rescans keep
        them.
      </p>
      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          gap: 20,
          maxHeight: 440,
          overflowY: 'auto',
        }}
      >
        {groups.map((g) => (
          <div key={g.number}>
            <div
              className="ch-mono"
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                fontSize: '0.66rem',
                letterSpacing: '0.04em',
                textTransform: 'uppercase',
                color: 'var(--warning)',
                marginBottom: 8,
              }}
            >
              <Icon name="alert-triangle" size={13} color="var(--warning)" />
              {issueLabel(g.number)} · {g.books.length} files
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              {g.books.map((b) => (
                <BookRow
                  key={b.id}
                  book={b}
                  value={values[b.id] ?? ''}
                  changed={
                    Boolean(values[b.id]?.trim()) && values[b.id]!.trim() !== (b.number ?? '')
                  }
                  onChange={(v) => setValues((s) => ({ ...s, [b.id]: v }))}
                />
              ))}
            </div>
          </div>
        ))}
      </div>
    </Dialog>
  );
}

function BookRow({
  book,
  value,
  changed,
  onChange,
}: {
  book: BookCard;
  value: string;
  changed: boolean;
  onChange: (v: string) => void;
}) {
  const client = useClient();
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
      }}
    >
      <div
        aria-hidden
        style={{
          width: 34,
          height: 50,
          flex: 'none',
          background: 'var(--surface-cover)',
          backgroundImage: `url(${client.coverUrl(book.id, 100)})`,
          backgroundSize: 'cover',
          backgroundPosition: 'center',
          borderRadius: 3,
        }}
      />
      <div style={{ flex: 1, minWidth: 0 }}>
        <div
          className="ch-mono"
          title={book.fileName}
          style={{
            fontSize: '0.72rem',
            color: 'var(--text-primary)',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
          }}
        >
          {book.fileName ?? issueLabel(book.number) ?? book.id}
        </div>
        <div
          className="ch-mono"
          style={{ fontSize: '0.64rem', color: 'var(--text-tertiary)', marginTop: 3 }}
        >
          {book.pageCount} pages
        </div>
      </div>
      <input
        value={value}
        onChange={(e) => onChange(e.target.value)}
        aria-label={`Issue number for ${book.fileName ?? book.id}`}
        style={{
          width: 150,
          flex: 'none',
          height: 32,
          padding: '0 10px',
          background: 'var(--surface-raised)',
          border: `1px solid ${changed ? 'var(--accent)' : 'var(--border-strong)'}`,
          borderRadius: 'var(--radius-md)',
          outline: 'none',
          color: changed ? 'var(--accent)' : 'var(--text-primary)',
          fontFamily: 'var(--font-body)',
          fontSize: '0.82rem',
        }}
      />
    </div>
  );
}

/**
 * Guesses the intended number from the file name: strips the extension and parenthesized
 * noise, then reads "<prefix> - <subtitle> NNN" as "<subtitle> NNN" ("Wonder Woman -
 * Futures End 001" -> "Futures End 1"). Returns undefined when the name has no subtitle —
 * that file keeps its current number and reads as the real issue.
 */
export function suggestNumber(book: BookCard): string | undefined {
  if (!book.fileName) return undefined;
  const stem = book.fileName.replace(/\.[^.]+$/, '');
  const cleaned = stem
    .replace(/[([][^)\]]*[)\]]/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();
  const m = /^(.*?)\s*[-–—]\s*(.*?)\s+#?0*(\d+(?:\.\d+)?)$/.exec(cleaned);
  if (!m || !m[2]) return undefined;
  return `${m[2].trim()} ${m[3]}`;
}
