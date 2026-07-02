package sqlstore

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/pkg/ulid"
)

func TestRebind(t *testing.T) {
	cases := []struct{ in, want string }{
		{"SELECT * FROM book WHERE id = ?", "SELECT * FROM book WHERE id = $1"},
		{"INSERT INTO t (a, b) VALUES (?, ?)", "INSERT INTO t (a, b) VALUES ($1, $2)"},
		{`SELECT '?' , x FROM t WHERE y = ?`, `SELECT '?' , x FROM t WHERE y = $1`},
		{`SELECT "col?" FROM t WHERE y = ? AND z = ?`, `SELECT "col?" FROM t WHERE y = $1 AND z = $2`},
		{"no placeholders", "no placeholders"},
	}
	for _, c := range cases {
		if got := rebind(DriverPostgres, c.in); got != c.want {
			t.Errorf("rebind(%q) = %q, want %q", c.in, got, c.want)
		}
		if got := rebind(DriverSQLite, c.in); got != c.in {
			t.Errorf("sqlite rebind should be identity, got %q", got)
		}
	}
}

// TestMigrationParity: both dialect directories must carry exactly the same numbered
// migrations — a schema change that lands in one but not the other is a bug.
func TestMigrationParity(t *testing.T) {
	lite, err := migrationFiles("migrations/sqlite")
	if err != nil {
		t.Fatalf("sqlite migrations: %v", err)
	}
	pg, err := migrationFiles("migrations/postgres")
	if err != nil {
		t.Fatalf("postgres migrations: %v", err)
	}
	if len(lite) != len(pg) {
		t.Fatalf("migration count mismatch: sqlite %d, postgres %d", len(lite), len(pg))
	}
	for i := range lite {
		if lite[i].version != pg[i].version || lite[i].name != pg[i].name {
			t.Errorf("migration %d: sqlite %q vs postgres %q", i, lite[i].filename, pg[i].filename)
		}
	}
}

// openPG returns a store on the Postgres server named by COMICHUB_TEST_PG_DSN, skipping
// when unset. Run one with:
//
//	docker run --rm -d -p 5433:5432 -e POSTGRES_PASSWORD=test postgres:17
//	COMICHUB_TEST_PG_DSN='postgres://postgres:test@127.0.0.1:5433/postgres?sslmode=disable' go test ./internal/store/sqlstore/
//
// Each run migrates into a fresh schema so tests are isolated and repeatable.
func openPG(t *testing.T) *DB {
	t.Helper()
	dsn := os.Getenv("COMICHUB_TEST_PG_DSN")
	if dsn == "" {
		t.Skip("COMICHUB_TEST_PG_DSN not set; skipping Postgres integration test")
	}
	db, err := Open(DriverPostgres, dsn)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	schema := "t" + ulid.New()
	if _, err := db.ExecContext(t.Context(), `CREATE SCHEMA "`+schema+`"`); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), `DROP SCHEMA "`+schema+`" CASCADE`)
	})
	if _, err := db.ExecContext(t.Context(), `SET search_path TO "`+schema+`"`); err != nil {
		t.Fatalf("set search_path: %v", err)
	}
	// Pin every pooled connection to the schema (search_path is per-connection).
	db.Unwrap().SetMaxOpenConns(1)

	if err := Migrate(t.Context(), db); err != nil {
		t.Fatalf("migrate postgres: %v", err)
	}
	return db
}

// TestPostgresEndToEnd exercises a representative slice of every dialect-sensitive
// path on a real Postgres: migrations, upserts (ON CONFLICT), quoted-identifier user
// CRUD, link tables with get-or-create, LWW progress, and full-text search.
func TestPostgresEndToEnd(t *testing.T) {
	db := openPG(t)
	ctx := t.Context()
	store := NewStore(db)
	now := time.Now().UnixMilli()

	// Library → series → book (scanner shape, including the ON CONFLICT upsert path).
	lib, err := store.Libraries().Create(ctx, domain.Library{
		ID: ulid.New(), Name: "DC", Kind: "comic", Roots: []string{"/comics"}, CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("create library: %v", err)
	}
	ser, err := store.Series().Upsert(ctx, domain.Series{
		ID: ulid.New(), LibraryID: lib.ID, FolderPath: "/comics/Batman", Name: "Batman",
		SortName: "Batman", CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("upsert series: %v", err)
	}
	book, err := store.Books().Upsert(ctx, domain.Book{
		ID: ulid.New(), SeriesID: ser.ID, LibraryID: lib.ID, FilePath: "/comics/Batman/001.cbz",
		FileFormat: "cbz", FileSize: 42, ContentHash: "hash-1", PageCount: 20,
		Title: "The Court of Owls", Number: "1", SortNumber: 1, AddedAt: now, UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("upsert book: %v", err)
	}
	// Re-upsert (same id, as the scanner does after GetByPath): converges, no duplicate.
	if _, err := store.Books().Upsert(ctx, domain.Book{
		ID: book.ID, SeriesID: ser.ID, LibraryID: lib.ID, FilePath: "/comics/Batman/001.cbz",
		FileFormat: "cbz", FileSize: 43, ContentHash: "hash-1", PageCount: 20,
		Title: "The Court of Owls", Number: "1", SortNumber: 1, AddedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("re-upsert book: %v", err)
	}
	books, err := store.Books().ListBySeries(ctx, ser.ID)
	if err != nil || len(books) != 1 {
		t.Fatalf("ListBySeries = %d books (%v), want 1", len(books), err)
	}
	byHash, err := store.Books().ByContentHash(ctx, lib.ID, "hash-1")
	if err != nil || len(byHash) != 1 {
		t.Fatalf("ByContentHash = %d (%v), want 1", len(byHash), err)
	}

	// Quoted "user" table: CRUD + the seeded owner.
	if _, err := store.Users().Get(ctx, domain.OwnerUserID); err != nil {
		t.Fatalf("seeded owner missing: %v", err)
	}
	u, err := store.Users().Create(ctx, domain.User{
		ID: ulid.New(), Username: "bruce", DisplayName: "Bruce Wayne",
		Role: domain.RoleMember, CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if _, err := store.Users().GetByUsername(ctx, "bruce"); err != nil {
		t.Fatalf("get by username: %v", err)
	}

	// Link tables through get-or-create (quoted "character", ON CONFLICT DO NOTHING).
	if err := store.Metadata().ReplaceBookCharacters(ctx, book.ID, []string{"Batman", "Alfred", "Batman"}); err != nil {
		t.Fatalf("replace characters: %v", err)
	}
	chars, err := store.Metadata().BookCharacters(ctx, book.ID)
	if err != nil || len(chars) != 2 {
		t.Fatalf("BookCharacters = %v (%v), want 2 distinct", chars, err)
	}
	if err := store.Metadata().ReplaceBookPeople(ctx, book.ID, map[string][]string{"writer": {"Scott Snyder"}}); err != nil {
		t.Fatalf("replace people: %v", err)
	}

	// Progress (LWW upsert on composite PK) for the created user.
	if _, err := store.Progress().Upsert(ctx, domain.Progress{
		UserID: u.ID, BookID: book.ID, Page: 7, PageCount: 20,
		Status: domain.StatusInProgress, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("progress upsert: %v", err)
	}
	p, err := store.Progress().Get(ctx, u.ID, book.ID)
	if err != nil || p.Page != 7 {
		t.Fatalf("progress get = %+v (%v), want page 7", p, err)
	}

	// Full-text search through the tsquery translation (prefix match).
	hits, err := store.Search().SearchSeries(ctx, "", "bat*", 10)
	if err != nil || len(hits) != 1 || hits[0].Name != "Batman" {
		t.Fatalf("SearchSeries(bat*) = %+v (%v), want Batman", hits, err)
	}
	bhits, err := store.Search().SearchBooks(ctx, lib.ID, "court* owl*", 10)
	if err != nil || len(bhits) != 1 {
		t.Fatalf("SearchBooks(court* owl*) = %+v (%v), want 1 hit", bhits, err)
	}

	// Stats aggregates (Milestone G SQL) run on the dialect too.
	if _, err := store.Progress().Upsert(ctx, domain.Progress{
		UserID: u.ID, BookID: book.ID, Page: 19, PageCount: 20,
		Status: domain.StatusRead, StartedAt: now, FinishedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("finish book: %v", err)
	}
	if booksRead, pagesRead, err := store.Stats().ReadCounts(ctx, u.ID); err != nil || booksRead != 1 || pagesRead != 20 {
		t.Fatalf("ReadCounts = %d/%d (%v), want 1/20", booksRead, pagesRead, err)
	}
	if fin, err := store.Stats().RecentlyFinished(ctx, u.ID, 5); err != nil || len(fin) != 1 || fin[0].SeriesName != "Batman" {
		t.Fatalf("RecentlyFinished = %+v (%v), want 1 Batman entry", fin, err)
	}
	if times, err := store.Stats().ActivityTimes(ctx, u.ID); err != nil || len(times) == 0 {
		t.Fatalf("ActivityTimes = %v (%v), want entries", times, err)
	}

	// Settings k/v round-trip.
	if err := store.Settings().Set(ctx, "k", "v"); err != nil {
		t.Fatalf("settings set: %v", err)
	}
	if v, err := store.Settings().Get(ctx, "k"); err != nil || v != "v" {
		t.Fatalf("settings get = %q (%v), want v", v, err)
	}
}
