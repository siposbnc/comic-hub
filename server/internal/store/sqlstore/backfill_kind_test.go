package sqlstore

import (
	"context"
	"io/fs"
	"testing"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/pkg/ulid"
)

// TestBackfillBookKind runs the actual 0017 backfill migration SQL against rows that predate
// classification (kind defaulted to 'issue'), asserting they get reclassified from their
// number label and filename — the fix for libraries scanned before 0016.
func TestBackfillBookKind(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	lib := seedLibrary(t, store)
	series := seedSeries(t, store, lib, "Batman")

	// Insert books as pre-0016 rows: Kind left empty → stored as the default 'issue'.
	mk := func(path, number string, sort float64) string {
		id := ulid.New()
		b := domain.Book{
			ID: id, SeriesID: series, LibraryID: lib, FilePath: path, FileFormat: "cbz",
			Number: number, SortNumber: sort, AddedAt: 1, UpdatedAt: 1,
		}
		if _, err := store.Books().Upsert(ctx, b); err != nil {
			t.Fatalf("upsert %s: %v", path, err)
		}
		return id
	}
	issue := mk(`C:\DC\Batman\Batman 001.cbz`, "1", 1)
	annual := mk(`C:\DC\Batman\Batman Annual 02.cbz`, "Annual 2", 1_000_002)
	oneShot := mk(`C:\DC\Batman\Batman One-Shot.cbz`, "One-Shot", 1_000_000)
	variant := mk(`C:\DC\Batman\Batman 001 (Variant).cbz`, "1", 1)
	varShort := mk(`C:\DC\Batman\Batman 003 var.cbz`, "", 0)
	covers := mk(`C:\DC\Batman\Batman Covers.cbz`, "", 0)
	undercover := mk(`C:\DC\Undercover\Undercover 003.cbz`, "3", 3)

	// Everything starts as 'issue'.
	if got, _ := store.Books().Get(ctx, annual); got.Kind != domain.KindIssue {
		t.Fatalf("precondition: annual kind = %q, want issue", got.Kind)
	}

	// Run the real backfill migration body (idempotent) against these rows.
	body, err := fs.ReadFile(migrationsFS, "migrations/sqlite/0017_backfill_book_kind.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	if _, err := store.db.ExecContext(ctx, string(body)); err != nil {
		t.Fatalf("exec backfill: %v", err)
	}

	want := map[string]domain.BookKind{
		issue:      domain.KindIssue,
		annual:     domain.KindAnnual,
		oneShot:    domain.KindOneShot,
		variant:    domain.KindVariant,
		varShort:   domain.KindVariant,
		covers:     domain.KindCover,
		undercover: domain.KindIssue, // "cover" substring but has a real number → stays an issue
	}
	for id, exp := range want {
		got, err := store.Books().Get(ctx, id)
		if err != nil {
			t.Fatalf("get %s: %v", id, err)
		}
		if got.Kind != exp {
			t.Errorf("%s: kind = %q, want %q", got.FilePath, got.Kind, exp)
		}
	}
}
