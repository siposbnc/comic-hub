package sqlstore

import (
	"context"
	"testing"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/pkg/ulid"
)

func seedSeries(t *testing.T, store *Store, libID, name string) string {
	t.Helper()
	id := ulid.New()
	_, err := store.Series().Upsert(context.Background(), domain.Series{
		ID: id, LibraryID: libID, Name: name, SortName: name, CreatedAt: 1, UpdatedAt: 1,
	})
	if err != nil {
		t.Fatalf("seed series %q: %v", name, err)
	}
	return id
}

func seedTitledBook(t *testing.T, store *Store, libID, seriesID, path, number, title string) string {
	t.Helper()
	id := ulid.New()
	_, err := store.Books().Upsert(context.Background(), domain.Book{
		ID: id, SeriesID: seriesID, LibraryID: libID, FilePath: path, FileFormat: "cbz",
		FileSize: 1, FileMTime: 1, Number: number, Title: title, AddedAt: 1, UpdatedAt: 1,
	})
	if err != nil {
		t.Fatalf("seed book %q: %v", title, err)
	}
	return id
}

func seriesIDs(hits []domain.SeriesSearchHit) map[string]bool {
	m := make(map[string]bool, len(hits))
	for _, h := range hits {
		m[h.ID] = true
	}
	return m
}

func TestSearchSeriesAndBooks(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	lib1 := seedLibrary(t, store)
	lib2 := seedLibrary(t, store)

	batman := seedSeries(t, store, lib1, "Batman")
	seedSeries(t, store, lib1, "Superman")
	beyond := seedSeries(t, store, lib2, "Batman Beyond")
	owls := seedTitledBook(t, store, lib1, batman, `C:\c\b1.cbz`, "1", "The Court of Owls")
	seedTitledBook(t, store, lib1, batman, `C:\c\b2.cbz`, "2", "") // untitled — won't match text

	repo := store.Search()

	// Prefix series search, scoped to one library (excludes the lib2 "Batman Beyond").
	hits, err := repo.SearchSeries(ctx, lib1, "bat*", 10)
	if err != nil {
		t.Fatalf("search series: %v", err)
	}
	ids := seriesIDs(hits)
	if !ids[batman] || ids[beyond] || len(hits) != 1 {
		t.Fatalf("scoped series search = %+v", hits)
	}
	if hits[0].LibraryName != "DC" {
		t.Fatalf("series hit library name = %q, want DC", hits[0].LibraryName)
	}

	// Unscoped spans both libraries.
	hits, _ = repo.SearchSeries(ctx, "", "bat*", 10)
	ids = seriesIDs(hits)
	if !ids[batman] || !ids[beyond] {
		t.Fatalf("unscoped series search = %+v", hits)
	}

	// Book title search carries the series name.
	books, err := repo.SearchBooks(ctx, lib1, "court*", 10)
	if err != nil {
		t.Fatalf("search books: %v", err)
	}
	if len(books) != 1 || books[0].ID != owls || books[0].SeriesName != "Batman" ||
		books[0].Number != "1" || books[0].Format != "cbz" || books[0].LibraryName != "DC" {
		t.Fatalf("book search = %+v", books)
	}
}

func TestSearchTriggersStayInSync(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	lib := seedLibrary(t, store)
	id := seedSeries(t, store, lib, "Batman")
	repo := store.Search()

	// Rename via Upsert (UPDATE) — the AFTER UPDATE trigger reindexes.
	if _, err := store.Series().Upsert(ctx, domain.Series{
		ID: id, LibraryID: lib, Name: "Batwoman", SortName: "Batwoman", CreatedAt: 1, UpdatedAt: 2,
	}); err != nil {
		t.Fatalf("rename: %v", err)
	}
	if hits, _ := repo.SearchSeries(ctx, lib, "batwoman*", 10); len(hits) != 1 {
		t.Fatalf("renamed series not found: %+v", hits)
	}
	if hits, _ := repo.SearchSeries(ctx, lib, "batman*", 10); len(hits) != 0 {
		t.Fatalf("stale name still indexed: %+v", hits)
	}

	// Deleting the library cascades to series + book; the DELETE triggers clear the index.
	seedTitledBook(t, store, lib, id, `C:\c\x.cbz`, "1", "Elegy")
	if err := store.Libraries().Delete(ctx, lib); err != nil {
		t.Fatalf("delete library: %v", err)
	}
	if hits, _ := repo.SearchSeries(ctx, "", "batwoman*", 10); len(hits) != 0 {
		t.Fatalf("series index not cleared on cascade delete: %+v", hits)
	}
	if books, _ := repo.SearchBooks(ctx, "", "elegy*", 10); len(books) != 0 {
		t.Fatalf("book index not cleared on cascade delete: %+v", books)
	}
}
