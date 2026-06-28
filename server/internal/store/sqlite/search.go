package sqlite

import (
	"context"
	"database/sql"

	"github.com/siposbnc/comic-hub/server/internal/domain"
)

// searchRepo runs full-text queries against the FTS5 tables created in 0004_search.sql.
// The `match` argument is a prepared FTS5 MATCH expression (see service/browse).
type searchRepo struct{ db *sql.DB }

func (r *searchRepo) SearchSeries(ctx context.Context, libraryID, match string, limit int) ([]domain.SeriesSearchHit, error) {
	// Resolve a display cover the same way the grid does: the series' configured cover,
	// else its first issue by sort order.
	q := `
		SELECT s.id, s.name, s.year,
			COALESCE(s.cover_book_id,
				(SELECT b.id FROM book b WHERE b.series_id = s.id
					ORDER BY b.sort_number, b.number LIMIT 1)) AS cover_book_id
		FROM series_fts f
		JOIN series s ON s.id = f.series_id
		WHERE f.name MATCH ?`
	args := []any{match}
	if libraryID != "" {
		q += " AND s.library_id = ?"
		args = append(args, libraryID)
	}
	q += " ORDER BY rank LIMIT ?"
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.SeriesSearchHit
	for rows.Next() {
		var (
			h     domain.SeriesSearchHit
			year  sql.NullInt64
			cover sql.NullString
		)
		if err := rows.Scan(&h.ID, &h.Name, &year, &cover); err != nil {
			return nil, err
		}
		h.Year = int(i64(year))
		h.CoverBookID = str(cover)
		out = append(out, h)
	}
	return out, rows.Err()
}

func (r *searchRepo) SearchBooks(ctx context.Context, libraryID, match string, limit int) ([]domain.BookSearchHit, error) {
	q := `
		SELECT b.id, b.series_id, s.name, b.number, b.title, b.file_format
		FROM book_fts f
		JOIN book b ON b.id = f.book_id
		JOIN series s ON s.id = b.series_id
		WHERE f.title MATCH ?`
	args := []any{match}
	if libraryID != "" {
		q += " AND b.library_id = ?"
		args = append(args, libraryID)
	}
	q += " ORDER BY rank LIMIT ?"
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.BookSearchHit
	for rows.Next() {
		var (
			h      domain.BookSearchHit
			number sql.NullString
			title  sql.NullString
		)
		if err := rows.Scan(&h.ID, &h.SeriesID, &h.SeriesName, &number, &title, &h.Format); err != nil {
			return nil, err
		}
		h.Number = str(number)
		h.Title = str(title)
		out = append(out, h)
	}
	return out, rows.Err()
}
