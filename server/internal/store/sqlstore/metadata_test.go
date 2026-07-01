package sqlstore

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/pkg/ulid"
)

// seedBook creates a series + book under a fresh library and returns the book id.
func seedBook(t *testing.T, store *Store) string {
	t.Helper()
	ctx := context.Background()
	libID := seedLibrary(t, store)
	s := domain.Series{ID: ulid.New(), LibraryID: libID, Name: "Wonder Woman", SortName: "Wonder Woman", CreatedAt: 1, UpdatedAt: 1}
	if _, err := store.Series().Upsert(ctx, s); err != nil {
		t.Fatalf("series upsert: %v", err)
	}
	bookID := ulid.New()
	b := domain.Book{
		ID: bookID, SeriesID: s.ID, LibraryID: libID,
		FilePath: `C:\DC\WW\` + bookID + `.cbz`, FileFormat: "cbz", FileSize: 1, FileMTime: 1,
		Number: "1", SortNumber: 1, AddedAt: 1, UpdatedAt: 1,
	}
	if _, err := store.Books().Upsert(ctx, b); err != nil {
		t.Fatalf("book upsert: %v", err)
	}
	return b.ID
}

func TestWriteBookMeta(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	meta := store.Metadata()
	bookID := seedBook(t, store)

	in := domain.BookMeta{
		Title: "The Lies Part One", Number: "1", Volume: 5, ReleaseDate: 1470009600000,
		AgeRating: "Teen", Language: "en", Summary: "A new era begins.",
		State: domain.MetaMatched, ProviderIDs: map[string]string{"comicvine": "1001"},
		LockedFields: []string{"summary", "title"},
	}
	if err := meta.WriteBookMeta(ctx, bookID, in); err != nil {
		t.Fatalf("WriteBookMeta: %v", err)
	}

	got, err := store.Books().Get(ctx, bookID)
	if err != nil {
		t.Fatalf("get book: %v", err)
	}
	if got.Title != in.Title || got.Summary != in.Summary || got.Volume != 5 ||
		got.ReleaseDate != in.ReleaseDate || got.MetadataState != domain.MetaMatched {
		t.Fatalf("scalar metadata not written: %+v", got)
	}

	if ids, _ := meta.BookProviderIDs(ctx, bookID); ids["comicvine"] != "1001" {
		t.Fatalf("provider ids = %v", ids)
	}
	if locked, _ := meta.LockedBookFields(ctx, bookID); !reflect.DeepEqual(locked, []string{"summary", "title"}) {
		t.Fatalf("locked fields = %v", locked)
	}
}

func TestWriteBookMetaNotFound(t *testing.T) {
	store := newTestStore(t)
	err := store.Metadata().WriteBookMeta(context.Background(), "no-such-book", domain.BookMeta{})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestReplaceCreditsGenresCharacters(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	meta := store.Metadata()
	bookID := seedBook(t, store)

	credits := map[string][]string{"writer": {"Greg Rucka"}, "penciler": {"Liam Sharp"}}
	if err := meta.ReplaceBookPeople(ctx, bookID, credits); err != nil {
		t.Fatalf("ReplaceBookPeople: %v", err)
	}
	if got, _ := meta.BookCredits(ctx, bookID); !reflect.DeepEqual(got, credits) {
		t.Fatalf("credits = %v, want %v", got, credits)
	}

	if err := meta.ReplaceBookGenres(ctx, bookID, []string{"Superhero", "Action"}); err != nil {
		t.Fatalf("ReplaceBookGenres: %v", err)
	}
	if got, _ := meta.BookGenres(ctx, bookID); !reflect.DeepEqual(got, []string{"Action", "Superhero"}) {
		t.Fatalf("genres = %v (want sorted)", got)
	}
	// Replace must swap, not append.
	if err := meta.ReplaceBookGenres(ctx, bookID, []string{"Action"}); err != nil {
		t.Fatalf("ReplaceBookGenres 2: %v", err)
	}
	if got, _ := meta.BookGenres(ctx, bookID); !reflect.DeepEqual(got, []string{"Action"}) {
		t.Fatalf("genres after replace = %v, want [Action]", got)
	}

	if err := meta.ReplaceBookCharacters(ctx, bookID, []string{"Wonder Woman", "Steve Trevor"}); err != nil {
		t.Fatalf("ReplaceBookCharacters: %v", err)
	}
	if got, _ := meta.BookCharacters(ctx, bookID); !reflect.DeepEqual(got, []string{"Steve Trevor", "Wonder Woman"}) {
		t.Fatalf("characters = %v (want sorted)", got)
	}

	// Crediting the same person on a second book must reuse the person row, not error.
	other := seedBook(t, store)
	if err := meta.ReplaceBookPeople(ctx, other, map[string][]string{"writer": {"Greg Rucka"}}); err != nil {
		t.Fatalf("reuse person: %v", err)
	}
	if got, _ := meta.BookCredits(ctx, other); !reflect.DeepEqual(got, map[string][]string{"writer": {"Greg Rucka"}}) {
		t.Fatalf("second book credits = %v", got)
	}
}
