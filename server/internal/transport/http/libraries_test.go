package http

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/siposbnc/comic-hub/server/internal/config"
	"github.com/siposbnc/comic-hub/server/internal/service/library"
	"github.com/siposbnc/comic-hub/server/internal/store/sqlstore"
)

// newTestServer wires a real migrated store + library service behind the router, with
// auth disabled (empty token), and returns an httptest server.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	dsn := "file:" + filepath.Join(t.TempDir(), "test.db") +
		"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"
	db, err := sqlstore.OpenSQLite(dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := sqlstore.Migrate(context.Background(), db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	store := sqlstore.NewStore(db)

	router := NewRouter(Deps{
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		DB:      db.Unwrap(),
		Config:  config.Config{Mode: config.ModeServer}, // Token empty -> auth disabled
		Library: library.New(store),
	})
	srv := httptest.NewServer(router)
	t.Cleanup(func() { srv.Close(); _ = db.Close() })
	return srv
}

func TestLibrariesAPILifecycle(t *testing.T) {
	srv := newTestServer(t)
	base := srv.URL + "/api/v1/libraries"

	// Create.
	body := `{"name":"DC","kind":"comic","roots":["./testdata/dc","./testdata/dc"]}`
	res, err := http.Post(base, "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", res.StatusCode)
	}
	var created libraryDTO
	decode(t, res, &created)
	if created.ID == "" || created.Name != "DC" {
		t.Fatalf("created = %+v", created)
	}
	if len(created.Roots) != 1 { // duplicate root de-duplicated
		t.Fatalf("expected 1 de-duplicated root, got %v", created.Roots)
	}

	// Get.
	res, err = http.Get(base + "/" + created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("get status = %d, want 200", res.StatusCode)
	}

	// List.
	res, err = http.Get(base)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var list struct {
		Items []libraryDTO `json:"items"`
	}
	decode(t, res, &list)
	if len(list.Items) != 1 {
		t.Fatalf("expected 1 library, got %d", len(list.Items))
	}

	// Delete.
	req, _ := http.NewRequest(http.MethodDelete, base+"/"+created.ID, nil)
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", res.StatusCode)
	}

	// Get after delete -> 404.
	res, err = http.Get(base + "/" + created.ID)
	if err != nil {
		t.Fatalf("get after delete: %v", err)
	}
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("get-after-delete status = %d, want 404", res.StatusCode)
	}
}

func TestCreateLibraryValidation(t *testing.T) {
	srv := newTestServer(t)
	base := srv.URL + "/api/v1/libraries"

	cases := map[string]string{
		"empty name": `{"name":"","roots":["./x"]}`,
		"no roots":   `{"name":"DC","roots":[]}`,
		"bad kind":   `{"name":"DC","kind":"audio","roots":["./x"]}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			res, err := http.Post(base, "application/json", bytes.NewBufferString(body))
			if err != nil {
				t.Fatalf("post: %v", err)
			}
			if res.StatusCode != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", res.StatusCode)
			}
		})
	}
}

func decode(t *testing.T, res *http.Response, dst any) {
	t.Helper()
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(dst); err != nil {
		t.Fatalf("decode: %v", err)
	}
}
