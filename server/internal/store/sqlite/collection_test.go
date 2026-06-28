package sqlite

import (
	"context"
	"errors"
	"testing"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/pkg/ulid"
)

func TestCollectionRoundTrip(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	lib := seedLibrary(t, store)
	series := seedSeries(t, store, lib, "Batman")
	b1 := seedTitledBook(t, store, lib, series, `C:\c\1.cbz`, "1", "One")
	b2 := seedTitledBook(t, store, lib, series, `C:\c\2.cbz`, "2", "Two")
	b3 := seedTitledBook(t, store, lib, series, `C:\c\3.cbz`, "3", "Three")
	repo := store.Collections()

	c := domain.Collection{ID: ulid.New(), Name: "Favorites", OwnerID: "owner", CreatedAt: 1, UpdatedAt: 1}
	if _, err := repo.Create(ctx, c); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.Get(ctx, c.ID)
	if err != nil || got.Name != "Favorites" || got.BookCount != 0 {
		t.Fatalf("get = %+v, err %v", got, err)
	}

	// Add items (with a duplicate that must be ignored), then verify order + count.
	if err := repo.AddItems(ctx, c.ID, []string{b1, b2, b1}); err != nil {
		t.Fatalf("add items: %v", err)
	}
	if err := repo.AddItems(ctx, c.ID, []string{b3}); err != nil {
		t.Fatalf("add items 2: %v", err)
	}
	items, _ := repo.Items(ctx, c.ID)
	if got := itemIDs(items); !equal(got, []string{b1, b2, b3}) {
		t.Fatalf("items order = %v", got)
	}
	if got, _ := repo.Get(ctx, c.ID); got.BookCount != 3 {
		t.Fatalf("book count = %d, want 3", got.BookCount)
	}

	// Reposition b3 before b1 (position bisects below b1).
	if err := repo.SetPosition(ctx, c.ID, b3, items[0].Position-1); err != nil {
		t.Fatalf("set position: %v", err)
	}
	items, _ = repo.Items(ctx, c.ID)
	if got := itemIDs(items); !equal(got, []string{b3, b1, b2}) {
		t.Fatalf("reordered = %v", got)
	}

	// Remove + cascade-on-delete.
	if err := repo.RemoveItem(ctx, c.ID, b1); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if items, _ = repo.Items(ctx, c.ID); !equal(itemIDs(items), []string{b3, b2}) {
		t.Fatalf("after remove = %v", itemIDs(items))
	}
	if err := repo.Delete(ctx, c.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := repo.Get(ctx, c.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("get after delete err = %v, want ErrNotFound", err)
	}
}

func TestCollectionMissingErrors(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	repo := store.Collections()
	if err := repo.Delete(ctx, "nope"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("delete missing = %v", err)
	}
	if err := repo.RemoveItem(ctx, "nope", "x"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("remove missing = %v", err)
	}
}

func itemIDs(items []domain.CollectionItem) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.BookID
	}
	return out
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
