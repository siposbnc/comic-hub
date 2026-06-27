package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/pkg/ulid"
)

// metadataRepo persists the per-book metadata envelope (scalar fields + provider ids +
// locked fields) and the normalized credit/genre/character link tables.
type metadataRepo struct{ db *sql.DB }

func (r *metadataRepo) WriteBookMeta(ctx context.Context, bookID string, m domain.BookMeta) error {
	state := m.State
	if state == "" {
		state = domain.MetaMatched
	}
	providerIDs, err := json.Marshal(nonNilMap(m.ProviderIDs))
	if err != nil {
		return err
	}
	locked, err := json.Marshal(nonNilSlice(m.LockedFields))
	if err != nil {
		return err
	}

	res, err := r.db.ExecContext(ctx, `
		UPDATE book SET
			title = ?, number = ?, volume = ?, release_date = ?, age_rating = ?,
			language = ?, summary = ?, metadata_state = ?, provider_ids = ?,
			locked_fields = ?, updated_at = ?
		WHERE id = ?`,
		nullString(m.Title), nullString(m.Number), nullInt(int64(m.Volume)), nullInt(m.ReleaseDate),
		nullString(m.AgeRating), nullString(m.Language), nullString(m.Summary), string(state),
		string(providerIDs), string(locked), time.Now().UnixMilli(), bookID,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *metadataRepo) LockedBookFields(ctx context.Context, bookID string) ([]string, error) {
	return scanJSONSlice(r.db.QueryRowContext(ctx, `SELECT locked_fields FROM book WHERE id = ?`, bookID))
}

func (r *metadataRepo) BookProviderIDs(ctx context.Context, bookID string) (map[string]string, error) {
	var raw sql.NullString
	err := r.db.QueryRowContext(ctx, `SELECT provider_ids FROM book WHERE id = ?`, bookID).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	if raw.String != "" {
		_ = json.Unmarshal([]byte(raw.String), &out)
	}
	return out, nil
}

func (r *metadataRepo) ReplaceBookPeople(ctx context.Context, bookID string, credits map[string][]string) error {
	return r.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM book_person WHERE book_id = ?`, bookID); err != nil {
			return err
		}
		for role, names := range credits {
			role = strings.TrimSpace(role)
			for _, name := range names {
				id, err := getOrCreateNamed(ctx, tx, "person", name)
				if err != nil {
					return err
				}
				if id == "" {
					continue
				}
				if _, err := tx.ExecContext(ctx,
					`INSERT OR IGNORE INTO book_person (book_id, person_id, role) VALUES (?, ?, ?)`,
					bookID, id, role); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (r *metadataRepo) ReplaceBookGenres(ctx context.Context, bookID string, names []string) error {
	return r.replaceLinks(ctx, bookID, "genre", "book_genre", "genre_id", names)
}

func (r *metadataRepo) ReplaceBookCharacters(ctx context.Context, bookID string, names []string) error {
	return r.replaceLinks(ctx, bookID, "character", "book_character", "character_id", names)
}

// replaceLinks swaps a book's rows in a simple (book_id, <entity>_id) join table, creating
// entities by name as needed. table/joinTable/joinCol are fixed constants, never user input.
func (r *metadataRepo) replaceLinks(ctx context.Context, bookID, table, joinTable, joinCol string, names []string) error {
	return r.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM `+joinTable+` WHERE book_id = ?`, bookID); err != nil {
			return err
		}
		for _, name := range names {
			id, err := getOrCreateNamed(ctx, tx, table, name)
			if err != nil {
				return err
			}
			if id == "" {
				continue
			}
			if _, err := tx.ExecContext(ctx,
				`INSERT OR IGNORE INTO `+joinTable+` (book_id, `+joinCol+`) VALUES (?, ?)`,
				bookID, id); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *metadataRepo) BookCredits(ctx context.Context, bookID string) (map[string][]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT bp.role, p.name FROM book_person bp
		JOIN person p ON p.id = bp.person_id
		WHERE bp.book_id = ? ORDER BY bp.role, p.name`, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]string{}
	for rows.Next() {
		var role, name string
		if err := rows.Scan(&role, &name); err != nil {
			return nil, err
		}
		out[role] = append(out[role], name)
	}
	return out, rows.Err()
}

func (r *metadataRepo) BookGenres(ctx context.Context, bookID string) ([]string, error) {
	return r.listNames(ctx, `SELECT g.name FROM book_genre bg JOIN genre g ON g.id = bg.genre_id WHERE bg.book_id = ? ORDER BY g.name`, bookID)
}

func (r *metadataRepo) BookCharacters(ctx context.Context, bookID string) ([]string, error) {
	return r.listNames(ctx, `SELECT c.name FROM book_character bc JOIN character c ON c.id = bc.character_id WHERE bc.book_id = ? ORDER BY c.name`, bookID)
}

func (r *metadataRepo) listNames(ctx context.Context, query, arg string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

func (r *metadataRepo) inTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

// getOrCreateNamed returns the id of the row in `table` (person/genre/character) with the
// given name, inserting it if absent. `table` is a fixed constant, never user input.
func getOrCreateNamed(ctx context.Context, tx *sql.Tx, table, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", nil
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT OR IGNORE INTO `+table+` (id, name) VALUES (?, ?)`, ulid.New(), name); err != nil {
		return "", err
	}
	var id string
	err := tx.QueryRowContext(ctx, `SELECT id FROM `+table+` WHERE name = ?`, name).Scan(&id)
	return id, err
}

func scanJSONSlice(row rowScanner) ([]string, error) {
	var raw sql.NullString
	if err := row.Scan(&raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	var out []string
	if raw.String != "" {
		_ = json.Unmarshal([]byte(raw.String), &out)
	}
	return out, nil
}

func nonNilMap(m map[string]string) map[string]string {
	if m == nil {
		return map[string]string{}
	}
	return m
}

func nonNilSlice(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
