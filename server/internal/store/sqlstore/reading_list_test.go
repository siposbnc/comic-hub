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
