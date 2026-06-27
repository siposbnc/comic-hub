package metadata

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/pkg/ulid"
	"github.com/siposbnc/comic-hub/server/internal/providers"
	"github.com/siposbnc/comic-hub/server/internal/store/sqlite"
)

// fakeProvider returns canned data so the apply pipeline can be tested without network.
type fakeProvider struct{}

func (fakeProvider) Name() string { return "fake" }

func (fakeProvider) SearchSeries(context.Context, string) ([]providers.SeriesCandidate, error) {
	return []providers.SeriesCandidate{
		{ProviderID: "vol-1", Name: "Wonder Woman", Year: 2016, IssueCount: 2},
		{ProviderID: "vol-9", Name: "Wonder Woman Annual", Year: 1988, IssueCount: 1},
	}, nil
}

func (fakeProvider) Issues(_ context.Context, vol string) ([]providers.IssueCandidate, error) {
	if vol != "vol-1" {
		return nil, nil
	}
	return []providers.IssueCandidate{
		{ProviderID: "iss-1", Number: "1"},
		{ProviderID: "iss-2", Number: "2"},
	}, nil
}

func (fakeProvider) Issue(_ context.Context, id string) (providers.IssueMeta, error) {
	switch id {
	case "iss-1":
		return providers.IssueMeta{
			Title: "The Lies Part One", Number: "1", Summary: "begins",
			People: map[string][]string{"writer": {"Greg Rucka"}}, Characters: []string{"Wonder Woman"},
		}, nil
	case "iss-2":
		return providers.IssueMeta{Title: "Year One Part One", Number: "2", Summary: "flashback"}, nil
	}
	return providers.IssueMeta{}, nil
}

func newStore(t *testing.T) *sqlite.Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "t.db")
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)", dbPath)
	db, err := sqlite.Open(dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := sqlite.Migrate(context.Background(), db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return sqlite.NewStore(db)
}

// seed creates a library + series + two books (#1, #2) and returns their ids.
func seed(t *testing.T, store *sqlite.Store) (seriesID, book1, book2 string) {
	t.Helper()
	ctx := context.Background()
	libID := ulid.New()
	if _, err := store.Libraries().Create(ctx, domain.Library{ID: libID, Name: "DC", Kind: "comic", Roots: []string{`C:\DC`}, CreatedAt: 1, UpdatedAt: 1}); err != nil {
		t.Fatal(err)
	}
	seriesID = ulid.New()
	if _, err := store.Series().Upsert(ctx, domain.Series{ID: seriesID, LibraryID: libID, Name: "Wonder Woman", SortName: "Wonder Woman", Year: 2016, CreatedAt: 1, UpdatedAt: 1}); err != nil {
		t.Fatal(err)
	}
	mk := func(num string) string {
		id := ulid.New()
		b := domain.Book{ID: id, SeriesID: seriesID, LibraryID: libID, FilePath: `C:\DC\` + id + `.cbz`, FileFormat: "cbz", FileSize: 1, FileMTime: 1, Number: num, AddedAt: 1, UpdatedAt: 1}
		if _, err := store.Books().Upsert(ctx, b); err != nil {
			t.Fatal(err)
		}
		return id
	}
	return seriesID, mk("1"), mk("2")
}

func TestCandidatesRanked(t *testing.T) {
	store := newStore(t)
	seriesID, _, _ := seed(t, store)
	svc := New(store, fakeProvider{})

	cands, err := svc.Candidates(context.Background(), seriesID, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(cands) == 0 || cands[0].ProviderID != "vol-1" {
		t.Fatalf("best candidate = %+v, want vol-1 first", cands)
	}
}

func TestMatchSeries(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	seriesID, book1, book2 := seed(t, store)
	svc := New(store, fakeProvider{})

	var lastDone, lastTotal int
	if err := svc.MatchSeries(ctx, seriesID, "", "vol-1", nil, func(done, total int) {
		lastDone, lastTotal = done, total
	}); err != nil {
		t.Fatalf("MatchSeries: %v", err)
	}
	if lastDone != 2 || lastTotal != 2 {
		t.Fatalf("progress final = %d/%d, want 2/2", lastDone, lastTotal)
	}

	b1, _ := store.Books().Get(ctx, book1)
	if b1.Title != "The Lies Part One" || b1.Summary != "begins" || b1.MetadataState != domain.MetaMatched {
		t.Fatalf("book1 not matched: %+v", b1)
	}
	if credits, _ := store.Metadata().BookCredits(ctx, book1); len(credits["writer"]) != 1 || credits["writer"][0] != "Greg Rucka" {
		t.Fatalf("book1 credits = %v", credits)
	}
	if chars, _ := store.Metadata().BookCharacters(ctx, book1); len(chars) != 1 || chars[0] != "Wonder Woman" {
		t.Fatalf("book1 characters = %v", chars)
	}
	if ids, _ := store.Metadata().BookProviderIDs(ctx, book1); ids["fake"] != "iss-1" {
		t.Fatalf("book1 provider link = %v", ids)
	}

	b2, _ := store.Books().Get(ctx, book2)
	if b2.Title != "Year One Part One" {
		t.Fatalf("book2 title = %q", b2.Title)
	}
}

func TestApplyBookRespectsLock(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	_, book1, _ := seed(t, store)
	svc := New(store, fakeProvider{})

	// Pin the title; the match must keep it but is free to fill the summary.
	if err := store.Metadata().WriteBookMeta(ctx, book1, domain.BookMeta{
		Title: "PINNED", Number: "1", State: domain.MetaLocked, LockedFields: []string{FieldTitle},
	}); err != nil {
		t.Fatal(err)
	}

	if err := svc.ApplyBook(ctx, book1, "", "iss-1", nil); err != nil {
		t.Fatalf("ApplyBook: %v", err)
	}

	b1, _ := store.Books().Get(ctx, book1)
	if b1.Title != "PINNED" {
		t.Fatalf("locked title overwritten: %q", b1.Title)
	}
	if b1.Summary != "begins" {
		t.Fatalf("unlocked summary not applied: %q", b1.Summary)
	}
	if locked, _ := store.Metadata().LockedBookFields(ctx, book1); len(locked) != 1 || locked[0] != FieldTitle {
		t.Fatalf("lock not preserved: %v", locked)
	}
}
