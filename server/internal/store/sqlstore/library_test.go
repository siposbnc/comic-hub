package sqlstore

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/pkg/ulid"
)

// newTestStore opens a fresh migrated database in a temp dir and returns a Store.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	dsn := "file:" + filepath.Join(t.TempDir(), "test.db") +
		"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"
	db, err := OpenSQLite(dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := Migrate(context.Background(), db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return NewStore(db)
}

func TestLibraryRoundTrip(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	repo := store.Libraries()

	in := domain.Library{
		ID:        ulid.New(),
		Name:      "DC",
		Kind:      "comic",
		Roots:     []string{`C:\Comics\DC`, `D:\More\DC`},
		CreatedAt: 1000,
		UpdatedAt: 1000,
	}
	if _, err := repo.Create(ctx, in); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.Get(ctx, in.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "DC" || got.Kind != "comic" {
		t.Fatalf("get returned %+v", got)
	}
	if len(got.Roots) != 2 {
		t.Fatalf("expected 2 roots, got %v", got.Roots)
	}

	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 library, got %d", len(list))
	}

	if err := repo.Delete(ctx, in.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := repo.Get(ctx, in.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestLibraryGetMissing(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.Libraries().Get(context.Background(), "nope"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestLibraryDeleteMissing(t *testing.T) {
	store := newTestStore(t)
	if err := store.Libraries().Delete(context.Background(), "nope"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// TestOwnerSeeded verifies migration 0002 inserts the implicit owner the handshake
// identity and progress rows depend on.
func TestOwnerSeeded(t *testing.T) {
	store := newTestStore(t)
	var role string
	err := store.db.QueryRowContext(context.Background(),
		`SELECT role FROM "user" WHERE id = 'owner'`).Scan(&role)
	if err != nil {
		t.Fatalf("query owner: %v", err)
	}
	if role != "owner" {
		t.Fatalf("owner role = %q, want owner", role)
	}
}
