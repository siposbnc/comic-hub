package sqlstore

import (
	"context"
	"errors"
	"testing"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/pkg/ulid"
)

func TestTagRoundTrip(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	lib := seedLibrary(t, store)
	series := seedSeries(t, store, lib, "Saga")
	b1 := seedTitledBook(t, store, lib, series, `C:\c\1.cbz`, "1", "One")
	b2 := seedTitledBook(t, store, lib, series, `C:\c\2.cbz`, "2", "Two")
	repo := store.Tags()

	tag := domain.Tag{ID: ulid.New(), Name: "Favorites", Color: "#ff0066"}
	if _, err := repo.Create(ctx, tag); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Name lookup is case-insensitive.
	if got, err := repo.GetByName(ctx, "favorites"); err != nil || got.ID != tag.ID {
		t.Fatalf("get by name = %+v, err %v", got, err)
	}
	if _, err := repo.GetByName(ctx, "nope"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing name = %v, want ErrNotFound", err)
	}

	// Assign to both books (with a duplicate that must be ignored) → count + listings.
	if err := repo.AssignToBook(ctx, b1, []string{tag.ID, tag.ID}); err != nil {
		t.Fatalf("assign b1: %v", err)
	}
	if err := repo.AssignToBook(ctx, b2, []string{tag.ID}); err != nil {
		t.Fatalf("assign b2: %v", err)
	}
	if got, _ := repo.Get(ctx, tag.ID); got.BookCount != 2 {
		t.Fatalf("book count = %d, want 2", got.BookCount)
	}
	if tags, _ := repo.BookTags(ctx, b1); len(tags) != 1 || tags[0].Name != "Favorites" || tags[0].Color != "#ff0066" {
		t.Fatalf("book tags = %+v", tags)
	}
	ids, _ := repo.TaggedBookIDs(ctx, tag.ID)
	if len(ids) != 2 {
		t.Fatalf("tagged books = %v", ids)
	}

	// Rename + recolor.
	tag.Name = "Top Picks"
	tag.Color = ""
	if err := repo.Update(ctx, tag); err != nil {
		t.Fatalf("update: %v", err)
	}
	if got, _ := repo.Get(ctx, tag.ID); got.Name != "Top Picks" || got.Color != "" {
		t.Fatalf("after update = %+v", got)
	}

	// Unassign one, then delete the tag (book_tag cascades).
	if err := repo.UnassignFromBook(ctx, b1, tag.ID); err != nil {
		t.Fatalf("unassign: %v", err)
	}
	if got, _ := repo.Get(ctx, tag.ID); got.BookCount != 1 {
		t.Fatalf("count after unassign = %d, want 1", got.BookCount)
	}
	if err := repo.Delete(ctx, tag.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if tags, _ := repo.BookTags(ctx, b2); len(tags) != 0 {
		t.Fatalf("tags after delete = %+v, want none", tags)
	}
}
