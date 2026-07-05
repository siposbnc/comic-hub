package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/pkg/ulid"
)

// readingListRepo persists per-user reading lists (reading_list + reading_list_item).
// Every read/write is scoped by user_id; ordering uses the same fractional position as
// collections.
type readingListRepo struct{ db *DB }

func (r *readingListRepo) Create(ctx context.Context, l domain.ReadingList) (domain.ReadingList, error) {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO reading_list (id, user_id, name, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		l.ID, l.UserID, l.Name, l.CreatedAt, l.UpdatedAt,
	)
	if err != nil {
		return domain.ReadingList{}, err
	}
	return l, nil
}

func (r *readingListRepo) Get(ctx context.Context, userID, id string) (domain.ReadingList, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT l.id, l.user_id, l.name, l.is_active, l.created_at, l.updated_at,
			(SELECT COUNT(*) FROM reading_list_item li WHERE li.list_id = l.id)
		FROM reading_list l WHERE l.id = ? AND l.user_id = ?`, id, userID)
	l, err := scanReadingList(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ReadingList{}, domain.ErrNotFound
	}
	return l, err
}

func (r *readingListRepo) List(ctx context.Context, userID string) ([]domain.ReadingList, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT l.id, l.user_id, l.name, l.is_active, l.created_at, l.updated_at,
			(SELECT COUNT(*) FROM reading_list_item li WHERE li.list_id = l.id)
		FROM reading_list l WHERE l.user_id = ? ORDER BY l.name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.ReadingList
	for rows.Next() {
		l, err := scanReadingList(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (r *readingListRepo) SetActive(ctx context.Context, userID, id string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx,
		`UPDATE reading_list SET is_active = 1 WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return err
	}
	if err := mustAffect(res); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE reading_list SET is_active = 0 WHERE user_id = ? AND id <> ?`, userID, id); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *readingListRepo) GetActive(ctx context.Context, userID string) (domain.ReadingList, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT l.id, l.user_id, l.name, l.is_active, l.created_at, l.updated_at,
			(SELECT COUNT(*) FROM reading_list_item li WHERE li.list_id = l.id)
		FROM reading_list l WHERE l.user_id = ? AND l.is_active = 1 LIMIT 1`, userID)
	l, err := scanReadingList(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ReadingList{}, domain.ErrNotFound
	}
	return l, err
}

func (r *readingListRepo) Update(ctx context.Context, l domain.ReadingList) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE reading_list SET name = ?, updated_at = ? WHERE id = ? AND user_id = ?`,
		l.Name, l.UpdatedAt, l.ID, l.UserID,
	)
	if err != nil {
		return err
	}
	return mustAffect(res)
}

func (r *readingListRepo) Delete(ctx context.Context, userID, id string) error {
	// reading_list_item rows cascade via ON DELETE CASCADE.
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM reading_list WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return err
	}
	return mustAffect(res)
}

func (r *readingListRepo) Items(ctx context.Context, listID string) ([]domain.ReadingListItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, book_id, position, added_at, series_name, number, title, content_hash
		FROM reading_list_item WHERE list_id = ? ORDER BY position`, listID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.ReadingListItem
	for rows.Next() {
		var (
			it     domain.ReadingListItem
			bookID sql.NullString
		)
		if err := rows.Scan(&it.ID, &bookID, &it.Position, &it.AddedAt,
			&it.SeriesName, &it.Number, &it.Title, &it.ContentHash); err != nil {
			return nil, err
		}
		it.BookID = str(bookID)
		out = append(out, it)
	}
	return out, rows.Err()
}

// itemSnapshotSelect pulls the display snapshot for a book (series name, number, title,
// content hash) — captured on add/relink so the row outlives the book.
const itemSnapshotSelect = `
	SELECT COALESCE(s.name, ''), COALESCE(b.number, ''), COALESCE(b.title, ''), COALESCE(b.content_hash, '')
	FROM book b LEFT JOIN series s ON s.id = b.series_id WHERE b.id = ?`

func (r *readingListRepo) AddItems(ctx context.Context, listID string, bookIDs []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var maxPos sql.NullFloat64
	if err := tx.QueryRowContext(ctx,
		`SELECT MAX(position) FROM reading_list_item WHERE list_id = ?`, listID,
	).Scan(&maxPos); err != nil {
		return err
	}

	now := time.Now().UnixMilli()
	pos := maxPos.Float64
	for _, bookID := range bookIDs {
		pos += positionGap
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO reading_list_item
				(id, list_id, book_id, position, added_at, series_name, number, title, content_hash)
			SELECT ?, ?, b.id, ?, ?,
				COALESCE(s.name, ''), COALESCE(b.number, ''), COALESCE(b.title, ''), COALESCE(b.content_hash, '')
			FROM book b LEFT JOIN series s ON s.id = b.series_id
			WHERE b.id = ?
			ON CONFLICT DO NOTHING`,
			ulid.New(), listID, pos, now, bookID,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *readingListRepo) AddManualItems(ctx context.Context, listID string, entries []domain.ManualListItem) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var maxPos sql.NullFloat64
	if err := tx.QueryRowContext(ctx,
		`SELECT MAX(position) FROM reading_list_item WHERE list_id = ?`, listID,
	).Scan(&maxPos); err != nil {
		return err
	}

	now := time.Now().UnixMilli()
	pos := maxPos.Float64
	for _, e := range entries {
		pos += positionGap
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO reading_list_item
				(id, list_id, book_id, position, added_at, series_name, number, title, content_hash)
			VALUES (?, ?, NULL, ?, ?, ?, ?, ?, '')`,
			ulid.New(), listID, pos, now, e.SeriesName, e.Number, e.Title,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *readingListRepo) RemoveItem(ctx context.Context, listID, ref string) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM reading_list_item WHERE list_id = ? AND (id = ? OR book_id = ?)`,
		listID, ref, ref)
	if err != nil {
		return err
	}
	return mustAffect(res)
}

func (r *readingListRepo) SetPosition(ctx context.Context, listID, ref string, position float64) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE reading_list_item SET position = ? WHERE list_id = ? AND (id = ? OR book_id = ?)`,
		position, listID, ref, ref)
	if err != nil {
		return err
	}
	return mustAffect(res)
}

func (r *readingListRepo) Relink(ctx context.Context, listID, itemID, bookID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var name, number, title, hash string
	err = tx.QueryRowContext(ctx, itemSnapshotSelect, bookID).Scan(&name, &number, &title, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotFound
	}
	if err != nil {
		return err
	}

	// The unique (list_id, book_id) index would reject this anyway; pre-check so the
	// caller gets a friendly validation error instead of a driver-specific one.
	var already int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM reading_list_item WHERE list_id = ? AND book_id = ? AND id <> ?`,
		listID, bookID, itemID).Scan(&already); err != nil {
		return err
	}
	if already > 0 {
		return fmt.Errorf("%w: that issue is already in this list", domain.ErrValidation)
	}

	res, err := tx.ExecContext(ctx, `
		UPDATE reading_list_item
		SET book_id = ?, series_name = ?, number = ?, title = ?, content_hash = ?
		WHERE list_id = ? AND id = ?`,
		bookID, name, number, title, hash, listID, itemID)
	if err != nil {
		return err
	}
	if err := mustAffect(res); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *readingListRepo) RelinkStaleByHash(ctx context.Context, contentHash, bookID string) (int, error) {
	if contentHash == "" {
		return 0, nil
	}
	// Snapshot refresh happens via subselects; the NOT EXISTS guard skips lists that
	// already contain the book (e.g. duplicate stale rows for the same file).
	res, err := r.db.ExecContext(ctx, `
		UPDATE reading_list_item
		SET book_id = ?,
			series_name = COALESCE((SELECT s.name FROM book b LEFT JOIN series s ON s.id = b.series_id WHERE b.id = ?), ''),
			number      = COALESCE((SELECT b.number FROM book b WHERE b.id = ?), ''),
			title       = COALESCE((SELECT b.title FROM book b WHERE b.id = ?), '')
		WHERE book_id IS NULL AND content_hash = ?
		  AND NOT EXISTS (
			SELECT 1 FROM reading_list_item x
			WHERE x.list_id = reading_list_item.list_id AND x.book_id = ?)`,
		bookID, bookID, bookID, bookID, contentHash, bookID)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func (r *readingListRepo) IDsForBook(ctx context.Context, userID, bookID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT li.list_id FROM reading_list_item li
		JOIN reading_list l ON l.id = li.list_id
		WHERE li.book_id = ? AND l.user_id = ?`, bookID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func scanReadingList(row rowScanner) (domain.ReadingList, error) {
	var (
		l      domain.ReadingList
		active int
	)
	if err := row.Scan(&l.ID, &l.UserID, &l.Name, &active, &l.CreatedAt, &l.UpdatedAt, &l.BookCount); err != nil {
		return domain.ReadingList{}, err
	}
	l.Active = active != 0
	return l, nil
}
