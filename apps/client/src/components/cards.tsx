import { useNavigate } from '@tanstack/react-router';
import { CoverCard } from '@comichub/ui';
import type { BookCard, SeriesCard } from '@comichub/api-client';
import { useClient } from '../lib/client.js';
import { issueLabel, progressFraction, toCoverStatus } from '../lib/format.js';

const COVER_PX = 168; // --cover-w-m
export const COVER_CELL_HEIGHT = COVER_PX * 1.5 + 7 + 38; // cover + gap + 2-line label

/** A book rendered as a CoverCard that routes to its detail page. */
export function BookCover({ book, seriesName }: { book: BookCard; seriesName?: string }) {
  const client = useClient();
  const navigate = useNavigate();
  const title = book.title || seriesName || issueLabel(book.number) || 'Untitled';
  const subtitle = issueLabel(book.number) ?? `${book.pageCount} pages`;
  return (
    <CoverCard
      cover={client.coverUrl(book.id, 300)}
      title={title}
      subtitle={subtitle}
      number={issueLabel(book.number)}
      status={toCoverStatus(book.progress?.status)}
      progress={progressFraction(book.progress)}
      onClick={() => navigate({ to: '/book/$id', params: { id: book.id } })}
    />
  );
}

/** A series rendered as a CoverCard (its cover book) that routes to the series page. */
export function SeriesCover({ series }: { series: SeriesCard }) {
  const client = useClient();
  const navigate = useNavigate();
  const allRead = series.bookCount > 0 && series.readCount >= series.bookCount;
  const someRead = series.readCount > 0 && !allRead;
  return (
    <CoverCard
      cover={series.coverBookId ? client.coverUrl(series.coverBookId, 300) : undefined}
      title={series.name}
      subtitle={`${series.bookCount} ${series.bookCount === 1 ? 'issue' : 'issues'}`}
      status={allRead ? 'read' : someRead ? 'reading' : 'unread'}
      progress={series.bookCount ? series.readCount / series.bookCount : 0}
      onClick={() => navigate({ to: '/series/$id', params: { id: series.id } })}
    />
  );
}
