package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"github.com/siposbnc/comic-hub/server/internal/domain"
)

// collectionRepo persists collections (collection + collection_item). Item order is a
// fractional `position` so a reorder rewrites a single row.
type collectionRepo struct{ db *sql.DB }

// positionGap is the spacing between appended items, leaving room to insert between them
// without renumbering.
const positionGap = 1024.0

func (r *collectionRepo) Create(ctx context.Context, c domain.Collection) (domain.Collection, error) {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO collection (id, name, description, cover_book_id, owner_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.Name, nullString(c.Description), nullString(c.CoverBookID), nullString(c.OwnerID),
		c.CreatedAt, c.UpdatedAt,
	)
	if err != nil {
		return domain.Collection{}, err
	}
	return c, nil
}

func (r *collectionRepo) Get(ctx context.Context, id string) (domain.Collection, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT c.id, c.name, c.description, c.cover_book_id, c.owner_id, c.created_at, c.updated_at,
			(SELECT COUNT(*) FROM collection_item ci WHERE ci.collection_id = c.id)
		FROM collection c WHERE c.id = ?`, id)
	c, err := scanCollection(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Collection{}, domain.ErrNotFound
	}
	return c, err
}

func (r *collectionRepo) List(ctx context.Context) ([]domain.Collection, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT c.id, c.name, c.description, c.cover_book_id, c.owner_id, c.created_at, c.updated_at,
			(SELECT COUNT(*) FROM collection_item ci WHERE ci.collection_id = c.id)
		FROM collection c ORDER BY c.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Collection
	for rows.Next() {
		c, err := scanCollection(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *collectionRepo) Update(ctx context.Context, c domain.Collection) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE collection SET name = ?, description = ?, cover_book_id = ?, updated_at = ?
		WHERE id = ?`,
		c.Name, nullString(c.Description), nullString(c.CoverBookID), c.UpdatedAt, c.ID,
	)
	if err != nil {
		return err
	}
	return mustAffect(res)
}

func (r *collectionRepo) Delete(ctx context.Context, id string) error {
	// collection_item rows cascade via ON DELETE CASCADE.
	res, err := r.db.ExecContext(ctx, `DELETE FROM collection WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return mustAffect(res)
}

func (r *collectionRepo) Items(ctx context.Context, collectionID string) ([]domain.CollectionItem, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT book_id, position FROM collection_item WHERE collection_id = ? ORDER BY position`,
		collectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.CollectionItem
	for rows.Next() {
		var it domain.CollectionItem
		if err := rows.Scan(&it.BookID, &it.Position); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (r *collectionRepo) AddItems(ctx context.Context, collectionID string, bookIDs []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var maxPos sql.NullFloat64
	if err := tx.QueryRowContext(ctx,
		`SELECT MAX(position) FROM collection_item WHERE collection_id = ?`, collectionID,
	).Scan(&maxPos); err != nil {
		return err
	}

	pos := maxPos.Float64
	for _, bookID := range bookIDs {
		pos += positionGap
		// Skip books already in the collection (keep their place) via INSERT OR IGNORE.
		if _, err := tx.ExecContext(ctx,
			`INSERT OR IGNORE INTO collection_item (collection_id, book_id, position) VALUES (?, ?, ?)`,
			collectionID, bookID, pos,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *collectionRepo) RemoveItem(ctx context.Context, collectionID, bookID string) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM collection_item WHERE collection_id = ? AND book_id = ?`, collectionID, bookID)
	if err != nil {
		return err
	}
	return mustAffect(res)
}

func (r *collectionRepo) SetPosition(ctx context.Context, collectionID, bookID string, position float64) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE collection_item SET position = ? WHERE collection_id = ? AND book_id = ?`,
		position, collectionID, bookID)
	if err != nil {
		return err
	}
	return mustAffect(res)
}

func (r *collectionRepo) IDsForBook(ctx context.Context, bookID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT collection_id FROM collection_item WHERE book_id = ?`, bookID)
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

func scanCollection(row rowScanner) (domain.Collection, error) {
	var (
		c     domain.Collection
		desc  sql.NullString
		cover sql.NullString
		owner sql.NullString
	)
	if err := row.Scan(&c.ID, &c.Name, &desc, &cover, &owner, &c.CreatedAt, &c.UpdatedAt, &c.BookCount); err != nil {
		return domain.Collection{}, err
	}
	c.Description = str(desc)
	c.CoverBookID = str(cover)
	c.OwnerID = str(owner)
	return c, nil
}

// mustAffect maps a zero-rows result to ErrNotFound for update/delete style statements.
func mustAffect(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}
