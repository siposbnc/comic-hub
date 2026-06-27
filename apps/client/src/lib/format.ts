import type { BookCard, ProgressSummary, ReadStatus } from '@comichub/api-client';

/** CoverCard's status enum (it calls the mid state "reading", the wire calls it "in_progress"). */
export function toCoverStatus(status?: ReadStatus): 'unread' | 'reading' | 'read' {
  switch (status) {
    case 'read':
      return 'read';
    case 'in_progress':
      return 'reading';
    default:
      return 'unread';
  }
}

/** 0..1 progress fraction for a book card. */
export function progressFraction(progress?: ProgressSummary): number {
  if (!progress) return 0;
  return Math.max(0, Math.min(1, progress.percent / 100));
}

/** Issue number in the print voice: "#012" when numeric, otherwise the raw label. */
export function issueLabel(number?: string): string | undefined {
  if (number == null || number === '') return undefined;
  const n = Number(number);
  if (Number.isFinite(n)) return `#${String(n).padStart(3, '0')}`;
  return `#${number}`;
}

/** A human title for a book card (issue title, else "Series #NNN", else just the number). */
export function bookCardTitle(book: BookCard, seriesName?: string): string {
  if (book.title) return book.title;
  const issue = issueLabel(book.number);
  if (seriesName && issue) return `${seriesName} ${issue}`;
  return issue ?? 'Untitled';
}

/** The page a "Continue"/"Read" CTA should open at. */
export function resumePage(progress?: ProgressSummary): number {
  if (!progress || progress.status === 'read') return 0;
  return Math.max(0, progress.page);
}
