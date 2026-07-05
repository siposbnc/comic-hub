package sqlstore

import (
	"context"
	"database/sql"
	"strings"

	"github.com/siposbnc/comic-hub/server/internal/domain"
)

// searchRepo runs full-text queries. The `match` argument is the FTS5 MATCH expression
// prepared by service/browse — sanitized prefix tokens like `bat* man*`. On SQLite it
// hits the FTS5 tables from 0004_search.sql; on Postgres it is translated to a tsquery
// (`bat:* & man:*`) against the GIN expression indexes from the same-numbered migration.
type searchRepo struct{ db *DB }

// pgQuery converts the sanitized FTS5 match expression into a tsquery string. Tokens
// are alphanumeric runs with a trailing `*` (see browse.buildMatch), so this is pure
// syntax translation, never user input.
func pgQuery(match string) string {
	tokens := strings.Fields(match)
	for i, t := range tokens {
		tokens[i] = strings.TrimSuffix(t, "*") + ":*"
	}
	return strings.Join(tokens, " & ")
}

func (r *searchRepo) SearchSeries(ctx context.Context, libraryID, match string, limit int) ([]domain.SeriesSearchHit, error) {
	// Resolve a display cover the same way the grid does: the series' configured cover,
	// else its first issue by sort order.
	var q string
	var args []any
	if r.db.Driver() == DriverPostgres {
		q = `
		SELECT s.id, s.name, s.year,
			COALESCE(s.cover_book_id,
				(SELECT b.id FROM book b WHERE b.series_id = s.id
					ORDER BY b.sort_number, b.number LIMIT 1)) AS cover_book_id,
			l.name AS library_name
		FROM series s
		JOIN library l ON l.id = s.library_id
		WHERE to_tsvector('simple', s.name) @@ to_tsquery('simple', ?)`
		args = []any{pgQuery(match)}
		if libraryID != "" {
			q += " AND s.library_id = ?"
			args = append(args, libraryID)
		}
		q += ` ORDER BY ts_rank(to_tsvector('simple', s.name), to_tsquery('simple', ?)) DESC LIMIT ?`
		args = append(args, pgQuery(match), limit)
	} else {
		q = `
		SELECT s.id, s.name, s.year,
			COALESCE(s.cover_book_id,
				(SELECT b.id FROM book b WHERE b.series_id = s.id
					ORDER BY b.sort_number, b.number LIMIT 1)) AS cover_book_id,
			l.name AS library_name
		FROM series_fts f
		JOIN series s ON s.id = f.series_id
		JOIN library l ON l.id = s.library_id
		WHERE f.name MATCH ?`
		args = []any{match}
		if libraryID != "" {
			q += " AND s.library_id = ?"
			args = append(args, libraryID)
		}
		q += " ORDER BY rank LIMIT ?"
		args = append(args, limit)
	}

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
		if err := rows.Scan(&h.ID, &h.Name, &year, &cover, &h.LibraryName); err != nil {
			return nil, err
		}
		h.Year = int(i64(year))
		h.CoverBookID = str(cover)
		out = append(out, h)
	}
	return out, rows.Err()
}

func (r *searchRepo) SearchBooks(ctx context.Context, libraryID, match string, limit int) ([]domain.BookSearchHit, error) {
	var q string
	var args []any
	if r.db.Driver() == DriverPostgres {
		q = `
		SELECT b.id, b.series_id, s.name, b.number, b.title, b.file_format, l.name
		FROM book b
		JOIN series s ON s.id = b.series_id
		JOIN library l ON l.id = b.library_id
		WHERE to_tsvector('simple', coalesce(b.title, '')) @@ to_tsquery('simple', ?)`
		args = []any{pgQuery(match)}
		if libraryID != "" {
			q += " AND b.library_id = ?"
			args = append(args, libraryID)
		}
		q += ` ORDER BY ts_rank(to_tsvector('simple', coalesce(b.title, '')), to_tsquery('simple', ?)) DESC LIMIT ?`
		args = append(args, pgQuery(match), limit)
	} else {
		q = `
		SELECT b.id, b.series_id, s.name, b.number, b.title, b.file_format, l.name
		FROM book_fts f
		JOIN book b ON b.id = f.book_id
		JOIN series s ON s.id = b.series_id
		JOIN library l ON l.id = b.library_id
		WHERE f.title MATCH ?`
		args = []any{match}
		if libraryID != "" {
			q += " AND b.library_id = ?"
			args = append(args, libraryID)
		}
		q += " ORDER BY rank LIMIT ?"
		args = append(args, limit)
	}

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
		if err := rows.Scan(&h.ID, &h.SeriesID, &h.SeriesName, &number, &title, &h.Format, &h.LibraryName); err != nil {
			return nil, err
		}
		h.Number = str(number)
		h.Title = str(title)
		out = append(out, h)
	}
	return out, rows.Err()
}
