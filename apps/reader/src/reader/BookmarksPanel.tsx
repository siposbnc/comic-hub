import { useEffect, useState } from 'react';
import { Badge, Icon } from '@comichub/ui';
import type { Bookmark } from '@comichub/api-client';
import { useReaderStore } from './store.js';
import { useThumbUrl } from './usePageSnapshot.js';
import { IconButton } from '../ui/IconButton.js';

/** Compact "x ago" from a ms timestamp. */
function relTime(ms: number): string {
  const s = Math.max(0, Math.floor((Date.now() - ms) / 1000));
  if (s < 45) return 'just now';
  const m = Math.floor(s / 60);
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h ago`;
  const d = Math.floor(h / 24);
  if (d < 7) return `${d}d ago`;
  return `${Math.floor(d / 7)}w ago`;
}

const NOTE_MAX = 140;

/** The slide-over bookmarks list (mirrors the Settings panel), matching the reader
 *  bookmarks design preview: a current-page toggle row, then page-ordered rows with
 *  thumbnail, note, and edit/delete; jump on click. Connected mode only. */
export function BookmarksPanel() {
  const bookmarks = useReaderStore((s) => s.bookmarks);
  const currentPage = useReaderStore((s) => s.currentPage);
  const mode = useReaderStore((s) => s.mode);
  const close = useReaderStore((s) => s.setBookmarksOpen);
  const toggle = useReaderStore((s) => s.toggleBookmark);
  const goToPage = useReaderStore((s) => s.goToPage);
  const updateNote = useReaderStore((s) => s.updateBookmarkNote);
  const remove = useReaderStore((s) => s.removeBookmark);

  const [editingId, setEditingId] = useState<string | null>(null);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') close(false);
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [close]);

  const onCurrent = bookmarks.some((b) => b.page === currentPage);
  const jump = (page: number) => {
    goToPage(page);
    close(false);
  };

  return (
    <aside className="bm-panel" role="dialog" aria-label="Bookmarks">
      <div className="bm-head">
        <h2>Bookmarks</h2>
        <Badge mono>{String(bookmarks.length)}</Badge>
        {mode === 'connected' && (
          <span className="bm-synced">
            <Icon name="refresh" size={11} /> Synced
          </span>
        )}
        <span className="bm-head__spacer" />
        <IconButton icon="x" label="Close" size={16} onClick={() => close(false)} />
      </div>

      <button type="button" className="bm-current" onClick={() => void toggle()}>
        <Icon name="bookmark" size={16} color={onCurrent ? 'var(--accent)' : 'var(--paper-400)'} />
        <span className="bm-current__label">
          {onCurrent ? 'Remove bookmark' : 'Bookmark this page'}
        </span>
        <span className={`bm-current__page${onCurrent ? ' is-on' : ''}`}>
          p.{String(currentPage + 1).padStart(2, '0')}
        </span>
      </button>

      <div className="bm-list">
        {bookmarks.length === 0 ? (
          <div className="bm-empty">
            <div className="ch-halftone-duo bm-empty__art" />
            <div className="bm-empty__title">No bookmarks yet</div>
            <p className="bm-empty__body">
              Press <kbd>B</kbd> or tap the bookmark button to mark this page.
            </p>
          </div>
        ) : (
          bookmarks.map((bm) => (
            <BookmarkRow
              key={bm.id}
              bm={bm}
              current={bm.page === currentPage}
              editing={editingId === bm.id}
              onJump={() => jump(bm.page)}
              onEdit={() => setEditingId(bm.id)}
              onDelete={() => void remove(bm.id)}
              onSave={(note) => {
                void updateNote(bm.id, note);
                setEditingId(null);
              }}
              onCancel={() => setEditingId(null)}
            />
          ))
        )}
      </div>
    </aside>
  );
}

function RowThumb({ page }: { page: number }) {
  const thumbs = useReaderStore((s) => s.thumbs);
  const url = useThumbUrl(page);
  thumbs?.ensure(page);
  return (
    <div className="ch-reg bm-row__thumb">
      {url ? <img src={url} alt="" draggable={false} /> : <span className="bm-row__thumb-ph" />}
    </div>
  );
}

function BookmarkRow({
  bm,
  current,
  editing,
  onJump,
  onEdit,
  onDelete,
  onSave,
  onCancel,
}: {
  bm: Bookmark;
  current: boolean;
  editing: boolean;
  onJump: () => void;
  onEdit: () => void;
  onDelete: () => void;
  onSave: (note: string) => void;
  onCancel: () => void;
}) {
  const [draft, setDraft] = useState(bm.note);
  useEffect(() => {
    if (editing) setDraft(bm.note);
  }, [editing, bm.note]);

  return (
    <div
      className={`bm-row${current ? ' is-current' : ''}`}
      onClick={() => !editing && onJump()}
      role="button"
      tabIndex={0}
    >
      <RowThumb page={bm.page} />
      <div className="bm-row__body">
        <div className="bm-row__top">
          <span className="bm-row__page">p.{String(bm.page + 1).padStart(2, '0')}</span>
          {current && <span className="bm-row__current">Current</span>}
          <span className="bm-row__spacer" />
          <span className="bm-row__ts">{relTime(bm.updatedAt)}</span>
        </div>

        {editing ? (
          <div className="bm-edit" onClick={(e) => e.stopPropagation()}>
            <input
              autoFocus
              value={draft}
              maxLength={NOTE_MAX}
              placeholder="Add a note…"
              onChange={(e) => setDraft(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') onSave(draft);
                if (e.key === 'Escape') onCancel();
              }}
            />
            <div className="bm-edit__actions">
              <button type="button" className="bm-edit__save" onClick={() => onSave(draft)}>
                Save
              </button>
              <button type="button" className="bm-edit__cancel" onClick={onCancel}>
                Cancel
              </button>
              <span className="bm-row__spacer" />
              <span className="bm-edit__count">
                {draft.length}/{NOTE_MAX}
              </span>
            </div>
          </div>
        ) : (
          <div className="bm-row__noteline">
            <p className={`bm-row__note${bm.note ? '' : ' is-empty'}`}>
              {bm.note || 'Add a note…'}
            </p>
            <div className="bm-row__actions">
              <IconButton
                icon="edit"
                label="Edit note"
                size={15}
                onClick={(e) => {
                  e.stopPropagation();
                  onEdit();
                }}
              />
              <IconButton
                icon="trash"
                label="Delete"
                size={15}
                onClick={(e) => {
                  e.stopPropagation();
                  onDelete();
                }}
              />
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
