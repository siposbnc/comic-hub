package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/siposbnc/comic-hub/server/internal/domain"
)

// bookRepo persists books and their pages. Upsert is keyed on the ULID id; the
// scanner resolves a file path to an existing book id before persisting.
type bookRepo struct{ db *DB }

const bookColumns = `id, series_id, library_id, file_path, file_format, file_size,
	file_mtime, content_hash, page_count, title, number, sort_number, volume,
	release_date, age_rating, language, summary, cover_page, metadata_state,
	is_corrupt, added_at, updated_at, kind`

func (r *bookRepo) Upsert(ctx context.Context, b domain.Book) (domain.Book, error) {
	state := b.MetadataState
	if state == "" {
		state = domain.MetaNone
	}
	kind := b.Kind
	if kind == "" {
		kind = domain.KindIssue
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO book (`+bookColumns+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			series_id      = excluded.series_id,
			file_path      = excluded.file_path,
			file_format    = excluded.file_format,
			file_size      = excluded.file_size,
			file_mtime     = excluded.file_mtime,
			content_hash   = excluded.content_hash,
			page_count     = excluded.page_count,
			title          = excluded.title,
			number         = excluded.number,
			sort_number    = excluded.sort_number,
			volume         = excluded.volume,
			release_date   = excluded.release_date,
			age_rating     = excluded.age_rating,
			language       = excluded.language,
			summary        = excluded.summary,
			cover_page     = excluded.cover_page,
			metadata_state = excluded.metadata_state,
			is_corrupt     = excluded.is_corrupt,
			updated_at     = excluded.updated_at,
			kind           = excluded.kind`,
		b.ID, b.SeriesID, b.LibraryID, b.FilePath, b.FileFormat, b.FileSize,
		b.FileMTime, nullString(b.ContentHash), b.PageCount, nullString(b.Title),
		nullString(b.Number), nullFloat(b.SortNumber), nullInt(int64(b.Volume)),
		nullInt(b.ReleaseDate), nullString(b.AgeRating), nullString(b.Language),
		nullString(b.Summary), b.CoverPage, string(state), boolToInt(b.IsCorrupt),
		b.AddedAt, b.UpdatedAt, string(kind),
	)
	if err != nil {
		return domain.Book{}, err
	}
	b.MetadataState = state
	b.Kind = kind
	return b, nil
}

func (r *bookRepo) Get(ctx context.Context, id string) (domain.Book, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+bookColumns+` FROM book WHERE id = ?`, id)
	b, err := scanBook(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Book{}, domain.ErrNotFound
	}
	return b, err
}

func (r *bookRepo) GetByPath(ctx context.Context, filePath string) (domain.Book, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+bookColumns+` FROM book WHERE file_path = ?`, filePath)
	b, err := scanBook(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Book{}, domain.ErrNotFound
	}
	return b, err
}

func (r *bookRepo) ListBySeries(ctx context.Context, seriesID string) ([]domain.Book, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+bookColumns+` FROM book WHERE series_id = ? ORDER BY sort_number, number`, seriesID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Book
	for rows.Next() {
		b, err := scanBook(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *bookRepo) ByContentHash(ctx context.Context, libraryID, hash string) ([]domain.Book, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+bookColumns+` FROM book WHERE library_id = ? AND content_hash = ?`, libraryID, hash)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Book
	for rows.Next() {
		b, err := scanBook(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *bookRepo) ListByLibrary(ctx context.Context, libraryID string) ([]domain.Book, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+bookColumns+` FROM book WHERE library_id = ? ORDER BY added_at DESC`, libraryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Book
	for rows.Next() {
		b, err := scanBook(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// ReplacePages atomically swaps a book's page rows for the supplied set.
func (r *bookRepo) ReplacePages(ctx context.Context, bookID string, pages []domain.Page) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM page WHERE book_id = ?`, bookID); err != nil {
		return fmt.Errorf("clear pages: %w", err)
	}
	for _, p := range pages {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO page (book_id, idx, file_name, width, height, size, page_type, is_double)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			bookID, p.Index, p.FileName, nullInt(int64(p.Width)), nullInt(int64(p.Height)),
			nullInt(p.Size), nullString(p.PageType), boolToInt(p.IsDouble),
		); err != nil {
			return fmt.Errorf("insert page %d: %w", p.Index, err)
		}
	}
	return tx.Commit()
}

// SetPageDimensions updates width/height for specific pages without touching the rest of
// each row (unlike ReplacePages, which swaps the whole set). Runs in one transaction.
func (r *bookRepo) SetPageDimensions(ctx context.Context, bookID string, dims []domain.PageDimension) error {
	if len(dims) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for _, d := range dims {
		if _, err := tx.ExecContext(ctx,
			`UPDATE page SET width = ?, height = ? WHERE book_id = ? AND idx = ?`,
			nullInt(int64(d.Width)), nullInt(int64(d.Height)), bookID, d.Index,
		); err != nil {
			return fmt.Errorf("update page %d dimensions: %w", d.Index, err)
		}
	}
	return tx.Commit()
}

func (r *bookRepo) ListPages(ctx context.Context, bookID string) ([]domain.Page, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT book_id, idx, file_name, width, height, size, page_type, is_double
		FROM page WHERE book_id = ? ORDER BY idx`, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Page
	for rows.Next() {
		var (
			p        domain.Page
			width    sql.NullInt64
			height   sql.NullInt64
			size     sql.NullInt64
			pageType sql.NullString
			double   int
		)
		if err := rows.Scan(&p.BookID, &p.Index, &p.FileName, &width, &height, &size, &pageType, &double); err != nil {
			return nil, err
		}
		p.Width = int(i64(width))
		p.Height = int(i64(height))
		p.Size = i64(size)
		p.PageType = str(pageType)
		p.IsDouble = double != 0
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *bookRepo) ListRecent(ctx context.Context, libraryID string, limit int) ([]domain.Book, error) {
	query := `SELECT ` + bookColumns + ` FROM book`
	args := []any{}
	if libraryID != "" {
		query += ` WHERE library_id = ?`
		args = append(args, libraryID)
	}
	query += ` ORDER BY added_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Book
	for rows.Next() {
		b, err := scanBook(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func scanBook(row rowScanner) (domain.Book, error) {
	var (
		b        domain.Book
		hash     sql.NullString
		title    sql.NullString
		number   sql.NullString
		sortNum  sql.NullFloat64
		volume   sql.NullInt64
		release  sql.NullInt64
		ageRate  sql.NullString
		language sql.NullString
		summary  sql.NullString
		corrupt  int
		kind     string
	)
	if err := row.Scan(
		&b.ID, &b.SeriesID, &b.LibraryID, &b.FilePath, &b.FileFormat, &b.FileSize,
		&b.FileMTime, &hash, &b.PageCount, &title, &number, &sortNum, &volume,
		&release, &ageRate, &language, &summary, &b.CoverPage, &b.MetadataState,
		&corrupt, &b.AddedAt, &b.UpdatedAt, &kind,
	); err != nil {
		return domain.Book{}, err
	}
	b.ContentHash = str(hash)
	b.Title = str(title)
	b.Number = str(number)
	b.SortNumber = f64(sortNum)
	b.Volume = int(i64(volume))
	b.ReleaseDate = i64(release)
	b.AgeRating = str(ageRate)
	b.Language = str(language)
	b.Summary = str(summary)
	b.IsCorrupt = corrupt != 0
	b.Kind = domain.BookKind(kind)
	return b, nil
}
