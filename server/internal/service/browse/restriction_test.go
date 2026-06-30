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
	"github.com/siposbnc/comic-hub/server/internal/store/sqlite"
)

// TestContentRestrictionFiltersBrowse verifies a restricted user's age ceiling hides
// over-rated issues from listings and refuses their detail, while leaving unrestricted
// users (empty ceiling) untouched.
func TestContentRestrictionFiltersBrowse(t *testing.T) {
	ctx := context.Background()
	dsn := "file:" + filepath.Join(t.TempDir(), "b.db") +
		"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"
	db, err := sqlite.Open(dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	if err := sqlite.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	store := sqlite.NewStore(db)

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

	svc := browse.New(store)
	restricted := access.WithCeiling(ctx, "Teen")

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
