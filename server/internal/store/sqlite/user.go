package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/siposbnc/comic-hub/server/internal/domain"
)

// userRepo persists accounts (the `user` table, seeded with the implicit owner in 0002).
type userRepo struct{ db *sql.DB }

const userColumns = `id, username, display_name, role, password_hash, age_rating_max, created_at`

func (r *userRepo) Create(ctx context.Context, u domain.User) (domain.User, error) {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO user (id, username, display_name, role, password_hash, age_rating_max, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		u.ID, u.Username, u.DisplayName, string(u.Role),
		nullString(u.PasswordHash), nullString(u.AgeRatingMax), u.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.User{}, domain.ErrConflict
		}
		return domain.User{}, err
	}
	return u, nil
}

func (r *userRepo) Get(ctx context.Context, id string) (domain.User, error) {
	return scanUser(r.db.QueryRowContext(ctx, `SELECT `+userColumns+` FROM user WHERE id = ?`, id))
}

func (r *userRepo) GetByUsername(ctx context.Context, username string) (domain.User, error) {
	return scanUser(r.db.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM user WHERE username = ?`, username))
}

func (r *userRepo) List(ctx context.Context) ([]domain.User, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+userColumns+` FROM user ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *userRepo) Count(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM user`).Scan(&n)
	return n, err
}

func (r *userRepo) Update(ctx context.Context, u domain.User) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE user SET display_name = ?, role = ?, age_rating_max = ? WHERE id = ?`,
		u.DisplayName, string(u.Role), nullString(u.AgeRatingMax), u.ID)
	if err != nil {
		return err
	}
	return mustAffect(res)
}

func (r *userRepo) SetPasswordHash(ctx context.Context, id, hash string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE user SET password_hash = ? WHERE id = ?`, nullString(hash), id)
	if err != nil {
		return err
	}
	return mustAffect(res)
}

func (r *userRepo) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM user WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return mustAffect(res)
}

func scanUser(row rowScanner) (domain.User, error) {
	var (
		u      domain.User
		role   string
		pwHash sql.NullString
		ageMax sql.NullString
	)
	if err := row.Scan(&u.ID, &u.Username, &u.DisplayName, &role, &pwHash, &ageMax, &u.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, domain.ErrNotFound
		}
		return domain.User{}, err
	}
	u.Role = domain.UserRole(role)
	u.PasswordHash = str(pwHash)
	u.AgeRatingMax = str(ageMax)
	return u, nil
}

// sessionRepo persists refresh-token sessions (0013).
type sessionRepo struct{ db *sql.DB }

func (r *sessionRepo) Create(ctx context.Context, s domain.Session) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO session (id, user_id, refresh_hash, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		s.ID, s.UserID, s.RefreshHash, s.ExpiresAt, s.CreatedAt)
	return err
}

func (r *sessionRepo) GetByHash(ctx context.Context, refreshHash string) (domain.Session, error) {
	var s domain.Session
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, refresh_hash, expires_at, created_at FROM session WHERE refresh_hash = ?`,
		refreshHash).Scan(&s.ID, &s.UserID, &s.RefreshHash, &s.ExpiresAt, &s.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Session{}, domain.ErrNotFound
	}
	return s, err
}

func (r *sessionRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM session WHERE id = ?`, id)
	return err
}

func (r *sessionRepo) DeleteForUser(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM session WHERE user_id = ?`, userID)
	return err
}

func (r *sessionRepo) DeleteExpired(ctx context.Context, now int64) (int, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM session WHERE expires_at <= ?`, now)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// isUniqueViolation reports whether err is a SQLite UNIQUE constraint failure.
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}
