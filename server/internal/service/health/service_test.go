package health

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/pkg/ulid"
	"github.com/siposbnc/comic-hub/server/internal/store/sqlstore"
)

func newStore(t *testing.T) *sqlstore.Store {
	t.Helper()
	dsn := "file:" + filepath.Join(t.TempDir(), "h.db") +
		"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"
	db, err := sqlstore.OpenSQLite(dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := sqlstore.Migrate(context.Background(), db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return sqlstore.NewStore(db)
}

func TestHealthReport(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)

	lib := domain.Library{ID: ulid.New(), Name: "DC", Kind: "comic", Roots: []string{`C:\DC`}, CreatedAt: 1, UpdatedAt: 1}
	if _, err := store.Libraries().Create(ctx, lib); err != nil {
		t.Fatalf("library: %v", err)
	}
	series := domain.Series{ID: ulid.New(), LibraryID: lib.ID, Name: "Batman", SortName: "Batman", CreatedAt: 1, UpdatedAt: 1}
	if _, err := store.Series().Upsert(ctx, series); err != nil {
		t.Fatalf("series: %v", err)
	}

	book := func(path, hash string, state domain.MetadataState, corrupt bool) string {
		id := ulid.New()
		_, err := store.Books().Upsert(ctx, domain.Book{
			ID: id, SeriesID: series.ID, LibraryID: lib.ID, FilePath: path, FileFormat: "cbz",
			FileSize: 1, FileMTime: 1, ContentHash: hash, MetadataState: state, IsCorrupt: corrupt,
			AddedAt: 1, UpdatedAt: 1,
		})
		if err != nil {
			t.Fatalf("book: %v", err)
		}
		return id
	}

	book(`C:\DC\1.cbz`, "h1", domain.MetaMatched, false)    // healthy, dup of b3
	book(`C:\DC\2.cbz`, "h2", domain.MetaMatched, true)     // corrupt
	book(`C:\DC\3.cbz`, "h1", domain.MetaNone, false)       // unmatched + dup of b1
	book(`C:\DC\gone.cbz`, "h3", domain.MetaMatched, false) // orphan (missing on disk)

	// Everything exists except the orphan path.
	present := map[string]bool{`C:\DC\1.cbz`: true, `C:\DC\2.cbz`: true, `C:\DC\3.cbz`: true}
	svc := New(store, WithExistsFunc(func(p string) bool { return present[p] }))

	rep, err := svc.Report(ctx, lib.ID)
	if err != nil {
		t.Fatalf("report: %v", err)
	}
	if rep.Counts.Books != 4 {
		t.Fatalf("books = %d, want 4", rep.Counts.Books)
	}
	if rep.Counts.Corrupt != 1 || len(rep.Corrupt) != 1 {
		t.Fatalf("corrupt = %d/%d, want 1", rep.Counts.Corrupt, len(rep.Corrupt))
	}
	if rep.Counts.Unmatched != 1 || len(rep.Unmatched) != 1 {
		t.Fatalf("unmatched = %d, want 1", rep.Counts.Unmatched)
	}
	if rep.Counts.Orphans != 1 || rep.Orphans[0].Path != `C:\DC\gone.cbz` {
		t.Fatalf("orphans = %+v, want the gone path", rep.Orphans)
	}
	if rep.Counts.DuplicateGroups != 1 || len(rep.Duplicates) != 1 || len(rep.Duplicates[0].Books) != 2 {
		t.Fatalf("duplicates = %+v, want one group of 2", rep.Duplicates)
	}
}

func TestHealthMissingLibrary(t *testing.T) {
	store := newStore(t)
	svc := New(store)
	if _, err := svc.Report(context.Background(), "nope"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing library err = %v, want ErrNotFound", err)
	}
}
