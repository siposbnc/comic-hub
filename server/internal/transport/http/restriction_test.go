package http

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/siposbnc/comic-hub/server/internal/config"
	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/pkg/ulid"
	"github.com/siposbnc/comic-hub/server/internal/service/auth"
	"github.com/siposbnc/comic-hub/server/internal/store/sqlite"
)

// TestContentRestrictionBlocksReader verifies the reader content routes refuse (403) a
// restricted user a book rated above their ceiling — the server-side security boundary.
func TestContentRestrictionBlocksReader(t *testing.T) {
	ctx := context.Background()
	dsn := "file:" + filepath.Join(t.TempDir(), "r.db") +
		"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"
	db, err := sqlite.Open(dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := sqlite.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	store := sqlite.NewStore(db)

	lib, _ := store.Libraries().Create(ctx, domain.Library{
		ID: ulid.New(), Name: "DC", Kind: "comic", Roots: []string{`C:\DC`}, CreatedAt: 1, UpdatedAt: 1,
	})
	ser, _ := store.Series().Upsert(ctx, domain.Series{
		ID: ulid.New(), LibraryID: lib.ID, Name: "Batman", SortName: "Batman", CreatedAt: 1, UpdatedAt: 1,
	})
	adult, _ := store.Books().Upsert(ctx, domain.Book{
		ID: ulid.New(), SeriesID: ser.ID, LibraryID: lib.ID, FilePath: `C:\DC\a.cbz`,
		FileFormat: "cbz", PageCount: 10, Number: "1", AgeRating: "Adults Only 18+", AddedAt: 1, UpdatedAt: 1,
	})

	authSvc := auth.New(store, []byte("secret"))
	if _, err := authSvc.CreateUser(ctx, auth.CreateUserInput{
		Username: "kid", DisplayName: "Kid", Role: domain.RoleRestricted, Password: "kidsecret1", AgeRatingMax: "Teen",
	}); err != nil {
		t.Fatalf("create restricted: %v", err)
	}

	router := NewRouter(Deps{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		DB:     db,
		Config: config.Config{Mode: config.ModeServer, AuthEnabled: true},
		Repo:   store,
		Auth:   authSvc,
	})
	srv := httptest.NewServer(router)
	t.Cleanup(func() { srv.Close(); _ = db.Close() })
	api := srv.URL + "/api/v1"

	access, _ := loginToken(t, api, "kid", "kidsecret1")

	for _, path := range []string{
		"/books/" + adult.ID + "/manifest",
		"/books/" + adult.ID + "/cover",
		"/books/" + adult.ID + "/pages/0",
	} {
		r := authReq(t, http.MethodGet, api+path, access, "")
		r.Body.Close()
		if r.StatusCode != http.StatusForbidden {
			t.Fatalf("restricted GET %s = %d, want 403", path, r.StatusCode)
		}
	}
}
