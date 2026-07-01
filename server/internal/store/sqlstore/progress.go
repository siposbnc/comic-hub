package sqlstore

import (
	"context"
	"database/sql"
	"errors"

	"github.com/siposbnc/comic-hub/server/internal/domain"
)

// progressRepo persists per-user reading progress. Writes are last-writer-wins by
// updated_at (the service enforces that); this layer just stores what it's given.
type progressRepo struct{ db *DB }

const progressColumns = `user_id, book_id, page, page_count, status, started_at,
	finished_at, updated_at, device`

func (r *progressRepo) Upsert(ctx context.Context, p domain.Progress) (domain.Progress, error) {
	status := p.Status
	if status == "" {
		status = domain.StatusUnread
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO read_progress (`+progressColumns+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, book_id) DO UPDATE SET
			page        = excluded.page,
			page_count  = excluded.page_count,
			status      = excluded.status,
			started_at  = excluded.started_at,
			finished_at = excluded.finished_at,
			updated_at  = excluded.updated_at,
			device      = excluded.device`,
		p.UserID, p.BookID, p.Page, p.PageCount, string(status), nullInt(p.StartedAt),
		nullInt(p.FinishedAt), p.UpdatedAt, nullString(p.Device),
	)
	if err != nil {
		return domain.Progress{}, err
	}
	p.Status = status
	return p, nil
}

func (r *progressRepo) Get(ctx context.Context, userID, bookID string) (domain.Progress, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+progressColumns+` FROM read_progress WHERE user_id = ? AND book_id = ?`,
		userID, bookID)
	p, err := scanProgress(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Progress{}, domain.ErrNotFound
	}
	return p, err
}

func (r *progressRepo) ContinueReading(ctx context.Context, userID string, limit int) ([]domain.Progress, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+progressColumns+`
		FROM read_progress
		WHERE user_id = ? AND status = ?
		ORDER BY updated_at DESC
		LIMIT ?`, userID, string(domain.StatusInProgress), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Progress
	for rows.Next() {
		p, err := scanProgress(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func scanProgress(row rowScanner) (domain.Progress, error) {
	var (
		p        domain.Progress
		started  sql.NullInt64
		finished sql.NullInt64
		device   sql.NullString
	)
	if err := row.Scan(
		&p.UserID, &p.BookID, &p.Page, &p.PageCount, &p.Status,
		&started, &finished, &p.UpdatedAt, &device,
	); err != nil {
		return domain.Progress{}, err
	}
	p.StartedAt = i64(started)
	p.FinishedAt = i64(finished)
	p.Device = str(device)
	return p, nil
}
