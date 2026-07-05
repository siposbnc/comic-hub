import { useRef, useState } from 'react';
import { useNavigate } from '@tanstack/react-router';
import { useQueries } from '@tanstack/react-query';
import { Button, Icon, Input, EmptyState } from '@comichub/ui';
import type { ReadingList, ReadingListDetail } from '@comichub/api-client';
import { useClient } from '../lib/client.js';
import { qk, useReadingLists, useCreateReadingList } from '../lib/queries.js';
import { LoadingState, ErrorState } from '../components/Page.js';
import { relativeTime } from '../lib/format.js';

/**
 * Reading lists index (per the design preview): a card grid with a fanned cover collage,
 * "In queue" / "N missing" pills, per-list read progress, and a dashed create tile.
 */
export function ReadingLists() {
  const navigate = useNavigate();
  const client = useClient();
  const q = useReadingLists();
  const create = useCreateReadingList();
  const [name, setName] = useState('');
  // The DS Input doesn't forward refs; focus the real <input> through a wrapper node.
  const nameWrapRef = useRef<HTMLDivElement>(null);
  const focusName = () => nameWrapRef.current?.querySelector('input')?.focus();

  const lists = q.data ?? [];

  // Each card's covers / read % / missing count come from the list's detail (the handoff
  // sanctions a per-list detail fetch; the detail view shares the same cache entry).
  const details = useQueries({
    queries: lists.map((l) => ({
      queryKey: qk.readingList(l.id),
      queryFn: () => client.readingList(l.id),
    })),
  });
  const detailOf = new Map<string, ReadingListDetail>();
  lists.forEach((l, i) => {
    const d = details[i]?.data;
    if (d) detailOf.set(l.id, d);
  });

  const submit = () => {
    const n = name.trim();
    if (!n || create.isPending) return;
    create.mutate(n);
    setName('');
  };

  if (q.isLoading) {
    return (
      <div style={{ padding: 'var(--pad-screen)' }}>
        <LoadingState />
      </div>
    );
  }
  if (q.isError) {
    return (
      <div style={{ padding: 'var(--pad-screen)' }}>
        <ErrorState
          message={q.error instanceof Error ? q.error.message : 'Could not load reading lists.'}
          onRetry={() => q.refetch()}
        />
      </div>
    );
  }

  return (
    <div style={{ padding: 'var(--pad-screen)', maxWidth: 'var(--content-max)', margin: '0 auto' }}>
      {/* Header */}
      <div
        style={{
          display: 'flex',
          alignItems: 'flex-end',
          gap: 24,
          flexWrap: 'wrap',
          marginBottom: 24,
        }}
      >
        <div style={{ flex: 1, minWidth: 280 }}>
          <div
            className="ch-mono"
            style={{
              fontSize: '0.66rem',
              fontWeight: 600,
              letterSpacing: '0.16em',
              textTransform: 'uppercase',
              color: 'var(--accent)',
            }}
          >
            Lists
          </div>
          <h1
            style={{
              margin: '8px 0 0',
              fontFamily: 'var(--font-display)',
              fontWeight: 800,
              fontSize: 'var(--text-display-l)',
              letterSpacing: '-0.01em',
              color: 'var(--text-primary)',
            }}
          >
            Reading lists
          </h1>
          <p className="ch-label" style={{ margin: '8px 0 0', color: 'var(--text-tertiary)' }}>
            {lists.length} list{lists.length === 1 ? '' : 's'} · curated cross-series reading orders
          </p>
        </div>
        <div style={{ flex: 'none', display: 'flex', gap: 8, alignItems: 'center' }}>
          <div ref={nameWrapRef} style={{ width: 220 }}>
            <Input
              icon="plus"
              placeholder="New list name…"
              aria-label="New reading list name"
              value={name}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setName(e.target.value)}
              onKeyDown={(e: React.KeyboardEvent) => {
                if (e.key === 'Enter') submit();
              }}
            />
          </div>
          <Button icon="plus" onClick={submit} disabled={!name.trim() || create.isPending}>
            Create
          </Button>
        </div>
      </div>

      {lists.length === 0 ? (
        <EmptyState title="Nothing here yet">
          Make a reading list to queue up issues to read next.
        </EmptyState>
      ) : (
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fill, minmax(258px, 1fr))',
            gap: 20,
          }}
        >
          {lists.map((l) => (
            <ListCard
              key={l.id}
              list={l}
              detail={detailOf.get(l.id)}
              coverUrl={(bookId) => client.coverUrl(bookId, 200)}
              onOpen={() => navigate({ to: '/reading-lists/$id', params: { id: l.id } })}
            />
          ))}

          {/* Create tile */}
          <button
            type="button"
            onClick={focusName}
            style={{
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              justifyContent: 'center',
              gap: 10,
              minHeight: 260,
              cursor: 'pointer',
              background: 'transparent',
              border: '1px dashed var(--border-strong)',
              borderRadius: 'var(--radius-lg)',
              color: 'var(--text-tertiary)',
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.borderColor = 'var(--accent)';
              e.currentTarget.style.color = 'var(--accent)';
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.borderColor = 'var(--border-strong)';
              e.currentTarget.style.color = 'var(--text-tertiary)';
            }}
          >
            <span
              style={{
                width: 44,
                height: 44,
                borderRadius: '50%',
                border: '1px solid currentColor',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              <Icon name="plus" size={20} color="currentColor" />
            </span>
            <span style={{ fontFamily: 'var(--font-body)', fontWeight: 600, fontSize: '0.9rem' }}>
              New reading list
            </span>
            <span className="ch-mono" style={{ fontSize: '0.62rem', color: 'var(--paper-600)' }}>
              Name it, add issues later
            </span>
          </button>
        </div>
      )}
    </div>
  );
}

function ListCard({
  list,
  detail,
  coverUrl,
  onOpen,
}: {
  list: ReadingList;
  detail?: ReadingListDetail;
  coverUrl: (bookId: string) => string;
  onOpen: () => void;
}) {
  const items = detail?.items ?? [];
  const linked = items.filter((it) => it.book);
  const missing = items.length - linked.length;
  const readCount = linked.filter((it) => it.book!.progress?.status === 'read').length;
  const pct = linked.length ? Math.round((readCount / linked.length) * 100) : 0;
  const covers = linked.slice(0, 4).map((it) => coverUrl(it.book!.id));

  return (
    <button
      type="button"
      onClick={onOpen}
      style={{
        display: 'block',
        textAlign: 'left',
        padding: 0,
        cursor: 'pointer',
        background: 'var(--surface-raised)',
        border: '1px solid var(--border-hairline)',
        borderRadius: 'var(--radius-lg)',
        overflow: 'hidden',
        transition: 'border-color 120ms',
      }}
      onMouseEnter={(e) => (e.currentTarget.style.borderColor = 'var(--border-strong)')}
      onMouseLeave={(e) => (e.currentTarget.style.borderColor = 'var(--border-hairline)')}
    >
      <div style={{ position: 'relative' }}>
        <CoverFan covers={covers} />
        {list.active && (
          <span
            style={{
              position: 'absolute',
              top: 10,
              left: 10,
              display: 'inline-flex',
              alignItems: 'center',
              gap: 5,
              height: 22,
              padding: '0 9px',
              borderRadius: 'var(--radius-pill)',
              background: 'var(--accent)',
              color: 'var(--text-on-accent)',
              fontFamily: 'var(--font-mono)',
              fontSize: '0.58rem',
              letterSpacing: '0.06em',
              textTransform: 'uppercase',
              boxShadow: '0 2px 8px rgba(0,0,0,.4)',
            }}
          >
            <Icon name="book-open" size={11} color="var(--text-on-accent)" /> In queue
          </span>
        )}
        {missing > 0 && (
          <span
            style={{
              position: 'absolute',
              top: 10,
              right: 10,
              display: 'inline-flex',
              alignItems: 'center',
              gap: 5,
              height: 22,
              padding: '0 9px',
              borderRadius: 'var(--radius-pill)',
              background: 'color-mix(in oklab, var(--warning) 92%, #000)',
              color: 'var(--ink-900)',
              fontFamily: 'var(--font-mono)',
              fontSize: '0.58rem',
              letterSpacing: '0.04em',
              boxShadow: '0 2px 8px rgba(0,0,0,.4)',
            }}
          >
            <Icon name="alert-triangle" size={11} color="var(--ink-900)" /> {missing} missing
          </span>
        )}
      </div>
      <div style={{ padding: '14px 16px 16px' }}>
        <div
          style={{
            fontFamily: 'var(--font-display)',
            fontWeight: 700,
            fontSize: '1.02rem',
            letterSpacing: '-0.01em',
            color: 'var(--paper-100)',
            whiteSpace: 'nowrap',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
          }}
        >
          {list.name}
        </div>
        <div
          className="ch-mono"
          style={{
            fontSize: '0.66rem',
            color: 'var(--text-tertiary)',
            marginTop: 6,
            display: 'flex',
            alignItems: 'center',
            gap: 8,
          }}
        >
          <span>
            {list.bookCount} issue{list.bookCount === 1 ? '' : 's'}
          </span>
          <span
            style={{ width: 3, height: 3, borderRadius: '50%', background: 'var(--paper-600)' }}
          />
          <span>{pct}% read</span>
        </div>
        <div className="ch-progress" style={{ borderRadius: 999, height: 5, marginTop: 12 }}>
          <span style={{ width: `${pct}%`, borderRadius: 999 }} />
        </div>
        <div
          className="ch-mono"
          style={{ fontSize: '0.6rem', color: 'var(--paper-600)', marginTop: 10 }}
        >
          Updated {relativeTime(list.updatedAt)}
        </div>
      </div>
    </button>
  );
}

/** Up to four covers fanned like pulls from a longbox (per the design preview). */
function CoverFan({ covers }: { covers: string[] }) {
  const arr = covers.slice(0, 4);
  const mid = (arr.length - 1) / 2;
  return (
    <div
      style={{
        position: 'relative',
        height: 158,
        background: 'linear-gradient(180deg, var(--ink-900), var(--ink-800))',
        overflow: 'hidden',
      }}
    >
      {arr.length === 0 && (
        <div
          style={{
            position: 'absolute',
            inset: 0,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}
        >
          <Icon name="book-open" size={22} color="var(--paper-600)" />
        </div>
      )}
      {arr.map((c, i) => {
        const off = i - mid;
        return (
          <img
            key={i}
            src={c}
            alt=""
            style={{
              position: 'absolute',
              left: '50%',
              top: 20,
              width: 86,
              height: 129,
              objectFit: 'cover',
              transform: `translateX(calc(-50% + ${off * 40}px)) rotate(${off * 6}deg)`,
              transformOrigin: 'bottom center',
              boxShadow: '0 8px 22px rgba(0,0,0,.55)',
              outline: '1px solid rgba(0,0,0,.45)',
              zIndex: i,
            }}
          />
        );
      })}
    </div>
  );
}
