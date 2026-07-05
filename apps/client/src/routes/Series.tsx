import { useState } from 'react';
import { getRouteApi, useNavigate } from '@tanstack/react-router';
import {
  Button,
  ProgressBar,
  Tabs,
  CoverCard,
  EmptyState,
  IconButton,
  Icon,
  Tag,
} from '@comichub/ui';
import type { BookCard, SeriesDetail, GroupingCard, MetadataState } from '@comichub/api-client';
import { useClient } from '../lib/client.js';
import { useSeriesDetail, useRescanSeries } from '../lib/queries.js';
import { useReadLaunch } from '../lib/launch.js';
import { useUiStore } from '../store/ui.js';
import { LoadingState, ErrorState } from '../components/Page.js';
import { MatchDialog } from '../components/MatchDialog.js';
import { issueLabel, resumePage, toCoverStatus, progressFraction } from '../lib/format.js';

const route = getRouteApi('/series/$id');

/** Series detail: a full-bleed cover-wash hero with a resume CTA over the run of issues. */
export function Series() {
  const { id } = route.useParams();
  const series = useSeriesDetail(id);

  if (series.isLoading)
    return (
      <Centered>
        <LoadingState />
      </Centered>
    );
  if (series.isError) {
    return (
      <Centered>
        <ErrorState
          message={
            series.error instanceof Error ? series.error.message : 'Could not load this series.'
          }
          onRetry={() => series.refetch()}
        />
      </Centered>
    );
  }
  if (!series.data) return null;
  return <SeriesView detail={series.data} />;
}

function SeriesView({ detail }: { detail: SeriesDetail }) {
  const client = useClient();
  const navigate = useNavigate();
  const launch = useReadLaunch();
  const addToast = useUiStore((s) => s.addToast);
  const rescan = useRescanSeries();
  const [tab, setTab] = useState('issues');
  const [matching, setMatching] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);
  const [confirmingRescan, setConfirmingRescan] = useState(false);

  const onRescan = async () => {
    setConfirmingRescan(false);
    try {
      await rescan.mutateAsync(detail.id);
      addToast({
        tone: 'info',
        title: `Rescanning ${detail.name}`,
        message: 'Re-cataloging its files from disk. Reading lists keep every issue.',
      });
      navigate({ to: '/' });
    } catch (err) {
      addToast({
        tone: 'danger',
        title: 'Could not rescan',
        message: err instanceof Error ? err.message : 'Unexpected error.',
      });
    }
  };

  const resumeBook =
    detail.books.find((b) => b.progress?.status === 'in_progress') ??
    detail.books.find((b) => b.progress?.status !== 'read') ??
    detail.books[0];
  const isResuming = resumeBook?.progress?.status === 'in_progress';
  const coverBook = resumeBook ?? detail.books[0];
  const coverUrl = coverBook ? client.coverUrl(coverBook.id, 600) : undefined;
  const format = detail.books[0]?.format?.toUpperCase();

  const meta = [
    detail.publisher,
    detail.year,
    `${detail.bookCount} issues`,
    detail.genres?.length ? detail.genres.join(' · ') : undefined,
  ]
    .filter(Boolean)
    .join(' · ');

  return (
    <div>
      {/* Hero. zIndex lifts it (and the overflow menu inside) above the body content;
          only the background wash is clipped, so the popover can escape the hero. */}
      <div
        style={{
          position: 'relative',
          zIndex: 3,
          borderBottom: '1px solid var(--border-hairline)',
        }}
      >
        <div aria-hidden style={{ position: 'absolute', inset: 0, overflow: 'hidden' }}>
          {coverUrl && (
            <div
              style={{
                position: 'absolute',
                inset: 0,
                backgroundImage: `url(${coverUrl})`,
                backgroundSize: 'cover',
                backgroundPosition: 'center',
                filter: 'blur(40px) saturate(1.3)',
                transform: 'scale(1.2)',
                opacity: 0.5,
              }}
            />
          )}
          <div
            style={{
              position: 'absolute',
              inset: 0,
              background:
                'linear-gradient(180deg, color-mix(in oklab, var(--ink-900) 70%, transparent), var(--ink-900) 88%)',
            }}
          />
        </div>
        <div
          style={{
            position: 'relative',
            display: 'flex',
            gap: 28,
            padding: 'var(--pad-screen)',
            maxWidth: 'var(--content-max)',
            margin: '0 auto',
          }}
        >
          <div className="ch-reg" style={{ flex: 'none' }}>
            <div
              style={{
                width: 200,
                aspectRatio: 'var(--cover-aspect)',
                background: 'var(--surface-cover)',
                backgroundImage: coverUrl ? `url(${coverUrl})` : undefined,
                backgroundSize: 'cover',
                backgroundPosition: 'center',
              }}
            />
          </div>
          <div style={{ flex: 1, minWidth: 0, paddingTop: 8 }}>
            <h1
              style={{
                margin: 0,
                fontFamily: 'var(--font-display)',
                fontWeight: 800,
                fontSize: 'var(--text-display-xl)',
                lineHeight: 1.02,
                letterSpacing: '-0.015em',
                color: 'var(--text-primary)',
              }}
            >
              {detail.name}
            </h1>
            <div style={{ marginTop: 14 }}>
              <MetaChip state={detail.metadataState} onMatch={() => setMatching(true)} />
            </div>
            {meta && (
              <p
                className="ch-label"
                style={{ margin: '12px 0 0', color: 'var(--text-secondary)' }}
              >
                {meta}
              </p>
            )}
            {detail.summary && (
              <p
                style={{
                  margin: '16px 0 0',
                  maxWidth: 560,
                  color: 'var(--text-secondary)',
                  fontSize: 'var(--text-body)',
                  lineHeight: 1.55,
                }}
              >
                {detail.summary}
              </p>
            )}
            {detail.bookCount > 0 && (
              <div style={{ maxWidth: 320, marginTop: 18 }}>
                <ProgressBar
                  value={detail.readCount}
                  max={detail.bookCount}
                  label="Read"
                  showCount
                />
              </div>
            )}
            <div style={{ display: 'flex', gap: 10, marginTop: 20 }}>
              {resumeBook && (
                <Button
                  icon="book-open"
                  onClick={() => launch(resumeBook.id, resumePage(resumeBook.progress))}
                >
                  {isResuming
                    ? `Continue · ${issueLabel(resumeBook.number) ?? ''}`
                    : 'Read first issue'}
                </Button>
              )}
              {detail.metadataState === 'matched' && (
                <Button variant="ghost" icon="search" onClick={() => setMatching(true)}>
                  Re-match
                </Button>
              )}
              <div style={{ position: 'relative' }}>
                <IconButton
                  icon="more-horizontal"
                  label="More"
                  variant="solid"
                  onClick={() => setMenuOpen((o) => !o)}
                />
                {menuOpen && (
                  <>
                    <div
                      role="presentation"
                      onClick={() => setMenuOpen(false)}
                      style={{ position: 'fixed', inset: 0, zIndex: 41 }}
                    />
                    <div
                      role="menu"
                      style={{
                        position: 'absolute',
                        top: 'calc(100% + 6px)',
                        right: 0,
                        zIndex: 42,
                        minWidth: 216,
                        padding: 6,
                        background: 'var(--surface-raised)',
                        border: '1px solid var(--border-hairline)',
                        borderRadius: 'var(--radius-md)',
                        boxShadow: 'var(--shadow-popover)',
                      }}
                    >
                      <button
                        type="button"
                        role="menuitem"
                        onClick={() => {
                          setMenuOpen(false);
                          setConfirmingRescan(true);
                        }}
                        onMouseEnter={(e) => (e.currentTarget.style.background = 'var(--ink-700)')}
                        onMouseLeave={(e) => (e.currentTarget.style.background = 'none')}
                        style={{
                          width: '100%',
                          display: 'flex',
                          alignItems: 'center',
                          gap: 10,
                          padding: '9px 10px',
                          background: 'none',
                          border: 'none',
                          borderRadius: 5,
                          cursor: 'pointer',
                          textAlign: 'left',
                          color: 'var(--text-primary)',
                          fontFamily: 'var(--font-body)',
                          fontSize: '0.86rem',
                        }}
                      >
                        <Icon name="refresh" size={16} color="var(--paper-400)" /> Rescan series
                      </button>
                    </div>
                  </>
                )}
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Body */}
      <div
        style={{ padding: 'var(--pad-screen)', maxWidth: 'var(--content-max)', margin: '0 auto' }}
      >
        <Tabs
          value={tab}
          onChange={setTab}
          tabs={[
            { value: 'issues', label: 'Issues', count: detail.bookCount },
            { value: 'volumes', label: 'Volumes', count: detail.volumes?.length ?? 0 },
            { value: 'arcs', label: 'Story Arcs', count: detail.storyArcs?.length ?? 0 },
            { value: 'details', label: 'Details' },
          ]}
          style={{ marginBottom: 22 }}
        />

        {tab === 'issues' ? (
          detail.books.length === 0 ? (
            <EmptyState title="No issues yet">This series has no scanned issues.</EmptyState>
          ) : (
            <div
              style={{
                display: 'grid',
                gridTemplateColumns: 'repeat(auto-fill, minmax(132px, 1fr))',
                gap: 16,
              }}
            >
              {detail.books.map((book) => (
                <IssueCover key={book.id} book={book} seriesName={detail.name} />
              ))}
            </div>
          )
        ) : tab === 'volumes' ? (
          detail.volumes && detail.volumes.length > 0 ? (
            <div
              style={{
                display: 'grid',
                gridTemplateColumns: 'repeat(auto-fill, minmax(240px, 1fr))',
                gap: 18,
              }}
            >
              {detail.volumes.map((v) => (
                <GroupingCardButton
                  key={v.id}
                  cover={detail.books[0] ? client.coverUrl(detail.books[0].id, 200) : undefined}
                  name={v.name}
                  meta={[v.year || undefined, `${v.issueCount} issues`].filter(Boolean).join(' · ')}
                  description={v.description}
                  onClick={() =>
                    navigate({
                      to: '/series/$id/volumes/$volume',
                      params: { id: detail.id, volume: v.id },
                    })
                  }
                />
              ))}
            </div>
          ) : (
            <MatchToPopulate
              kind="volumes"
              state={detail.metadataState}
              onMatch={() => setMatching(true)}
            />
          )
        ) : tab === 'arcs' ? (
          detail.storyArcs && detail.storyArcs.length > 0 ? (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              {detail.storyArcs.map((a) => (
                <ArcRow
                  key={a.id}
                  arc={a}
                  onClick={() =>
                    navigate({
                      to: '/series/$id/story-arcs/$arcId',
                      params: { id: detail.id, arcId: a.id },
                    })
                  }
                />
              ))}
            </div>
          ) : (
            <MatchToPopulate
              kind="story arcs"
              state={detail.metadataState}
              onMatch={() => setMatching(true)}
            />
          )
        ) : (
          <div
            style={{ display: 'grid', gridTemplateColumns: '1.2fr 1fr', gap: 40, maxWidth: 820 }}
          >
            <div>
              {detail.summary && (
                <p
                  style={{
                    margin: '0 0 20px',
                    color: 'var(--text-secondary)',
                    lineHeight: 1.6,
                  }}
                >
                  {detail.summary}
                </p>
              )}
              <div
                style={{
                  display: 'grid',
                  gridTemplateColumns: '120px 1fr',
                  gap: '9px 20px',
                  fontSize: 'var(--text-body)',
                }}
              >
                {(
                  [
                    ['Publisher', detail.publisher],
                    ['Year', detail.year ? String(detail.year) : undefined],
                    ['Issues', String(detail.bookCount)],
                    ['Format', format],
                    [
                      'Reading direction',
                      detail.readingDir === 'rtl' ? 'Right to left' : 'Left to right',
                    ],
                  ] as const
                )
                  .filter(([, v]) => Boolean(v))
                  .map(([k, v]) => (
                    <Detail key={k} label={k} value={v as string} />
                  ))}
              </div>
            </div>
            <div>
              {detail.genres && detail.genres.length > 0 && (
                <>
                  <div
                    className="ch-label"
                    style={{ color: 'var(--text-tertiary)', marginBottom: 10 }}
                  >
                    Genres
                  </div>
                  <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: 22 }}>
                    {detail.genres.map((g) => (
                      <Tag key={g}>{g}</Tag>
                    ))}
                  </div>
                </>
              )}
              <div className="ch-label" style={{ color: 'var(--text-tertiary)', marginBottom: 10 }}>
                Characters
              </div>
              {detail.characters && detail.characters.length > 0 ? (
                <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
                  {detail.characters.map((c) => (
                    <Tag key={c}>{c}</Tag>
                  ))}
                </div>
              ) : (
                <p style={{ color: 'var(--text-tertiary)', fontSize: '0.82rem' }}>—</p>
              )}
            </div>
          </div>
        )}
      </div>

      {matching && (
        <MatchDialog
          seriesId={detail.id}
          seriesName={detail.name}
          onClose={() => setMatching(false)}
        />
      )}
      {confirmingRescan && (
        <RescanConfirmDialog
          name={detail.name}
          onClose={() => setConfirmingRescan(false)}
          onConfirm={onRescan}
        />
      )}
    </div>
  );
}

/**
 * Confirm for the semi-destructive series rescan (per the design preview): one calm
 * sentence about what resets, and an explicit note that reading lists are safe.
 */
function RescanConfirmDialog({
  name,
  onClose,
  onConfirm,
}: {
  name: string;
  onClose: () => void;
  onConfirm: () => void;
}) {
  return (
    <div
      role="presentation"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) onClose();
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
      <div
        role="alertdialog"
        aria-modal="true"
        aria-label={`Rescan ${name}?`}
        style={{
          width: 'min(460px, 100%)',
          background: 'var(--surface-raised)',
          border: '1px solid var(--border-strong)',
          borderRadius: 'var(--radius-lg)',
          boxShadow: 'var(--shadow-popover)',
          overflow: 'hidden',
        }}
      >
        <div style={{ padding: '22px 24px 6px', display: 'flex', gap: 14 }}>
          <span
            style={{
              flex: 'none',
              width: 38,
              height: 38,
              borderRadius: '50%',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              background: 'color-mix(in oklab, var(--warning) 18%, transparent)',
            }}
          >
            <Icon name="refresh" size={19} color="var(--warning)" />
          </span>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div
              style={{
                fontFamily: 'var(--font-display)',
                fontWeight: 800,
                fontSize: '1.15rem',
                letterSpacing: '-0.01em',
                color: 'var(--text-primary)',
              }}
            >
              Rescan {name}?
            </div>
            <p
              style={{
                margin: '10px 0 0',
                color: 'var(--text-secondary)',
                fontSize: 'var(--text-body)',
                lineHeight: 1.55,
              }}
            >
              ComicHub deletes this series and re-catalogs it from the files on disk. Local metadata
              and your read progress for {name} reset — your reading lists keep every issue.
            </p>
          </div>
        </div>
        <div
          style={{
            margin: '18px 24px 0',
            padding: '11px 14px',
            display: 'flex',
            alignItems: 'center',
            gap: 9,
            background: 'var(--accent-soft)',
            borderRadius: 'var(--radius-md)',
          }}
        >
          <Icon name="bookmark" size={15} color="var(--accent)" />
          <span style={{ fontSize: '0.82rem', color: 'var(--text-secondary)' }}>
            Reading-list entries hold their place and re-attach automatically once the files come
            back.
          </span>
        </div>
        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 10, padding: '20px 24px' }}>
          <Button variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button variant="primary" icon="refresh" onClick={onConfirm}>
            Rescan series
          </Button>
        </div>
      </div>
    </div>
  );
}

/** One issue as a small CoverCard: clicking it opens the reader at its resume page; a
 *  Details button appears on hover to open the issue's detail page. */
function IssueCover({ book, seriesName }: { book: BookCard; seriesName: string }) {
  const client = useClient();
  const navigate = useNavigate();
  const launch = useReadLaunch();
  const [hover, setHover] = useState(false);
  const number = issueLabel(book.number);
  return (
    <div
      style={{ position: 'relative' }}
      onMouseEnter={() => setHover(true)}
      onMouseLeave={() => setHover(false)}
    >
      <CoverCard
        cover={client.coverUrl(book.id, 300)}
        title={number ?? book.title ?? seriesName}
        subtitle={`${book.pageCount} pp`}
        number={number}
        status={toCoverStatus(book.progress?.status)}
        progress={progressFraction(book.progress)}
        size="s"
        style={{ width: '100%' }}
        onClick={() => launch(book.id, resumePage(book.progress))}
      />
      {hover && (
        <div style={{ position: 'absolute', top: 6, right: 6, zIndex: 1 }}>
          <IconButton
            icon="info"
            label="Issue details"
            variant="solid"
            size="sm"
            onClick={(e: React.MouseEvent) => {
              e.stopPropagation();
              navigate({ to: '/book/$id', params: { id: book.id } });
            }}
          />
        </div>
      )}
    </div>
  );
}

/** Series metadata-state chip: matched (accent) or incomplete/no-match (warning + Match). */
function MetaChip({ state, onMatch }: { state?: MetadataState; onMatch: () => void }) {
  if (state === 'matched') {
    return (
      <span
        className="ch-mono"
        style={{
          display: 'inline-flex',
          alignItems: 'center',
          gap: 6,
          height: 24,
          padding: '0 10px',
          borderRadius: 'var(--radius-pill)',
          background: 'var(--accent-soft)',
          color: 'var(--accent)',
          fontSize: '0.64rem',
          letterSpacing: '0.04em',
          textTransform: 'uppercase',
        }}
      >
        <Icon name="check" size={13} color="var(--accent)" /> Matched · Comic Vine
      </span>
    );
  }
  return (
    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 10 }}>
      <span
        className="ch-mono"
        style={{
          display: 'inline-flex',
          alignItems: 'center',
          gap: 6,
          height: 24,
          padding: '0 10px',
          borderRadius: 'var(--radius-pill)',
          background: 'color-mix(in oklab, var(--warning) 20%, transparent)',
          color: 'var(--warning)',
          fontSize: '0.64rem',
          letterSpacing: '0.04em',
          textTransform: 'uppercase',
        }}
      >
        <Icon name="alert-triangle" size={13} color="var(--warning)" />{' '}
        {state === 'none' ? 'No match' : 'Incomplete'}
      </span>
      <Button size="sm" variant="secondary" icon="refresh" onClick={onMatch}>
        Match metadata
      </Button>
    </span>
  );
}

/** Empty state for the Volumes/Story Arcs tabs when the series isn't fully matched. */
function MatchToPopulate({
  kind,
  state,
  onMatch,
}: {
  kind: string;
  state?: MetadataState;
  onMatch: () => void;
}) {
  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        textAlign: 'center',
        padding: '40px 24px',
        border: '1px dashed var(--border-strong)',
        borderRadius: 'var(--radius-lg)',
      }}
    >
      <div
        className="ch-halftone-duo"
        style={{ width: 72, height: 72, borderRadius: '50%', marginBottom: 16, opacity: 0.5 }}
      />
      <div style={{ fontFamily: 'var(--font-display)', fontWeight: 800, fontSize: '1.05rem' }}>
        No {kind} yet
      </div>
      <p
        style={{
          margin: '8px 0 16px',
          maxWidth: 360,
          fontSize: '0.85rem',
          color: 'var(--text-secondary)',
          lineHeight: 1.5,
        }}
      >
        {state === 'none'
          ? 'This series has no metadata-provider match.'
          : 'This series is only partly matched.'}{' '}
        Match it to Comic Vine to pull in {kind}.
      </p>
      <Button variant="secondary" icon="refresh" onClick={onMatch}>
        Match metadata
      </Button>
    </div>
  );
}

/** A Volume card on the Volumes tab. */
function GroupingCardButton({
  cover,
  name,
  meta,
  description,
  onClick,
}: {
  cover?: string;
  name: string;
  meta: string;
  description?: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      style={{
        display: 'flex',
        gap: 14,
        padding: 12,
        textAlign: 'left',
        cursor: 'pointer',
        background: 'var(--surface-raised)',
        border: '1px solid var(--border-hairline)',
        borderRadius: 'var(--radius-lg)',
      }}
    >
      <div
        style={{
          width: 60,
          height: 90,
          flex: 'none',
          background: 'var(--surface-cover)',
          backgroundImage: cover ? `url(${cover})` : undefined,
          backgroundSize: 'cover',
          backgroundPosition: 'center',
          boxShadow: '0 2px 10px rgba(0,0,0,.5)',
        }}
      />
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ fontWeight: 600, fontSize: '0.92rem', color: 'var(--text-primary)' }}>
          {name}
        </div>
        <div
          className="ch-mono"
          style={{ fontSize: '0.66rem', color: 'var(--text-tertiary)', marginTop: 4 }}
        >
          {meta}
        </div>
        {description && (
          <p
            style={{
              margin: '8px 0 0',
              fontSize: '0.78rem',
              color: 'var(--text-secondary)',
              lineHeight: 1.4,
              display: '-webkit-box',
              WebkitLineClamp: 2,
              WebkitBoxOrient: 'vertical',
              overflow: 'hidden',
            }}
          >
            {description}
          </p>
        )}
      </div>
    </button>
  );
}

/** A Story Arc row on the Story Arcs tab. */
function ArcRow({ arc, onClick }: { arc: GroupingCard; onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 16,
        padding: '14px 16px',
        textAlign: 'left',
        cursor: 'pointer',
        background: 'var(--surface-raised)',
        border: '1px solid var(--border-hairline)',
        borderRadius: 'var(--radius-md)',
      }}
    >
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ fontWeight: 600, fontSize: '0.95rem', color: 'var(--text-primary)' }}>
          {arc.name}
        </div>
        {arc.description && (
          <p style={{ margin: '3px 0 0', fontSize: '0.78rem', color: 'var(--text-secondary)' }}>
            {arc.description}
          </p>
        )}
      </div>
      <span
        className="ch-mono"
        style={{ flex: 'none', fontSize: '0.68rem', color: 'var(--text-tertiary)' }}
      >
        {arc.issueCount} issues
      </span>
      <Icon name="chevron-right" size={16} color="var(--text-tertiary)" />
    </button>
  );
}

function Detail({ label, value }: { label: string; value: string }) {
  return (
    <>
      <div className="ch-label" style={{ color: 'var(--text-tertiary)', paddingTop: 2 }}>
        {label}
      </div>
      <div style={{ color: 'var(--text-primary)' }}>{value}</div>
    </>
  );
}

function Centered({ children }: { children: React.ReactNode }) {
  return <div style={{ padding: 'var(--pad-screen)' }}>{children}</div>;
}
