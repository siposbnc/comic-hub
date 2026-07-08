package sqlstore

import (
	"context"
	"testing"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/pkg/ulid"
)

// seedLibrary creates a library and returns its id, for tests that need FK parents.
func seedLibrary(t *testing.T, store *Store) string {
	t.Helper()
	lib := domain.Library{ID: ulid.New(), Name: "DC", Kind: "comic", Roots: []string{`C:\DC`}, CreatedAt: 1, UpdatedAt: 1}
	if _, err := store.Libraries().Create(context.Background(), lib); err != nil {
		t.Fatalf("seed library: %v", err)
	}
	return lib.ID
}

// TestCatalogRoundTrip exercises series + book + page + progress with a mix of set
// and absent (NULL) optional fields, verifying the nullable scan paths.
func TestCatalogRoundTrip(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	libID := seedLibrary(t, store)

	// Series with most optional fields left empty (stored NULL).
	s := domain.Series{
		ID:        ulid.New(),
		LibraryID: libID,
		Name:      "Batman",
		SortName:  "Batman",
		CreatedAt: 1, UpdatedAt: 1,
	}
	if _, err := store.Series().Upsert(ctx, s); err != nil {
		t.Fatalf("series upsert: %v", err)
	}
	gotSeries, err := store.Series().Get(ctx, s.ID)
	if err != nil {
		t.Fatalf("series get: %v", err)
	}
	if gotSeries.ReadingDir != domain.LTR {
		t.Fatalf("reading_dir = %q, want default ltr", gotSeries.ReadingDir)
	}

	// Book with optional metadata populated.
	b := domain.Book{
		ID:         ulid.New(),
		SeriesID:   s.ID,
		LibraryID:  libID,
		FilePath:   `C:\DC\Batman\Batman 001.cbz`,
		FileFormat: "cbz",
		FileSize:   1234,
		FileMTime:  10,
		PageCount:  22,
		Number:     "1",
		SortNumber: 1.0,
		AddedAt:    1, UpdatedAt: 1,
	}
	if _, err := store.Books().Upsert(ctx, b); err != nil {
		t.Fatalf("book upsert: %v", err)
	}
	gotBook, err := store.Books().Get(ctx, b.ID)
	if err != nil {
		t.Fatalf("book get: %v", err)
	}
	if gotBook.Number != "1" || gotBook.SortNumber != 1.0 || gotBook.MetadataState != domain.MetaNone {
		t.Fatalf("book round-trip mismatch: %+v", gotBook)
	}

	pages := []domain.Page{
		{BookID: b.ID, Index: 0, FileName: "000.jpg", Width: 988, Height: 1500, PageType: "FrontCover"},
		{BookID: b.ID, Index: 1, FileName: "001.jpg"}, // dims absent -> NULL
	}
	if err := store.Books().ReplacePages(ctx, b.ID, pages); err != nil {
		t.Fatalf("replace pages: %v", err)
	}

	// Progress upsert + continue-reading filter.
	p := domain.Progress{
		UserID: "owner", BookID: b.ID, Page: 5, PageCount: 22,
		Status: domain.StatusInProgress, UpdatedAt: 100,
	}
	if _, err := store.Progress().Upsert(ctx, p); err != nil {
		t.Fatalf("progress upsert: %v", err)
	}
	cont, err := store.Progress().ContinueReading(ctx, "owner", 10)
	if err != nil {
		t.Fatalf("continue reading: %v", err)
	}
	if len(cont) != 1 || cont[0].BookID != b.ID || cont[0].Page != 5 {
		t.Fatalf("continue reading returned %+v", cont)
	}
}

// TestSetPageDimensions backfills dimensions onto a page that was scanned without them,
// and verifies untouched pages (and other fields of the touched page) are preserved.
func TestSetPageDimensions(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	libID := seedLibrary(t, store)

	s := domain.Series{ID: ulid.New(), LibraryID: libID, Name: "Batman", SortName: "Batman", CreatedAt: 1, UpdatedAt: 1}
	if _, err := store.Series().Upsert(ctx, s); err != nil {
		t.Fatalf("series upsert: %v", err)
	}
	b := domain.Book{
		ID: ulid.New(), SeriesID: s.ID, LibraryID: libID,
		FilePath: `C:\DC\Batman\Batman 001.cbz`, FileFormat: "cbz", PageCount: 2,
		AddedAt: 1, UpdatedAt: 1,
	}
	if _, err := store.Books().Upsert(ctx, b); err != nil {
		t.Fatalf("book upsert: %v", err)
	}
	pages := []domain.Page{
		{BookID: b.ID, Index: 0, FileName: "000.jpg", Width: 988, Height: 1500, PageType: "FrontCover"},
		{BookID: b.ID, Index: 1, FileName: "001.jpg", IsDouble: true}, // dims absent -> NULL
	}
	if err := store.Books().ReplacePages(ctx, b.ID, pages); err != nil {
		t.Fatalf("replace pages: %v", err)
	}

	// Backfill only the second page.
	if err := store.Books().SetPageDimensions(ctx, b.ID, []domain.PageDimension{{Index: 1, Width: 2000, Height: 1500}}); err != nil {
		t.Fatalf("set page dimensions: %v", err)
	}

	got, err := store.Books().ListPages(ctx, b.ID)
	if err != nil {
		t.Fatalf("list pages: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d pages, want 2", len(got))
	}
	// Page 0 untouched.
	if got[0].Width != 988 || got[0].Height != 1500 || got[0].PageType != "FrontCover" {
		t.Fatalf("page 0 changed: %+v", got[0])
	}
	// Page 1 dims filled, other fields intact.
	if got[1].Width != 2000 || got[1].Height != 1500 {
		t.Fatalf("page 1 dims = %dx%d, want 2000x1500", got[1].Width, got[1].Height)
	}
	if got[1].FileName != "001.jpg" || !got[1].IsDouble {
		t.Fatalf("page 1 non-dim fields clobbered: %+v", got[1])
	}
}
