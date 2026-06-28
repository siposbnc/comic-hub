package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/siposbnc/comic-hub/server/internal/domain"
)

// readerPrefRepo persists per-user, per-book reader overrides (opaque settings JSON).
type readerPrefRepo struct{ db *sql.DB }

func (r *readerPrefRepo) Get(ctx context.Context, userID, bookID string) (string, error) {
	var settings string
	err := r.db.QueryRowContext(ctx,
		`SELECT settings FROM reader_pref WHERE user_id = ? AND book_id = ?`, userID, bookID,
	).Scan(&settings)
	if errors.Is(err, sql.ErrNoRows) {
		return "", domain.ErrNotFound
	}
	return settings, err
}

func (r *readerPrefRepo) Put(ctx context.Context, userID, bookID, settings string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO reader_pref (user_id, book_id, settings, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(user_id, book_id) DO UPDATE SET
			settings = excluded.settings,
			updated_at = excluded.updated_at`,
		userID, bookID, settings, time.Now().UnixMilli())
	return err
}
