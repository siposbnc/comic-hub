package scanner

import (
	"archive/zip"
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/siposbnc/comic-hub/server/internal/archive"
	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/pkg/ulid"
	"github.com/siposbnc/comic-hub/server/internal/store/sqlstore"
)

func newStore(t *testing.T) *sqlstore.Store {
	t.Helper()
	dsn := "file:" + filepath.Join(t.TempDir(), "scan.db") +
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

func writeCBZ(t *testing.T, path string, entries map[string]string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	for name, data := range entries {
		w, _ := zw.Create(name)
		_, _ = io.WriteString(w, data)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
}

func seriesByName(t *testing.T, store *sqlstore.Store, libID, name string) domain.Series {
	t.Helper()
	all, err := store.Series().ListByLibrary(context.Background(), libID)
	if err != nil {
		t.Fatalf("list series: %v", err)
	}
	for _, s := range all {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("series %q not found (have %d)", name, len(all))
	return domain.Series{}
}

func TestScanLibrary(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	root := t.TempDir()

	// Saga: two issues, the first with a ComicInfo.xml sidecar + page types.
	writeCBZ(t, filepath.Join(root, "Saga", "Saga 001 (2012).cbz"), map[string]string{
		"p1.jpg": "a", "p2.jpg": "b",
		"ComicInfo.xml": `<ComicInfo><Series>Saga</Series><Number>1</Number><Pages><Page Image="0" Type="FrontCover"/></Pages></ComicInfo>`,
	})
	writeCBZ(t, filepath.Join(root, "Saga", "Saga 002 (2012).cbz"), map[string]string{
		"p1.jpg": "a", "p2.jpg": "b", "p3.jpg": "c",
	})
	// Batman: one valid issue (no sidecar) + one corrupt .cbr.
	writeCBZ(t, filepath.Join(root, "Batman", "Batman 001.cbz"), map[string]string{"p1.jpg": "x"})
	if err := os.WriteFile(filepath.Join(root, "Batman", "broken.cbr"), []byte("not a rar"), 0o644); err != nil {
		t.Fatalf("write corrupt: %v", err)
	}

	lib := domain.Library{ID: ulid.New(), Name: "Lib", Kind: "comic", Roots: []string{root}, CreatedAt: 1, UpdatedAt: 1}
	if _, err := store.Libraries().Create(ctx, lib); err != nil {
		t.Fatalf("create lib: %v", err)
	}

	sc := New(store, archive.DefaultRegistry(), slog.New(slog.NewTextHandler(io.Discard, nil)), 0)
	if err := sc.Scan(ctx, lib.ID, true, nil); err != nil {
		t.Fatalf("scan: %v", err)
	}

	// Two series grouped by folder.
	allSeries, _ := store.Series().ListByLibrary(ctx, lib.ID)
	if len(allSeries) != 2 {
		t.Fatalf("expected 2 series, got %d", len(allSeries))
	}

	// Saga issue 1: sidecar metadata + page types.
	saga := seriesByName(t, store, lib.ID, "Saga")
	books, _ := store.Books().ListBySeries(ctx, saga.ID)
	if len(books) != 2 {
		t.Fatalf("expected 2 Saga books, got %d", len(books))
	}
	b1 := books[0] // ordered by sort_number, so issue 1 first
	if b1.Number != "1" || b1.MetadataState != domain.MetaSidecar || b1.PageCount != 2 {
		t.Fatalf("Saga #1 wrong: %+v", b1)
	}
	if b1.ContentHash == "" {
		t.Error("expected a content hash")
	}
	pages, err := store.Books().ListPages(ctx, b1.ID)
	if err != nil {
		t.Fatalf("list pages: %v", err)
	}
	if len(pages) != 2 || pages[0].PageType != "FrontCover" {
		t.Fatalf("Saga #1 pages wrong: %+v", pages)
	}

	// Batman: corrupt .cbr flagged, valid issue intact.
	batman := seriesByName(t, store, lib.ID, "Batman")
	bbooks, _ := store.Books().ListBySeries(ctx, batman.ID)
	if len(bbooks) != 2 {
		t.Fatalf("expected 2 Batman books (1 ok, 1 corrupt), got %d", len(bbooks))
	}
	var corrupt, ok int
	for _, b := range bbooks {
		if b.IsCorrupt {
			corrupt++
			if b.PageCount != 0 {
				t.Errorf("corrupt book should have 0 pages, got %d", b.PageCount)
			}
		} else {
			ok++
		}
	}
	if corrupt != 1 || ok != 1 {
		t.Fatalf("expected 1 corrupt + 1 ok, got corrupt=%d ok=%d", corrupt, ok)
	}
}

func TestScanGroupsSubfoldersUnderOneSeries(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	root := t.TempDir()

	// A series folder whose issues live in subfolders (variant covers, a volume, annuals).
	writeCBZ(t, filepath.Join(root, "Nova Tide", "Nova Tide 001.cbz"), map[string]string{"p.jpg": "a"})
	writeCBZ(t, filepath.Join(root, "Nova Tide", "Variant Covers", "Nova Tide 001 var.cbz"), map[string]string{"p.jpg": "b"})
	writeCBZ(t, filepath.Join(root, "Nova Tide", "Vol. 2", "Nova Tide 007.cbz"), map[string]string{"p.jpg": "c"})
	writeCBZ(t, filepath.Join(root, "Nova Tide", "Annuals", "Nova Tide Annual 1.cbz"), map[string]string{"p.jpg": "d"})

	lib := domain.Library{ID: ulid.New(), Name: "L", Kind: "comic", Roots: []string{root}, CreatedAt: 1, UpdatedAt: 1}
	if _, err := store.Libraries().Create(ctx, lib); err != nil {
		t.Fatalf("create lib: %v", err)
	}
	sc := New(store, archive.DefaultRegistry(), slog.New(slog.NewTextHandler(io.Discard, nil)), 0)
	if err := sc.Scan(ctx, lib.ID, true, nil); err != nil {
		t.Fatalf("scan: %v", err)
	}

	// One series, not four — subfolders don't fragment it.
	allSeries, _ := store.Series().ListByLibrary(ctx, lib.ID)
	if len(allSeries) != 1 || allSeries[0].Name != "Nova Tide" {
		t.Fatalf("expected 1 'Nova Tide' series, got %+v", allSeries)
	}
	books, _ := store.Books().ListBySeries(ctx, allSeries[0].ID)
	if len(books) != 4 {
		t.Fatalf("expected all 4 issues under the series, got %d", len(books))
	}
}

func TestScanReconcilesMovedFile(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	root := t.TempDir()
	orig := filepath.Join(root, "Saga", "Saga 001.cbz")
	writeCBZ(t, orig, map[string]string{"p1.jpg": "a", "p2.jpg": "b"})

	lib := domain.Library{ID: ulid.New(), Name: "L", Kind: "comic", Roots: []string{root}, CreatedAt: 1, UpdatedAt: 1}
	_, _ = store.Libraries().Create(ctx, lib)
	sc := New(store, archive.DefaultRegistry(), slog.New(slog.NewTextHandler(io.Discard, nil)), 0)

	if err := sc.Scan(ctx, lib.ID, true, nil); err != nil {
		t.Fatalf("scan 1: %v", err)
	}
	saga := seriesByName(t, store, lib.ID, "Saga")
	before, _ := store.Books().ListBySeries(ctx, saga.ID)
	if len(before) != 1 {
		t.Fatalf("expected 1 book, got %d", len(before))
	}
	origID := before[0].ID

	// Move the file to a different series folder (rename on disk), then rescan.
	moved := filepath.Join(root, "Image", "Saga 001.cbz")
	if err := os.MkdirAll(filepath.Dir(moved), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Rename(orig, moved); err != nil {
		t.Fatalf("rename: %v", err)
	}
	if err := sc.Scan(ctx, lib.ID, false, nil); err != nil {
		t.Fatalf("scan 2: %v", err)
	}

	// The same row should now point at the new path — one book total, id preserved.
	all, _ := store.Books().ListByLibrary(ctx, lib.ID)
	if len(all) != 1 {
		t.Fatalf("expected 1 book after move (no duplicate/orphan), got %d", len(all))
	}
	if all[0].ID != origID {
		t.Fatalf("move created a new row: id %s != %s", all[0].ID, origID)
	}
	if all[0].FilePath != moved {
		t.Fatalf("path not updated: %s", all[0].FilePath)
	}
}

// TestFullRescanRepairsStaleAndPreservesMatched models a library scanned under an older
// parser (a stale wrong number in the catalog, files unchanged on disk) that has since
// been provider-matched. A full rescan must re-parse the stale book, but must not wipe
// the matched series header or matched/locked book fields.
func TestFullRescanRepairsStaleAndPreservesMatched(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	root := t.TempDir()

	writeCBZ(t, filepath.Join(root, "Wonder Woman", "Wonder Woman 023.1 - Cheetah 001 (2013).cbz"),
		map[string]string{"p1.jpg": "a"})
	writeCBZ(t, filepath.Join(root, "Wonder Woman", "Wonder Woman 024 (2013).cbz"),
		map[string]string{"p1.jpg": "b"})

	lib := domain.Library{ID: ulid.New(), Name: "L", Kind: "comic", Roots: []string{root}, CreatedAt: 1, UpdatedAt: 1}
	_, _ = store.Libraries().Create(ctx, lib)
	sc := New(store, archive.DefaultRegistry(), slog.New(slog.NewTextHandler(io.Discard, nil)), 0)
	if err := sc.Scan(ctx, lib.ID, true, nil); err != nil {
		t.Fatalf("scan 1: %v", err)
	}

	ww := seriesByName(t, store, lib.ID, "Wonder Woman")
	books, _ := store.Books().ListBySeries(ctx, ww.ID)
	if len(books) != 2 {
		t.Fatalf("expected 2 books, got %d", len(books))
	}

	// Simulate the pre-fix catalog: the point-one issue carries a stale wrong number.
	stale := books[0]
	stale.Number = "1"
	stale.SortNumber = 1
	if _, err := store.Books().Upsert(ctx, stale); err != nil {
		t.Fatalf("stale upsert: %v", err)
	}
	// Provider-match the series header and the other book.
	if err := store.Series().WriteMatch(ctx, ww.ID, domain.SeriesMatch{
		Publisher: "DC Comics", Year: 2011, Description: "The Amazon warrior.",
		State: domain.MetaMatched, Provider: "comicvine", ProviderID: "v1",
	}); err != nil {
		t.Fatalf("series match: %v", err)
	}
	matched := books[1]
	matched.Title = "Dark Gods Part 1"
	matched.Summary = "From the provider."
	matched.MetadataState = domain.MetaMatched
	if _, err := store.Books().Upsert(ctx, matched); err != nil {
		t.Fatalf("matched upsert: %v", err)
	}

	// Files on disk never changed; a full rescan re-parses everything.
	if err := sc.Scan(ctx, lib.ID, true, nil); err != nil {
		t.Fatalf("scan 2: %v", err)
	}

	ww2, _ := store.Series().Get(ctx, ww.ID)
	if ww2.Publisher != "DC Comics" || ww2.Year != 2011 || ww2.Description != "The Amazon warrior." {
		t.Fatalf("full rescan wiped matched series header: %+v", ww2)
	}
	after, _ := store.Books().ListBySeries(ctx, ww.ID)
	byID := make(map[string]domain.Book, len(after))
	for _, b := range after {
		byID[b.ID] = b
	}
	if got := byID[stale.ID]; got.Number != "23.1" || got.SortNumber != 23.1 {
		t.Fatalf("stale number not repaired: %+v", got)
	}
	if got := byID[matched.ID]; got.Title != "Dark Gods Part 1" || got.Summary != "From the provider." ||
		got.MetadataState != domain.MetaMatched {
		t.Fatalf("full rescan clobbered matched book fields: %+v", got)
	}
}

func TestScanIncrementalIdempotent(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	root := t.TempDir()
	writeCBZ(t, filepath.Join(root, "Saga", "Saga 001.cbz"), map[string]string{"p1.jpg": "a"})

	lib := domain.Library{ID: ulid.New(), Name: "L", Kind: "comic", Roots: []string{root}, CreatedAt: 1, UpdatedAt: 1}
	_, _ = store.Libraries().Create(ctx, lib)
	sc := New(store, archive.DefaultRegistry(), slog.New(slog.NewTextHandler(io.Discard, nil)), 0)

	if err := sc.Scan(ctx, lib.ID, true, nil); err != nil {
		t.Fatalf("scan 1: %v", err)
	}
	saga := seriesByName(t, store, lib.ID, "Saga")
	first, _ := store.Books().ListBySeries(ctx, saga.ID)
	firstID := first[0].ID

	// Incremental rescan: unchanged file -> same row (same id), no duplicates.
	if err := sc.Scan(ctx, lib.ID, false, nil); err != nil {
		t.Fatalf("scan 2: %v", err)
	}
	again, _ := store.Books().ListBySeries(ctx, saga.ID)
	if len(again) != 1 || again[0].ID != firstID {
		t.Fatalf("incremental rescan changed catalog: %+v", again)
	}
}
