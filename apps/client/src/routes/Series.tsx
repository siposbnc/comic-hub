import { useState } from 'react';
import { getRouteApi, useNavigate } from '@tanstack/react-router';
import { useMutation } from '@tanstack/react-query';
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
import { isSpecialKind } from '@comichub/api-client';
import type { BookCard, SeriesDetail, GroupingCard, MetadataState } from '@comichub/api-client';
import { useClient } from '../lib/client.js';
import { useSeriesDetail, useRescanSeries } from '../lib/queries.js';
import { useReadLaunch } from '../lib/launch.js';
import { useUiStore } from '../store/ui.js';
import { LoadingState, ErrorState } from '../components/Page.js';
import { MatchDialog } from '../components/MatchDialog.js';
import {
  ResolveDuplicatesDialog,
  type DuplicateGroup,
} from '../components/ResolveDuplicatesDialog.js';
import { ManageFilesDialog } from '../components/ManageFilesDialog.js';
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
  // Empty tabs (Specials/Volumes/Story Arcs) are hidden entirely; if the data shifts under
  // a selected tab (e.g. a re-match empties it), fall back to Issues instead of a blank body.
  const tabExists =
    tab === 'issues' ||
    tab === 'details' ||
    (tab === 'specials' && detail.books.some((b) => isSpecialKind(b.kind))) ||
    (tab === 'volumes' && (detail.volumes?.length ?? 0) > 0) ||
    (tab === 'arcs' && (detail.storyArcs?.length ?? 0) > 0);
  const activeTab = tabExists ? tab : 'issues';
  const [matching, setMatching] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);
  const [confirmingRescan, setConfirmingRescan] = useState(false);
  const [resolving, setResolving] = useState(false);
  const [managingFiles, setManagingFiles] = useState(false);

  // An interrupted match (e.g. a provider rate limit) leaves the series incomplete but
  // still linked to its provider volume; re-running the same match resumes it server-side,
  // fetching only the issues that aren't applied yet.
  const canContinueMatch = detail.metadataState === 'incomplete' && !!detail.matchProviderId;
  const continueMatch = useMutation({
    mutationFn: () =>
      client.applySeriesMatch(detail.id, detail.matchProviderId!, {
        provider: detail.matchProvider,
      }),
    onSuccess: () =>
      addToast({
        tone: 'info',
        title: 'Continuing match',
        message: `Fetching the remaining issues for ${detail.name}…`,
      }),
    onError: (e) =>
      addToast({
        tone: 'danger',
        title: 'Could not continue the match',
        message: e instanceof Error ? e.message : 'Unexpected error.',
      }),
  });

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

  // Specials (annuals, one-shots, event tie-ins, …) live on their own tab, out of the
  // numbered run. Resume still prefers the run: a half-read annual shouldn't hijack the CTA
  // over the next unread issue, but it wins over nothing.
  const issues = detail.books.filter((b) => !isSpecialKind(b.kind));
  const specials = detail.books.filter((b) => isSpecialKind(b.kind));

  // Regular issues sharing a number are almost always a mis-parsed special/tie-in (noisy
  // folder names defeat the filename heuristics) — surface them with a resolve flow.
  const duplicateGroups = findDuplicateNumbers(issues);

  const resumeBook =
    issues.find((b) => b.progress?.status === 'in_progress') ??
    specials.find((b) => b.progress?.status === 'in_progress') ??
    issues.find((b) => b.progress?.status !== 'read') ??
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
              <MetaChip
                state={detail.metadataState}
                onMatch={() => setMatching(true)}
                onContinue={
                  canContinueMatch && !continueMatch.isPending
                    ? () => continueMatch.mutate()
                    : undefined
                }
              />
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
                      <MenuItem
                        icon="list"
                        label="Manage files"
                        onClick={() => {
                          setMenuOpen(false);
                          setManagingFiles(true);
                        }}
                      />
                      <MenuItem
                        icon="refresh"
                        label="Rescan series"
                        onClick={() => {
                          setMenuOpen(false);
                          setConfirmingRescan(true);
                        }}
                      />
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
        {duplicateGroups.length > 0 && (
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 12,
              padding: '12px 16px',
              marginBottom: 20,
              background: 'color-mix(in oklab, var(--warning) 12%, transparent)',
              border: '1px solid color-mix(in oklab, var(--warning) 35%, transparent)',
              borderRadius: 'var(--radius-md)',
            }}
          >
            <Icon name="alert-triangle" size={18} color="var(--warning)" />
            <div style={{ flex: 1, minWidth: 0 }}>
              <div style={{ fontWeight: 600, fontSize: '0.9rem', color: 'var(--text-primary)' }}>
                Duplicate issue numbers
              </div>
              <div style={{ fontSize: '0.8rem', color: 'var(--text-secondary)', marginTop: 2 }}>
                {duplicateGroups.length === 1
                  ? `${issueLabel(duplicateGroups[0]!.number)} appears on ${duplicateGroups[0]!.books.length} files`
                  : `${duplicateGroups.length} issue numbers appear on more than one file`}{' '}
                — usually a special or tie-in parsed as a regular issue.
              </div>
            </div>
            <Button size="sm" variant="secondary" onClick={() => setResolving(true)}>
              Resolve
            </Button>
          </div>
        )}
        <Tabs
          value={activeTab}
          onChange={setTab}
          tabs={[
            { value: 'issues', label: 'Issues', count: issues.length },
            ...(specials.length > 0
              ? [{ value: 'specials', label: 'Specials', count: specials.length }]
              : []),
            ...(detail.volumes?.length
              ? [{ value: 'volumes', label: 'Volumes', count: detail.volumes.length }]
              : []),
            ...(detail.storyArcs?.length
              ? [{ value: 'arcs', label: 'Story Arcs', count: detail.storyArcs.length }]
              : []),
            { value: 'details', label: 'Details' },
          ]}
          style={{ marginBottom: 22 }}
        />

        {activeTab === 'issues' ? (
          issues.length === 0 ? (
            <EmptyState title="No issues yet">This series has no scanned issues.</EmptyState>
          ) : (
            <IssueGrid books={issues} seriesName={detail.name} />
          )
        ) : activeTab === 'specials' ? (
          <IssueGrid books={specials} seriesName={detail.name} />
        ) : activeTab === 'volumes' ? (
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: 'repeat(auto-fill, minmax(240px, 1fr))',
              gap: 18,
            }}
          >
            {(detail.volumes ?? []).map((v) => (
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
        ) : activeTab === 'arcs' ? (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            {(detail.storyArcs ?? []).map((a) => (
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
      {resolving && (
        <ResolveDuplicatesDialog
          seriesId={detail.id}
          groups={duplicateGroups}
          onClose={() => setResolving(false)}
        />
      )}
      {managingFiles && (
        <ManageFilesDialog
          seriesId={detail.id}
          seriesName={detail.name}
          onClose={() => setManagingFiles(false)}
        />
      )}
    </div>
  );
}

/** A row in the series overflow menu (icon + label), matching the popover's hover styling. */
function MenuItem({
  icon,
  label,
  onClick,
}: {
  icon: 'list' | 'refresh';
  label: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      role="menuitem"
      onClick={onClick}
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
      <Icon name={icon} size={16} color="var(--paper-400)" /> {label}
    </button>
  );
}

/** Groups regular issues that share a normalized number ("001" == "1"); a group with more
 *  than one book is a duplicate worth resolving. */
function findDuplicateNumbers(issues: BookCard[]): DuplicateGroup[] {
  const byNumber = new Map<string, BookCard[]>();
  for (const b of issues) {
    const raw = b.number?.trim();
    if (!raw) continue;
    const n = Number(raw);
    const key = Number.isFinite(n) ? String(n) : raw.toLowerCase();
    const group = byNumber.get(key);
    if (group) group.push(b);
    else byNumber.set(key, [b]);
  }
  return [...byNumber.entries()]
    .filter(([, books]) => books.length > 1)
    .map(([key, books]) => ({ number: books[0]!.number ?? key, books }));
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

/** The cover grid shared by the Issues and Specials tabs. */
function IssueGrid({ books, seriesName }: { books: BookCard[]; seriesName: string }) {
  return (
    <div
      style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fill, minmax(132px, 1fr))',
        gap: 16,
      }}
    >
      {books.map((book) => (
        <IssueCover key={book.id} book={book} seriesName={seriesName} />
      ))}
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

/** Series metadata-state chip: matched (accent) or incomplete/no-match (warning + Match).
 *  When an interrupted match can be resumed, a Continue matching action leads. */
function MetaChip({
  state,
  onMatch,
  onContinue,
}: {
  state?: MetadataState;
  onMatch: () => void;
  onContinue?: () => void;
}) {
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
      {onContinue && (
        <Button size="sm" variant="secondary" icon="refresh" onClick={onContinue}>
          Continue matching
        </Button>
      )}
      <Button
        size="sm"
        variant={onContinue ? 'ghost' : 'secondary'}
        icon="search"
        onClick={onMatch}
      >
        Match metadata
      </Button>
    </span>
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
