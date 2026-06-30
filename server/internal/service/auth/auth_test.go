package auth_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/service/auth"
	"github.com/siposbnc/comic-hub/server/internal/store/sqlite"
)

func newStore(t *testing.T) *sqlite.Store {
	t.Helper()
	dsn := "file:" + filepath.Join(t.TempDir(), "auth.db") +
		"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"
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

func TestLoginRefreshLogout(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	svc := auth.New(store, []byte("test-secret"))

	if err := svc.EnsureAdmin(ctx, "alice", "Alice", "hunter2hunter"); err != nil {
		t.Fatalf("ensure admin: %v", err)
	}

	// Wrong password and unknown user both yield ErrUnauthorized (no oracle).
	if _, _, err := svc.Login(ctx, "alice", "wrong"); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("bad password = %v, want ErrUnauthorized", err)
	}
	if _, _, err := svc.Login(ctx, "nobody", "hunter2hunter"); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("unknown user = %v, want ErrUnauthorized", err)
	}

	toks, user, err := svc.Login(ctx, "alice", "hunter2hunter")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if user.Username != "alice" || user.Role != domain.RoleAdmin {
		t.Fatalf("login user = %+v", user)
	}

	// Access token authenticates to the same user.
	got, err := svc.Authenticate(ctx, toks.Access)
	if err != nil || got.ID != user.ID {
		t.Fatalf("authenticate = %+v, %v", got, err)
	}
	// A tampered token is rejected.
	if _, err := svc.Authenticate(ctx, toks.Access+"x"); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("tampered token = %v, want ErrUnauthorized", err)
	}

	// Refresh rotates: new pair works, the old refresh token is now spent.
	toks2, _, err := svc.Refresh(ctx, toks.Refresh)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if toks2.Refresh == toks.Refresh {
		t.Fatal("refresh token was not rotated")
	}
	if _, _, err := svc.Refresh(ctx, toks.Refresh); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("reused refresh = %v, want ErrUnauthorized", err)
	}

	// Logout revokes the current session; its refresh no longer works.
	if err := svc.Logout(ctx, toks2.Refresh); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, _, err := svc.Refresh(ctx, toks2.Refresh); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("post-logout refresh = %v, want ErrUnauthorized", err)
	}
}

func TestPasswordlessOwnerCannotLogin(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	svc := auth.New(store, []byte("secret"))
	// The seeded owner (0002) has no password hash.
	if _, _, err := svc.Login(ctx, "owner", ""); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("passwordless login = %v, want ErrUnauthorized", err)
	}
}

func TestEnsureAdminSetsOwnerPassword(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	svc := auth.New(store, []byte("secret"))

	// Bootstrapping the existing "owner" username sets its password (doesn't duplicate).
	if err := svc.EnsureAdmin(ctx, "owner", "", "ownerpass123"); err != nil {
		t.Fatalf("ensure owner: %v", err)
	}
	if _, u, err := svc.Login(ctx, "owner", "ownerpass123"); err != nil || u.Role != domain.RoleOwner {
		t.Fatalf("owner login after bootstrap = %+v, %v", u, err)
	}
	users, _ := store.Users().List(ctx)
	if len(users) != 1 {
		t.Fatalf("got %d users, want 1 (owner updated in place)", len(users))
	}

	// A short password is rejected.
	if err := svc.EnsureAdmin(ctx, "bob", "", "short"); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("short password = %v, want ErrValidation", err)
	}
}
