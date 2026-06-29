package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/pkg/ulid"
)

// ReplaceSeriesStoryArcs swaps a series' story arcs and their book links in one tx, so a
// re-match never leaves stale arcs behind.
func (r *metadataRepo) ReplaceSeriesStoryArcs(ctx context.Context, seriesID string, arcs []domain.StoryArcInput) error {
	now := time.Now().UnixMilli()
	return r.inTx(ctx, func(tx *sql.Tx) error {
		// ON DELETE CASCADE clears story_arc_book for the removed arcs.
		if _, err := tx.ExecContext(ctx, `DELETE FROM story_arc WHERE series_id = ?`, seriesID); err != nil {
			return err
		}
		for _, a := range arcs {
			arcID := ulid.New()
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO story_arc (id, series_id, provider, provider_id, name, description, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?)`,
				arcID, seriesID, "", a.ProviderID, a.Name, a.Description, now); err != nil {
				return err
			}
			for _, bookID := range a.BookIDs {
				if _, err := tx.ExecContext(ctx,
					`INSERT OR IGNORE INTO story_arc_book (arc_id, book_id) VALUES (?, ?)`,
					arcID, bookID); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

const storyArcCountExpr = `(SELECT COUNT(*) FROM story_arc_book sab WHERE sab.arc_id = sa.id)`

// minArcBookSort orders arcs by where they start in the run.
const minArcBookSort = `(SELECT MIN(b.sort_number) FROM story_arc_book sab
	JOIN book b ON b.id = sab.book_id WHERE sab.arc_id = sa.id)`

func (r *metadataRepo) SeriesStoryArcs(ctx context.Context, seriesID string) ([]domain.StoryArc, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT sa.id, sa.series_id, sa.name, sa.description, `+storyArcCountExpr+`
		FROM story_arc sa
		WHERE sa.series_id = ?
		ORDER BY `+minArcBookSort+`, sa.name`, seriesID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.StoryArc, 0)
	for rows.Next() {
		var a domain.StoryArc
		if err := rows.Scan(&a.ID, &a.SeriesID, &a.Name, &a.Description, &a.IssueCount); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *metadataRepo) StoryArc(ctx context.Context, arcID string) (domain.StoryArc, error) {
	var a domain.StoryArc
	err := r.db.QueryRowContext(ctx, `
		SELECT sa.id, sa.series_id, sa.name, sa.description, `+storyArcCountExpr+`
		FROM story_arc sa WHERE sa.id = ?`, arcID).
		Scan(&a.ID, &a.SeriesID, &a.Name, &a.Description, &a.IssueCount)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.StoryArc{}, domain.ErrNotFound
	}
	return a, err
}

func (r *metadataRepo) StoryArcBookIDs(ctx context.Context, arcID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT sab.book_id FROM story_arc_book sab
		JOIN book b ON b.id = sab.book_id
		WHERE sab.arc_id = ?
		ORDER BY b.sort_number, b.number`, arcID)
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
