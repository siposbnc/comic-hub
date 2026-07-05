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

/** Issue number in the print voice: "#012" when numeric ("#023.1" for point issues),
 * otherwise the raw label. */
export function issueLabel(number?: string): string | undefined {
  if (number == null || number === '') return undefined;
  const raw = number.startsWith('#') ? number.slice(1) : number; // tolerate hand-typed "#001"
  if (raw === '') return undefined;
  const n = Number(raw);
  if (Number.isFinite(n)) {
    const [int = '', frac] = String(n).split('.');
    return `#${int.padStart(3, '0')}${frac ? `.${frac}` : ''}`;
  }
  return `#${raw}`;
}

/** A human title for a book card (issue title, else "Series #NNN", else just the number). */
export function bookCardTitle(book: BookCard, seriesName?: string): string {
  if (book.title) return book.title;
  const issue = issueLabel(book.number);
  if (seriesName && issue) return `${seriesName} ${issue}`;
  return issue ?? 'Untitled';
}

/** Coarse relative timestamp for quiet sub-lines ("just now", "3 days ago"). */
export function relativeTime(ms: number): string {
  const diff = Date.now() - ms;
  const min = Math.floor(diff / 60_000);
  if (min < 1) return 'just now';
  if (min < 60) return `${min}m ago`;
  const h = Math.floor(min / 60);
  if (h < 24) return `${h}h ago`;
  const d = Math.floor(h / 24);
  if (d < 7) return d === 1 ? 'yesterday' : `${d} days ago`;
  const w = Math.floor(d / 7);
  if (w < 5) return w === 1 ? 'last week' : `${w} weeks ago`;
  const mo = Math.floor(d / 30);
  if (mo < 12) return mo <= 1 ? 'a month ago' : `${mo} months ago`;
  const y = Math.floor(d / 365);
  return y <= 1 ? 'a year ago' : `${y} years ago`;
}

/** The page a "Continue"/"Read" CTA should open at. */
export function resumePage(progress?: ProgressSummary): number {
  if (!progress || progress.status === 'read') return 0;
  return Math.max(0, progress.page);
}
