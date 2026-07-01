package browse_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/siposbnc/comic-hub/server/internal/access"
	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/pkg/ulid"
	"github.com/siposbnc/comic-hub/server/internal/service/browse"
	"github.com/siposbnc/comic-hub/server/internal/store/sqlstore"
)

// TestContentRestrictionFiltersBrowse verifies a restricted user's age ceiling hides
// over-rated issues from listings and refuses their detail, while leaving unrestricted
// users (empty ceiling) untouched.
func TestContentRestrictionFiltersBrowse(t *testing.T) {
	ctx := context.Background()
	dsn := "file:" + filepath.Join(t.TempDir(), "b.db") +
		"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"
	db, err := sqlstore.OpenSQLite(dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	if err := sqlstore.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	store := sqlstore.NewStore(db)

	lib, err := store.Libraries().Create(ctx, domain.Library{
		ID: ulid.New(), Name: "DC", Kind: "comic", Roots: []string{`C:\DC`}, CreatedAt: 1, UpdatedAt: 1,
	})
	if err != nil {
		t.Fatalf("library: %v", err)
	}
	ser, err := store.Series().Upsert(ctx, domain.Series{
		ID: ulid.New(), LibraryID: lib.ID, Name: "Batman", SortName: "Batman", CreatedAt: 1, UpdatedAt: 1,
	})
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	mkBook := func(num, rating, path string) string {
		b, err := store.Books().Upsert(ctx, domain.Book{
			ID: ulid.New(), SeriesID: ser.ID, LibraryID: lib.ID, FilePath: path,
			FileFormat: "cbz", PageCount: 10, Number: num, AgeRating: rating, AddedAt: 1, UpdatedAt: 1,
		})
		if err != nil {
			t.Fatalf("book %s: %v", num, err)
		}
		return b.ID
	}
	teenID := mkBook("1", "Teen", `C:\DC\Batman\1.cbz`)
	adultID := mkBook("2", "Adults Only 18+", `C:\DC\Batman\2.cbz`)

	// A second, entirely adult-rated series — must be invisible to the restricted user.
	adultSeries, _ := store.Series().Upsert(ctx, domain.Series{
		ID: ulid.New(), LibraryID: lib.ID, Name: "Hellblazer", SortName: "Hellblazer", CreatedAt: 1, UpdatedAt: 1,
	})
	if _, err := store.Books().Upsert(ctx, domain.Book{
		ID: ulid.New(), SeriesID: adultSeries.ID, LibraryID: lib.ID, FilePath: `C:\DC\Hellblazer\1.cbz`,
		FileFormat: "cbz", PageCount: 10, Number: "1", AgeRating: "Adults Only 18+", AddedAt: 1, UpdatedAt: 1,
	}); err != nil {
		t.Fatalf("adult series book: %v", err)
	}

	svc := browse.New(store)
	restricted := access.WithCeiling(ctx, "Teen")

	// Restricted listing: the all-adult series is gone; the partial one reports only its
	// visible issue and a visible cover.
	list, err := svc.ListSeries(restricted, lib.ID, "owner")
	if err != nil {
		t.Fatalf("list series: %v", err)
	}
	if len(list) != 1 || list[0].ID != ser.ID {
		t.Fatalf("restricted listing = %+v, want only Batman", list)
	}
	if list[0].BookCount != 1 || list[0].CoverBookID != teenID {
		t.Fatalf("restricted Batman card = count %d cover %q, want 1 / teen cover", list[0].BookCount, list[0].CoverBookID)
	}
	// Unrestricted listing shows both series.
	if full, _ := svc.ListSeries(ctx, lib.ID, "owner"); len(full) != 2 {
		t.Fatalf("unrestricted listing = %d series, want 2", len(full))
	}

	// Restricted user: series shows only the Teen issue.
	d, err := svc.SeriesDetail(restricted, ser.ID, "owner")
	if err != nil {
		t.Fatalf("series detail: %v", err)
	}
	if d.BookCount != 1 || len(d.Books) != 1 || d.Books[0].ID != teenID {
		t.Fatalf("restricted series = %d books %+v, want only the Teen issue", d.BookCount, d.Books)
	}
	// The adult issue's detail is hidden (404, not 403 — don't reveal it exists).
	if _, err := svc.BookDetail(restricted, adultID, "owner"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("restricted adult detail = %v, want ErrNotFound", err)
	}
	// The teen issue is readable.
	if _, err := svc.BookDetail(restricted, teenID, "owner"); err != nil {
		t.Fatalf("restricted teen detail: %v", err)
	}

	// Unrestricted user (no ceiling): sees both issues.
	d2, err := svc.SeriesDetail(ctx, ser.ID, "owner")
	if err != nil {
		t.Fatalf("unrestricted series detail: %v", err)
	}
	if d2.BookCount != 2 {
		t.Fatalf("unrestricted series = %d books, want 2", d2.BookCount)
	}
	if _, err := svc.BookDetail(ctx, adultID, "owner"); err != nil {
		t.Fatalf("unrestricted adult detail: %v", err)
	}
}
