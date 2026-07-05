package sqlstore

import (
	"context"
	"errors"
	"testing"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/pkg/ulid"
)

func TestReadingListRoundTrip(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	lib := seedLibrary(t, store)
	series := seedSeries(t, store, lib, "Saga")
	b1 := seedTitledBook(t, store, lib, series, `C:\c\1.cbz`, "1", "One")
	b2 := seedTitledBook(t, store, lib, series, `C:\c\2.cbz`, "2", "Two")
	repo := store.ReadingLists()

	l := domain.ReadingList{ID: ulid.New(), UserID: "owner", Name: "To Read", CreatedAt: 1, UpdatedAt: 1}
	if _, err := repo.Create(ctx, l); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Owner-scoped reads: visible to the owner, invisible to anyone else.
	if got, err := repo.Get(ctx, "owner", l.ID); err != nil || got.Name != "To Read" {
		t.Fatalf("owner get = %+v, err %v", got, err)
	}
	if _, err := repo.Get(ctx, "someone-else", l.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("cross-user get = %v, want ErrNotFound", err)
	}

	if err := repo.AddItems(ctx, l.ID, []string{b1, b2, b1}); err != nil {
		t.Fatalf("add items: %v", err)
	}
	if items, _ := repo.Items(ctx, l.ID); !equal(rlItemIDs(items), []string{b1, b2}) {
		t.Fatalf("items = %v", rlItemIDs(items))
	}
	if got, _ := repo.Get(ctx, "owner", l.ID); got.BookCount != 2 {
		t.Fatalf("book count = %d, want 2", got.BookCount)
	}

	// Reposition b2 ahead of b1.
	items, _ := repo.Items(ctx, l.ID)
	if err := repo.SetPosition(ctx, l.ID, b2, items[0].Position-1); err != nil {
		t.Fatalf("set position: %v", err)
	}
	if items, _ = repo.Items(ctx, l.ID); !equal(rlItemIDs(items), []string{b2, b1}) {
		t.Fatalf("reordered = %v", rlItemIDs(items))
	}

	// A non-owner cannot delete it; the owner can.
	if err := repo.Delete(ctx, "someone-else", l.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("cross-user delete = %v, want ErrNotFound", err)
	}
	if err := repo.Delete(ctx, "owner", l.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := repo.Get(ctx, "owner", l.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("get after delete = %v, want ErrNotFound", err)
	}
}

func rlItemIDs(items []domain.ReadingListItem) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.BookID
	}
	return out
}

// TestReadingListStaleLifecycle covers the deletion-proofing: entries snapshot their
// display data on add, survive a series delete as stale placeholders (order intact),
// re-attach by content hash when the book returns, and support manual placeholders +
// explicit relinks.
func TestReadingListStaleLifecycle(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	lib := seedLibrary(t, store)
	series := seedSeries(t, store, lib, "Wonder Woman")

	hashedBook := func(seriesID, path, number, title, hash string) string {
		id := ulid.New()
		if _, err := store.Books().Upsert(ctx, domain.Book{
			ID: id, SeriesID: seriesID, LibraryID: lib, FilePath: path, FileFormat: "cbz",
			FileSize: 1, FileMTime: 1, ContentHash: hash, Number: number, Title: title,
			AddedAt: 1, UpdatedAt: 1,
		}); err != nil {
			t.Fatalf("seed book: %v", err)
		}
		return id
	}
	b23 := hashedBook(series, `C:\c\ww23.cbz`, "23", "", "hash-23")
	b231 := hashedBook(series, `C:\c\ww23.1.cbz`, "23.1", "Cheetah", "hash-23.1")

	repo := store.ReadingLists()
	l := domain.ReadingList{ID: ulid.New(), UserID: "owner", Name: "Villain Month", CreatedAt: 1, UpdatedAt: 1}
	if _, err := repo.Create(ctx, l); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := repo.AddItems(ctx, l.ID, []string{b23, b231}); err != nil {
		t.Fatalf("add: %v", err)
	}
	// A manual placeholder for an issue not in the library.
	if err := repo.AddManualItems(ctx, l.ID, []domain.ManualListItem{
		{SeriesName: "Wonder Woman", Number: "23.2", Title: "First Born"},
	}); err != nil {
		t.Fatalf("manual add: %v", err)
	}

	items, _ := repo.Items(ctx, l.ID)
	if len(items) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(items))
	}
	if items[0].SeriesName != "Wonder Woman" || items[0].Number != "23" || items[0].ContentHash != "hash-23" {
		t.Fatalf("snapshot not captured on add: %+v", items[0])
	}
	if !items[2].Stale() || items[2].Number != "23.2" {
		t.Fatalf("manual entry wrong: %+v", items[2])
	}

	// Delete the series (books cascade): linked entries must go stale in place.
	if err := store.Series().Delete(ctx, series); err != nil {
		t.Fatalf("series delete: %v", err)
	}
	items, _ = repo.Items(ctx, l.ID)
	if len(items) != 3 {
		t.Fatalf("series delete changed entry count: %d", len(items))
	}
	for i, it := range items {
		if !it.Stale() {
			t.Fatalf("entry %d still linked after series delete: %+v", i, it)
		}
	}
	if items[1].SeriesName != "Wonder Woman" || items[1].Number != "23.1" || items[1].Title != "Cheetah" {
		t.Fatalf("stale entry lost its snapshot: %+v", items[1])
	}

	// The book "returns" (rescan recreates it, new id, same content): hash relink.
	series2 := seedSeries(t, store, lib, "Wonder Woman")
	nb23 := hashedBook(series2, `C:\c\ww23.cbz`, "23", "", "hash-23")
	if n, err := repo.RelinkStaleByHash(ctx, "hash-23", nb23); err != nil || n != 1 {
		t.Fatalf("relink by hash = %d, %v; want 1", n, err)
	}
	items, _ = repo.Items(ctx, l.ID)
	if items[0].BookID != nb23 || items[0].Stale() {
		t.Fatalf("hash relink failed: %+v", items[0])
	}

	// Explicit relink of the manual placeholder onto a real book.
	nb232 := hashedBook(series2, `C:\c\ww23.2.cbz`, "23.2", "First Born", "hash-23.2")
	if err := repo.Relink(ctx, l.ID, items[2].ID, nb232); err != nil {
		t.Fatalf("relink: %v", err)
	}
	items, _ = repo.Items(ctx, l.ID)
	if items[2].BookID != nb232 || items[2].Title != "First Born" {
		t.Fatalf("explicit relink failed: %+v", items[2])
	}
	// Relinking another entry to the same book must be rejected (already in the list).
	if err := repo.Relink(ctx, l.ID, items[1].ID, nb232); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("duplicate relink = %v, want ErrValidation", err)
	}
}
