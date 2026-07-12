package sqlstore

import (
	"context"
	"database/sql"
	"errors"

	"github.com/siposbnc/comic-hub/server/internal/domain"
)

// trackRepo persists the tracker overlay: standalone tracks (track) and the manual issues
// attached to a track or a library series (track_issue). Library series/issues are not
// stored here — the service projects them live from the catalog. Every read/write is
// scoped by user_id.
type trackRepo struct{ db *DB }

func (r *trackRepo) CreateTrack(ctx context.Context, t domain.Track) (domain.Track, error) {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO track (id, user_id, name, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		t.ID, t.UserID, t.Name, t.CreatedAt, t.UpdatedAt,
	)
	if err != nil {
		return domain.Track{}, err
	}
	return t, nil
}

func (r *trackRepo) GetTrack(ctx context.Context, userID, id string) (domain.Track, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, name, created_at, updated_at FROM track WHERE id = ? AND user_id = ?`,
		id, userID)
	var t domain.Track
	err := row.Scan(&t.ID, &t.UserID, &t.Name, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Track{}, domain.ErrNotFound
	}
	return t, err
}

func (r *trackRepo) ListTracks(ctx context.Context, userID string) ([]domain.Track, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, name, created_at, updated_at FROM track WHERE user_id = ? ORDER BY name`,
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Track
	for rows.Next() {
		var t domain.Track
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *trackRepo) RenameTrack(ctx context.Context, t domain.Track) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE track SET name = ?, updated_at = ? WHERE id = ? AND user_id = ?`,
		t.Name, t.UpdatedAt, t.ID, t.UserID)
	if err != nil {
		return err
	}
	return mustAffect(res)
}

func (r *trackRepo) DeleteTrack(ctx context.Context, userID, id string) error {
	// track_issue rows cascade via ON DELETE CASCADE.
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM track WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return err
	}
	return mustAffect(res)
}

func (r *trackRepo) OverlayIssues(ctx context.Context, userID string) ([]domain.TrackIssue, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, COALESCE(track_id, ''), COALESCE(series_id, ''), number, sort,
		       is_read, COALESCE(read_at, 0), created_at
		FROM track_issue WHERE user_id = ? ORDER BY sort`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.TrackIssue
	for rows.Next() {
		it, err := scanTrackIssue(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (r *trackRepo) AddIssues(ctx context.Context, issues []domain.TrackIssue) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for _, it := range issues {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO track_issue
				(id, user_id, track_id, series_id, number, sort, is_read, read_at, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT DO NOTHING`,
			it.ID, it.UserID, nullString(it.TrackID), nullString(it.SeriesID), it.Number, it.Sort,
			boolToInt(it.Read), nullInt(it.ReadAt), it.CreatedAt,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *trackRepo) GetIssue(ctx context.Context, userID, id string) (domain.TrackIssue, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, COALESCE(track_id, ''), COALESCE(series_id, ''), number, sort,
		       is_read, COALESCE(read_at, 0), created_at
		FROM track_issue WHERE id = ? AND user_id = ?`, id, userID)
	it, err := scanTrackIssue(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.TrackIssue{}, domain.ErrNotFound
	}
	return it, err
}

func (r *trackRepo) RemoveIssue(ctx context.Context, userID, id string) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM track_issue WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return err
	}
	return mustAffect(res)
}

func (r *trackRepo) SetIssueRead(ctx context.Context, userID, id string, read bool, at int64) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE track_issue SET is_read = ?, read_at = ? WHERE id = ? AND user_id = ?`,
		boolToInt(read), nullInt(at), id, userID)
	if err != nil {
		return err
	}
	return mustAffect(res)
}

func scanTrackIssue(row rowScanner) (domain.TrackIssue, error) {
	var (
		it   domain.TrackIssue
		read int
	)
	if err := row.Scan(&it.ID, &it.UserID, &it.TrackID, &it.SeriesID, &it.Number, &it.Sort,
		&read, &it.ReadAt, &it.CreatedAt); err != nil {
		return domain.TrackIssue{}, err
	}
	it.Read = read != 0
	return it, nil
}
