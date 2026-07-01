package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/pkg/ulid"
)

// libraryRepo persists libraries and their roots. A library's roots live in the
// library_root table; reads and writes keep them together so callers see a whole
// domain.Library.
type libraryRepo struct{ db *DB }

func (r *libraryRepo) Create(ctx context.Context, lib domain.Library) (domain.Library, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Library{}, err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO library (id, name, kind, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		lib.ID, lib.Name, lib.Kind, lib.CreatedAt, lib.UpdatedAt,
	); err != nil {
		return domain.Library{}, fmt.Errorf("insert library: %w", err)
	}

	for _, root := range lib.Roots {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO library_root (id, library_id, path, enabled) VALUES (?, ?, ?, 1)`,
			ulid.New(), lib.ID, root,
		); err != nil {
			return domain.Library{}, fmt.Errorf("insert library root: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return domain.Library{}, err
	}
	return lib, nil
}

func (r *libraryRepo) Get(ctx context.Context, id string) (domain.Library, error) {
	var lib domain.Library
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, kind, created_at, updated_at FROM library WHERE id = ?`, id,
	).Scan(&lib.ID, &lib.Name, &lib.Kind, &lib.CreatedAt, &lib.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Library{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Library{}, err
	}

	roots, err := r.roots(ctx, id)
	if err != nil {
		return domain.Library{}, err
	}
	lib.Roots = roots
	return lib, nil
}

func (r *libraryRepo) List(ctx context.Context) ([]domain.Library, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, kind, created_at, updated_at FROM library ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var libs []domain.Library
	for rows.Next() {
		var lib domain.Library
		if err := rows.Scan(&lib.ID, &lib.Name, &lib.Kind, &lib.CreatedAt, &lib.UpdatedAt); err != nil {
			return nil, err
		}
		libs = append(libs, lib)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Attach roots per library. The library count is small (handful of roots each),
	// so a query per library is fine; revisit if libraries ever number in the hundreds.
	for i := range libs {
		roots, err := r.roots(ctx, libs[i].ID)
		if err != nil {
			return nil, err
		}
		libs[i].Roots = roots
	}
	return libs, nil
}

func (r *libraryRepo) Delete(ctx context.Context, id string) error {
	// library_root / series / book cascade via ON DELETE CASCADE.
	res, err := r.db.ExecContext(ctx, `DELETE FROM library WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *libraryRepo) roots(ctx context.Context, libraryID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT path FROM library_root WHERE library_id = ? ORDER BY path`, libraryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roots []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		roots = append(roots, p)
	}
	return roots, rows.Err()
}
