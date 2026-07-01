package sqlstore

import (
	"context"
	"database/sql"
	"errors"

	"github.com/siposbnc/comic-hub/server/internal/domain"
)

// tagRepo persists tags (tag) and their book assignments (book_tag).
type tagRepo struct{ db *DB }

func (r *tagRepo) Create(ctx context.Context, t domain.Tag) (domain.Tag, error) {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO tag (id, name, color) VALUES (?, ?, ?)`,
		t.ID, t.Name, nullString(t.Color))
	if err != nil {
		return domain.Tag{}, err
	}
	return t, nil
}

func (r *tagRepo) Get(ctx context.Context, id string) (domain.Tag, error) {
	return r.scanOne(ctx, `WHERE t.id = ?`, id)
}

func (r *tagRepo) GetByName(ctx context.Context, name string) (domain.Tag, error) {
	return r.scanOne(ctx, `WHERE t.name = ? COLLATE NOCASE`, name)
}

func (r *tagRepo) scanOne(ctx context.Context, where string, arg any) (domain.Tag, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT t.id, t.name, t.color,
			(SELECT COUNT(*) FROM book_tag bt WHERE bt.tag_id = t.id)
		FROM tag t `+where, arg)
	t, err := scanTag(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Tag{}, domain.ErrNotFound
	}
	return t, err
}

func (r *tagRepo) List(ctx context.Context) ([]domain.Tag, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT t.id, t.name, t.color,
			(SELECT COUNT(*) FROM book_tag bt WHERE bt.tag_id = t.id)
		FROM tag t ORDER BY t.name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Tag
	for rows.Next() {
		t, err := scanTag(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *tagRepo) Update(ctx context.Context, t domain.Tag) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE tag SET name = ?, color = ? WHERE id = ?`, t.Name, nullString(t.Color), t.ID)
	if err != nil {
		return err
	}
	return mustAffect(res)
}

func (r *tagRepo) Delete(ctx context.Context, id string) error {
	// book_tag rows cascade via ON DELETE CASCADE.
	res, err := r.db.ExecContext(ctx, `DELETE FROM tag WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return mustAffect(res)
}

func (r *tagRepo) AssignToBook(ctx context.Context, bookID string, tagIDs []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, tagID := range tagIDs {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO book_tag (book_id, tag_id) VALUES (?, ?) ON CONFLICT DO NOTHING`, bookID, tagID,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *tagRepo) UnassignFromBook(ctx context.Context, bookID, tagID string) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM book_tag WHERE book_id = ? AND tag_id = ?`, bookID, tagID)
	if err != nil {
		return err
	}
	return mustAffect(res)
}

func (r *tagRepo) BookTags(ctx context.Context, bookID string) ([]domain.Tag, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT t.id, t.name, t.color, 0
		FROM book_tag bt JOIN tag t ON t.id = bt.tag_id
		WHERE bt.book_id = ? ORDER BY t.name COLLATE NOCASE`, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Tag
	for rows.Next() {
		t, err := scanTag(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *tagRepo) TaggedBookIDs(ctx context.Context, tagID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT bt.book_id FROM book_tag bt
		JOIN book b ON b.id = bt.book_id
		WHERE bt.tag_id = ? ORDER BY b.added_at DESC`, tagID)
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

func scanTag(row rowScanner) (domain.Tag, error) {
	var (
		t     domain.Tag
		color sql.NullString
	)
	if err := row.Scan(&t.ID, &t.Name, &color, &t.BookCount); err != nil {
		return domain.Tag{}, err
	}
	t.Color = str(color)
	return t, nil
}
