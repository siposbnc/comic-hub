package sqlstore

import (
	"context"
	"database/sql"
	"errors"

	"github.com/siposbnc/comic-hub/server/internal/domain"
)

// bookmarkRepo persists per-user, per-book bookmarks (page + optional note).
type bookmarkRepo struct{ db *DB }

func scanBookmark(s interface {
	Scan(dest ...any) error
}) (domain.Bookmark, error) {
	var b domain.Bookmark
	err := s.Scan(&b.ID, &b.UserID, &b.BookID, &b.Page, &b.Note, &b.CreatedAt, &b.UpdatedAt)
	return b, err
}

const bookmarkCols = `id, user_id, book_id, page, note, created_at, updated_at`

func (r *bookmarkRepo) List(ctx context.Context, userID, bookID string) ([]domain.Bookmark, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+bookmarkCols+` FROM bookmark
		 WHERE user_id = ? AND book_id = ?
		 ORDER BY page ASC`, userID, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Bookmark, 0)
	for rows.Next() {
		b, err := scanBookmark(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *bookmarkRepo) Get(ctx context.Context, userID, id string) (domain.Bookmark, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+bookmarkCols+` FROM bookmark WHERE id = ? AND user_id = ?`, id, userID)
	b, err := scanBookmark(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Bookmark{}, domain.ErrNotFound
	}
	return b, err
}

func (r *bookmarkRepo) GetByPage(ctx context.Context, userID, bookID string, page int) (domain.Bookmark, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+bookmarkCols+` FROM bookmark WHERE user_id = ? AND book_id = ? AND page = ?`,
		userID, bookID, page)
	b, err := scanBookmark(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Bookmark{}, domain.ErrNotFound
	}
	return b, err
}

func (r *bookmarkRepo) Create(ctx context.Context, b domain.Bookmark) (domain.Bookmark, error) {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO bookmark (`+bookmarkCols+`) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		b.ID, b.UserID, b.BookID, b.Page, b.Note, b.CreatedAt, b.UpdatedAt)
	if err != nil {
		return domain.Bookmark{}, err
	}
	return b, nil
}

func (r *bookmarkRepo) UpdateNote(ctx context.Context, userID, id, note string, updatedAt int64) (domain.Bookmark, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE bookmark SET note = ?, updated_at = ? WHERE id = ? AND user_id = ?`,
		note, updatedAt, id, userID)
	if err != nil {
		return domain.Bookmark{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.Bookmark{}, domain.ErrNotFound
	}
	return r.Get(ctx, userID, id)
}

func (r *bookmarkRepo) Delete(ctx context.Context, userID, id string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM bookmark WHERE id = ? AND user_id = ?`, id, userID)
	return err
}
