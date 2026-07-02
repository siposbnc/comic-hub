package sqlstore

import (
	"context"
	"database/sql"

	"github.com/siposbnc/comic-hub/server/internal/domain"
)

// statsRepo serves per-user reading aggregates (Milestone G). All SQL here is
// dialect-portable; calendar math happens in the stats service.
type statsRepo struct{ db *DB }

func (r *statsRepo) ReadCounts(ctx context.Context, userID string) (int, int, error) {
	var books, pages sql.NullInt64
	err := r.db.QueryRowContext(ctx, `
		SELECT
			SUM(CASE WHEN status = 'read' THEN 1 ELSE 0 END),
			SUM(CASE WHEN status = 'read' THEN page_count ELSE page END)
		FROM read_progress WHERE user_id = ?`, userID).Scan(&books, &pages)
	if err != nil {
		return 0, 0, err
	}
	return int(i64(books)), int(i64(pages)), nil
}

func (r *statsRepo) FinishedTimes(ctx context.Context, userID string, since int64) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT finished_at FROM read_progress
		WHERE user_id = ? AND status = 'read' AND finished_at IS NOT NULL AND finished_at >= ?`,
		userID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanInt64s(rows)
}

func (r *statsRepo) ActivityTimes(ctx context.Context, userID string) ([]int64, error) {
	// Every day-relevant timestamp the progress rows carry. UNION ALL keeps it one
	// round-trip; NULLs are filtered per branch.
	rows, err := r.db.QueryContext(ctx, `
		SELECT updated_at FROM read_progress WHERE user_id = ?
		UNION ALL
		SELECT started_at FROM read_progress WHERE user_id = ? AND started_at IS NOT NULL
		UNION ALL
		SELECT finished_at FROM read_progress WHERE user_id = ? AND finished_at IS NOT NULL`,
		userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanInt64s(rows)
}

func (r *statsRepo) TopGenres(ctx context.Context, userID string, limit int) ([]domain.NameCount, error) {
	return r.nameCounts(ctx, `
		SELECT g.name, COUNT(*) AS n
		FROM read_progress rp
		JOIN book_genre bg ON bg.book_id = rp.book_id
		JOIN genre g ON g.id = bg.genre_id
		WHERE rp.user_id = ? AND rp.status = 'read'
		GROUP BY g.name ORDER BY n DESC, g.name LIMIT ?`, userID, limit)
}

func (r *statsRepo) TopPublishers(ctx context.Context, userID string, limit int) ([]domain.NameCount, error) {
	return r.nameCounts(ctx, `
		SELECT s.publisher, COUNT(*) AS n
		FROM read_progress rp
		JOIN book b ON b.id = rp.book_id
		JOIN series s ON s.id = b.series_id
		WHERE rp.user_id = ? AND rp.status = 'read' AND s.publisher IS NOT NULL AND s.publisher <> ''
		GROUP BY s.publisher ORDER BY n DESC, s.publisher LIMIT ?`, userID, limit)
}

func (r *statsRepo) RecentlyFinished(ctx context.Context, userID string, limit int) ([]domain.FinishedBook, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT b.id, b.title, b.number, s.id, s.name, rp.finished_at
		FROM read_progress rp
		JOIN book b ON b.id = rp.book_id
		JOIN series s ON s.id = b.series_id
		WHERE rp.user_id = ? AND rp.status = 'read' AND rp.finished_at IS NOT NULL
		ORDER BY rp.finished_at DESC LIMIT ?`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.FinishedBook
	for rows.Next() {
		var (
			f             domain.FinishedBook
			title, number sql.NullString
		)
		if err := rows.Scan(&f.BookID, &title, &number, &f.SeriesID, &f.SeriesName, &f.FinishedAt); err != nil {
			return nil, err
		}
		f.Title = str(title)
		f.Number = str(number)
		out = append(out, f)
	}
	return out, rows.Err()
}

func (r *statsRepo) nameCounts(ctx context.Context, query string, args ...any) ([]domain.NameCount, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.NameCount
	for rows.Next() {
		var nc domain.NameCount
		if err := rows.Scan(&nc.Name, &nc.Count); err != nil {
			return nil, err
		}
		out = append(out, nc)
	}
	return out, rows.Err()
}

func scanInt64s(rows *sql.Rows) ([]int64, error) {
	var out []int64
	for rows.Next() {
		var v sql.NullInt64
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		if v.Valid {
			out = append(out, v.Int64)
		}
	}
	return out, rows.Err()
}
