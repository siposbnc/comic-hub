package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/siposbnc/comic-hub/server/internal/domain"
)

// readingListRepo persists per-user reading lists (reading_list + reading_list_item).
// Every read/write is scoped by user_id; ordering uses the same fractional position as
// collections.
type readingListRepo struct{ db *sql.DB }

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
		SELECT l.id, l.user_id, l.name, l.created_at, l.updated_at,
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
		SELECT l.id, l.user_id, l.name, l.created_at, l.updated_at,
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
	rows, err := r.db.QueryContext(ctx,
		`SELECT book_id, position FROM reading_list_item WHERE list_id = ? ORDER BY position`, listID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.ReadingListItem
	for rows.Next() {
		var it domain.ReadingListItem
		if err := rows.Scan(&it.BookID, &it.Position); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

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
		if _, err := tx.ExecContext(ctx,
			`INSERT OR IGNORE INTO reading_list_item (list_id, book_id, position, added_at) VALUES (?, ?, ?, ?)`,
			listID, bookID, pos, now,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *readingListRepo) RemoveItem(ctx context.Context, listID, bookID string) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM reading_list_item WHERE list_id = ? AND book_id = ?`, listID, bookID)
	if err != nil {
		return err
	}
	return mustAffect(res)
}

func (r *readingListRepo) SetPosition(ctx context.Context, listID, bookID string, position float64) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE reading_list_item SET position = ? WHERE list_id = ? AND book_id = ?`,
		position, listID, bookID)
	if err != nil {
		return err
	}
	return mustAffect(res)
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
	var l domain.ReadingList
	if err := row.Scan(&l.ID, &l.UserID, &l.Name, &l.CreatedAt, &l.UpdatedAt, &l.BookCount); err != nil {
		return domain.ReadingList{}, err
	}
	return l, nil
}
