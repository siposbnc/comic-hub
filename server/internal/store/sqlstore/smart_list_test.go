package sqlstore

import (
	"context"
	"sort"
	"testing"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/pkg/ulid"
)

func seedFormatBook(t *testing.T, store *Store, libID, seriesID, path, number, format string) string {
	t.Helper()
	id := ulid.New()
	_, err := store.Books().Upsert(context.Background(), domain.Book{
		ID: id, SeriesID: seriesID, LibraryID: libID, FilePath: path, FileFormat: format,
		FileSize: 1, FileMTime: 1, Number: number, AddedAt: 1, UpdatedAt: 1,
	})
	if err != nil {
		t.Fatalf("seed book: %v", err)
	}
	return id
}

func TestSmartListEvaluate(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	lib := seedLibrary(t, store)
	batman := seedSeries(t, store, lib, "Batman")
	saga := seedSeries(t, store, lib, "Saga")
	b1 := seedFormatBook(t, store, lib, batman, `C:\c\1.cbz`, "1", "cbz")
	b2 := seedFormatBook(t, store, lib, batman, `C:\c\2.cbz`, "2", "cbz")
	b3 := seedFormatBook(t, store, lib, saga, `C:\c\3.cbr`, "1", "cbr")

	// Tag "Fav" on b1 + b3.
	tag := domain.Tag{ID: ulid.New(), Name: "Fav"}
	if _, err := store.Tags().Create(ctx, tag); err != nil {
		t.Fatalf("tag: %v", err)
	}
	if err := store.Tags().AssignToBook(ctx, b1, []string{tag.ID}); err != nil {
		t.Fatalf("assign: %v", err)
	}
	if err := store.Tags().AssignToBook(ctx, b3, []string{tag.ID}); err != nil {
		t.Fatalf("assign: %v", err)
	}

	// b1 is read by the owner.
	if _, err := store.Progress().Upsert(ctx, domain.Progress{
		UserID: "owner", BookID: b1, Page: 9, PageCount: 10, Status: domain.StatusRead, UpdatedAt: 1,
	}); err != nil {
		t.Fatalf("progress: %v", err)
	}

	repo := store.SmartLists()
	rule := func(f, op, v string) domain.SmartRule { return domain.SmartRule{Field: f, Op: op, Value: v} }
	eval := func(match string, rules ...domain.SmartRule) []string {
		ids, err := repo.Evaluate(ctx, domain.SmartRules{Match: match, Rules: rules}, "owner", 0)
		if err != nil {
			t.Fatalf("evaluate: %v", err)
		}
		return ids
	}

	cases := []struct {
		name string
		got  []string
		want []string
	}{
		{"tag is", eval("all", rule("tag", "is", tag.ID)), []string{b1, b3}},
		{"tag isNot", eval("all", rule("tag", "isNot", tag.ID)), []string{b2}},
		{"series is", eval("all", rule("series", "is", "Batman")), []string{b1, b2}},
		{"format is", eval("all", rule("format", "is", "cbr")), []string{b3}},
		{"readStatus read", eval("all", rule("readStatus", "is", "read")), []string{b1}},
		{"readStatus unread", eval("all", rule("readStatus", "is", "unread")), []string{b2, b3}},
		{"match all", eval("all", rule("tag", "is", tag.ID), rule("series", "is", "Batman")), []string{b1}},
		{"match any", eval("any", rule("series", "is", "Saga"), rule("format", "is", "cbz")), []string{b1, b2, b3}},
		{"series contains", eval("all", rule("series", "contains", "bat")), []string{b1, b2}},
	}
	for _, c := range cases {
		if !sameSet(c.got, c.want) {
			t.Errorf("%s = %v, want set %v", c.name, c.got, c.want)
		}
	}

	// Count agrees with Evaluate.
	n, err := repo.Count(ctx, domain.SmartRules{Match: "all", Rules: []domain.SmartRule{rule("tag", "is", tag.ID)}}, "owner")
	if err != nil || n != 2 {
		t.Fatalf("count = %d, err %v, want 2", n, err)
	}
}

func TestSmartListCRUD(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	repo := store.SmartLists()

	in := domain.SmartList{
		ID: ulid.New(), OwnerID: "owner", Name: "Unread Manga",
		Rules: domain.SmartRules{
			Match: "all",
			Rules: []domain.SmartRule{{Field: "readStatus", Op: "is", Value: "unread"}},
		},
		CreatedAt: 1, UpdatedAt: 1,
	}
	if _, err := repo.Create(ctx, in); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := repo.Get(ctx, in.ID)
	if err != nil || got.Name != "Unread Manga" || len(got.Rules.Rules) != 1 ||
		got.Rules.Rules[0].Value != "unread" {
		t.Fatalf("get round-trip = %+v, err %v", got, err)
	}

	got.Name = "Renamed"
	got.UpdatedAt = 2
	if err := repo.Update(ctx, got); err != nil {
		t.Fatalf("update: %v", err)
	}
	if after, _ := repo.Get(ctx, in.ID); after.Name != "Renamed" {
		t.Fatalf("after update = %+v", after)
	}
	if err := repo.Delete(ctx, in.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
}

func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	x, y := append([]string(nil), a...), append([]string(nil), b...)
	sort.Strings(x)
	sort.Strings(y)
	for i := range x {
		if x[i] != y[i] {
			return false
		}
	}
	return true
}
