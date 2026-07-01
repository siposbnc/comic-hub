package sqlstore

import (
	"context"
	"database/sql"
	"errors"

	"github.com/siposbnc/comic-hub/server/internal/domain"
)

// jobRepo persists background job records.
type jobRepo struct{ db *DB }

const jobColumns = `id, type, state, payload, progress, total, done, error,
	created_at, started_at, finished_at`

func (r *jobRepo) Create(ctx context.Context, j domain.Job) (domain.Job, error) {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO job (id, type, state, payload, progress, total, done, error, created_at, started_at, finished_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		j.ID, j.Type, string(j.State), j.Payload, j.Progress, nullInt(j.Total),
		j.Done, nullString(j.Error), j.CreatedAt, nullInt(j.StartedAt), nullInt(j.FinishedAt),
	)
	if err != nil {
		return domain.Job{}, err
	}
	return j, nil
}

func (r *jobRepo) Update(ctx context.Context, j domain.Job) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE job SET
			state = ?, payload = ?, progress = ?, total = ?, done = ?, error = ?,
			started_at = ?, finished_at = ?
		WHERE id = ?`,
		string(j.State), j.Payload, j.Progress, nullInt(j.Total), j.Done,
		nullString(j.Error), nullInt(j.StartedAt), nullInt(j.FinishedAt), j.ID,
	)
	return err
}

func (r *jobRepo) Get(ctx context.Context, id string) (domain.Job, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+jobColumns+` FROM job WHERE id = ?`, id)
	j, err := scanJob(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Job{}, domain.ErrNotFound
	}
	return j, err
}

func (r *jobRepo) ListByState(ctx context.Context, state domain.JobState, limit int) ([]domain.Job, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+jobColumns+` FROM job WHERE state = ? ORDER BY created_at DESC LIMIT ?`,
		string(state), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Job
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

func scanJob(row rowScanner) (domain.Job, error) {
	var (
		j        domain.Job
		state    string
		total    sql.NullInt64
		errMsg   sql.NullString
		started  sql.NullInt64
		finished sql.NullInt64
	)
	if err := row.Scan(
		&j.ID, &j.Type, &state, &j.Payload, &j.Progress, &total, &j.Done,
		&errMsg, &j.CreatedAt, &started, &finished,
	); err != nil {
		return domain.Job{}, err
	}
	j.State = domain.JobState(state)
	j.Total = i64(total)
	j.Error = str(errMsg)
	j.StartedAt = i64(started)
	j.FinishedAt = i64(finished)
	return j, nil
}
