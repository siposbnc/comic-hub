package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/siposbnc/comic-hub/server/internal/domain"
)

// seriesRepo persists series. Upsert is keyed on the ULID id; the scanner resolves a
// folder to an existing series id before calling Upsert (lookup-by-folder lands with
// the scanner in a later milestone).
type seriesRepo struct{ db *sql.DB }

const seriesColumns = `id, library_id, folder_path, name, sort_name, year, publisher,
	description, reading_dir, cover_book_id, created_at, updated_at`

func (r *seriesRepo) Upsert(ctx context.Context, s domain.Series) (domain.Series, error) {
	rd := s.ReadingDir
	if rd == "" {
		rd = domain.LTR
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO series (`+seriesColumns+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			folder_path   = excluded.folder_path,
			name          = excluded.name,
			sort_name     = excluded.sort_name,
			year          = excluded.year,
			publisher     = excluded.publisher,
			description   = excluded.description,
			reading_dir   = excluded.reading_dir,
			cover_book_id = excluded.cover_book_id,
			updated_at    = excluded.updated_at`,
		s.ID, s.LibraryID, nullString(s.FolderPath), s.Name, s.SortName, nullInt(int64(s.Year)),
		nullString(s.Publisher), nullString(s.Description), string(rd), nullString(s.CoverBookID),
		s.CreatedAt, s.UpdatedAt,
	)
	if err != nil {
		return domain.Series{}, err
	}
	s.ReadingDir = rd
	return s, nil
}

func (r *seriesRepo) Get(ctx context.Context, id string) (domain.Series, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+seriesColumns+` FROM series WHERE id = ?`, id)
	s, err := scanSeries(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Series{}, domain.ErrNotFound
	}
	return s, err
}

func (r *seriesRepo) GetByFolder(ctx context.Context, libraryID, folderPath string) (domain.Series, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+seriesColumns+` FROM series WHERE library_id = ? AND folder_path = ?`,
		libraryID, folderPath)
	s, err := scanSeries(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Series{}, domain.ErrNotFound
	}
	return s, err
}

func (r *seriesRepo) ListByLibrary(ctx context.Context, libraryID string) ([]domain.Series, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+seriesColumns+` FROM series WHERE library_id = ? ORDER BY sort_name`, libraryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Series
	for rows.Next() {
		s, err := scanSeries(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// Summaries returns each series in a library with its book count, the user's read
// count, and the cover book (the series' configured cover, else its first issue) — all
// in one query so the grid doesn't N+1.
func (r *seriesRepo) Summaries(ctx context.Context, libraryID, userID string) ([]domain.SeriesSummary, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+prefixedSeriesColumns("s")+`,
			(SELECT COUNT(*) FROM book b WHERE b.series_id = s.id) AS book_count,
			(SELECT COUNT(*) FROM book b
				JOIN read_progress p ON p.book_id = b.id AND p.user_id = ?
				WHERE b.series_id = s.id AND p.status = 'read') AS read_count,
			COALESCE(s.cover_book_id,
				(SELECT b.id FROM book b WHERE b.series_id = s.id
					ORDER BY b.sort_number, b.number LIMIT 1)) AS cover_book_id
		FROM series s
		WHERE s.library_id = ?
		ORDER BY s.sort_name`, userID, libraryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.SeriesSummary
	for rows.Next() {
		sum, err := scanSeriesSummary(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sum)
	}
	return out, rows.Err()
}

// prefixedSeriesColumns returns seriesColumns with a table alias prefix.
func prefixedSeriesColumns(alias string) string {
	cols := []string{"id", "library_id", "folder_path", "name", "sort_name", "year",
		"publisher", "description", "reading_dir", "cover_book_id", "created_at", "updated_at"}
	for i, c := range cols {
		cols[i] = alias + "." + c
	}
	return strings.Join(cols, ", ")
}

func scanSeriesSummary(rows *sql.Rows) (domain.SeriesSummary, error) {
	var (
		s        domain.Series
		folder   sql.NullString
		year     sql.NullInt64
		pub      sql.NullString
		desc     sql.NullString
		readDir  string
		coverBID sql.NullString
		bookCnt  int
		readCnt  int
		cover    sql.NullString
	)
	if err := rows.Scan(
		&s.ID, &s.LibraryID, &folder, &s.Name, &s.SortName, &year, &pub,
		&desc, &readDir, &coverBID, &s.CreatedAt, &s.UpdatedAt,
		&bookCnt, &readCnt, &cover,
	); err != nil {
		return domain.SeriesSummary{}, err
	}
	s.FolderPath = str(folder)
	s.Year = int(i64(year))
	s.Publisher = str(pub)
	s.Description = str(desc)
	s.ReadingDir = domain.ReadingDirection(readDir)
	s.CoverBookID = str(coverBID)
	return domain.SeriesSummary{Series: s, BookCount: bookCnt, ReadCount: readCnt, CoverBookID: str(cover)}, nil
}

// rowScanner is satisfied by both *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanSeries(row rowScanner) (domain.Series, error) {
	var (
		s        domain.Series
		folder   sql.NullString
		year     sql.NullInt64
		pub      sql.NullString
		desc     sql.NullString
		readDir  string
		coverBID sql.NullString
	)
	if err := row.Scan(
		&s.ID, &s.LibraryID, &folder, &s.Name, &s.SortName, &year, &pub,
		&desc, &readDir, &coverBID, &s.CreatedAt, &s.UpdatedAt,
	); err != nil {
		return domain.Series{}, err
	}
	s.FolderPath = str(folder)
	s.Year = int(i64(year))
	s.Publisher = str(pub)
	s.Description = str(desc)
	s.ReadingDir = domain.ReadingDirection(readDir)
	s.CoverBookID = str(coverBID)
	return s, nil
}
