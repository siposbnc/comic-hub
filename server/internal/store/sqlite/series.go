package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/siposbnc/comic-hub/server/internal/domain"
)

// seriesRepo persists series. Upsert conflicts on the natural key (library_id,
// folder_path) so concurrent scans converge on one row per folder rather than creating
// duplicates; it returns the surviving row's id.
type seriesRepo struct{ db *sql.DB }

// seriesColumns are the scanner-owned columns Upsert writes. The match columns
// (metadata_state, match_provider, match_provider_id) are deliberately excluded so a
// rescan never clobbers a series' match; they're written via WriteMatch/SetMetadataState.
const seriesColumns = `id, library_id, folder_path, name, sort_name, year, publisher,
	description, reading_dir, cover_book_id, created_at, updated_at`

// seriesReadColumns is seriesColumns plus the match columns, for reads.
const seriesReadColumns = seriesColumns + `, metadata_state, match_provider, match_provider_id`

func (r *seriesRepo) Upsert(ctx context.Context, s domain.Series) (domain.Series, error) {
	rd := s.ReadingDir
	if rd == "" {
		rd = domain.LTR
	}
	// Conflict on the ULID id for the normal path: a rescan re-upserts the same series in
	// place (updating its name/folder/etc.).
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
		// The UNIQUE(library_id, folder_path) index rejected the insert: a *different* id
		// already owns this folder, i.e. a concurrent scan created the series first. Converge
		// on that existing row instead of duplicating (book is likewise protected by
		// UNIQUE(file_path)). If no such row exists, the error is something else — surface it.
		if s.FolderPath != "" {
			var existingID string
			if e2 := r.db.QueryRowContext(ctx,
				`SELECT id FROM series WHERE library_id = ? AND folder_path = ?`,
				s.LibraryID, s.FolderPath).Scan(&existingID); e2 == nil && existingID != "" {
				s.ID = existingID
				s.ReadingDir = rd
				return s, nil
			}
		}
		return domain.Series{}, err
	}
	s.ReadingDir = rd
	return s, nil
}

// WriteMatch records a series' online-match result without touching scanner-owned columns
// (name, folder, reading_dir). Year/publisher/description are only overwritten when the
// match supplies a value, so a partial match never blanks existing data.
func (r *seriesRepo) WriteMatch(ctx context.Context, id string, m domain.SeriesMatch) error {
	state := m.State
	if state == "" {
		state = domain.MetaMatched
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE series SET
			year              = CASE WHEN ? > 0 THEN ? ELSE year END,
			publisher         = CASE WHEN ? <> '' THEN ? ELSE publisher END,
			description       = CASE WHEN ? <> '' THEN ? ELSE description END,
			metadata_state    = ?,
			match_provider    = ?,
			match_provider_id = ?,
			updated_at        = ?
		WHERE id = ?`,
		m.Year, m.Year,
		m.Publisher, m.Publisher,
		m.Description, m.Description,
		string(state), m.Provider, m.ProviderID,
		time.Now().UnixMilli(), id,
	)
	return err
}

// SetMetadataState updates only a series' metadata state.
func (r *seriesRepo) SetMetadataState(ctx context.Context, id string, state domain.MetadataState) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE series SET metadata_state = ?, updated_at = ? WHERE id = ?`,
		string(state), time.Now().UnixMilli(), id)
	return err
}

// DeleteEmpty removes a library's series with no books, returning the count deleted.
func (r *seriesRepo) DeleteEmpty(ctx context.Context, libraryID string) (int, error) {
	res, err := r.db.ExecContext(ctx, `
		DELETE FROM series
		WHERE library_id = ?
		  AND NOT EXISTS (SELECT 1 FROM book b WHERE b.series_id = series.id)`, libraryID)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func (r *seriesRepo) Get(ctx context.Context, id string) (domain.Series, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+seriesReadColumns+` FROM series WHERE id = ?`, id)
	s, err := scanSeries(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Series{}, domain.ErrNotFound
	}
	return s, err
}

func (r *seriesRepo) GetByFolder(ctx context.Context, libraryID, folderPath string) (domain.Series, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+seriesReadColumns+` FROM series WHERE library_id = ? AND folder_path = ?`,
		libraryID, folderPath)
	s, err := scanSeries(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Series{}, domain.ErrNotFound
	}
	return s, err
}

func (r *seriesRepo) ListByLibrary(ctx context.Context, libraryID string) ([]domain.Series, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+seriesReadColumns+` FROM series WHERE library_id = ? ORDER BY sort_name`, libraryID)
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
		"publisher", "description", "reading_dir", "cover_book_id", "created_at", "updated_at",
		"metadata_state", "match_provider", "match_provider_id"}
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
		mState   string
		mProv    string
		mProvID  string
		bookCnt  int
		readCnt  int
		cover    sql.NullString
	)
	if err := rows.Scan(
		&s.ID, &s.LibraryID, &folder, &s.Name, &s.SortName, &year, &pub,
		&desc, &readDir, &coverBID, &s.CreatedAt, &s.UpdatedAt,
		&mState, &mProv, &mProvID,
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
	s.MetadataState = domain.MetadataState(mState)
	s.MatchProvider = mProv
	s.MatchProviderID = mProvID
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
		mState   string
		mProv    string
		mProvID  string
	)
	if err := row.Scan(
		&s.ID, &s.LibraryID, &folder, &s.Name, &s.SortName, &year, &pub,
		&desc, &readDir, &coverBID, &s.CreatedAt, &s.UpdatedAt,
		&mState, &mProv, &mProvID,
	); err != nil {
		return domain.Series{}, err
	}
	s.FolderPath = str(folder)
	s.Year = int(i64(year))
	s.Publisher = str(pub)
	s.Description = str(desc)
	s.ReadingDir = domain.ReadingDirection(readDir)
	s.CoverBookID = str(coverBID)
	s.MetadataState = domain.MetadataState(mState)
	s.MatchProvider = mProv
	s.MatchProviderID = mProvID
	return s, nil
}
