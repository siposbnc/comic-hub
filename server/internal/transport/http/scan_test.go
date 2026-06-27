package http

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/siposbnc/comic-hub/server/internal/archive"
	"github.com/siposbnc/comic-hub/server/internal/config"
	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/image"
	"github.com/siposbnc/comic-hub/server/internal/jobs"
	"github.com/siposbnc/comic-hub/server/internal/scanner"
	"github.com/siposbnc/comic-hub/server/internal/service/library"
	"github.com/siposbnc/comic-hub/server/internal/service/reader"
	"github.com/siposbnc/comic-hub/server/internal/store/sqlite"
)

// newScanServer wires the full scan stack (store + runner + scanner) behind the router.
func newScanServer(t *testing.T) (string, *sqlite.Store) {
	t.Helper()
	dsn := "file:" + filepath.Join(t.TempDir(), "scan.db") +
		"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"
	db, err := sqlite.Open(dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := sqlite.Migrate(context.Background(), db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	store := sqlite.NewStore(db)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	registry := archive.DefaultRegistry()

	derived, err := image.NewDiskCache(filepath.Join(t.TempDir(), "derived"))
	if err != nil {
		t.Fatalf("derived cache: %v", err)
	}
	readerSvc, err := reader.New(store, registry, image.New(), derived)
	if err != nil {
		t.Fatalf("reader svc: %v", err)
	}

	runner := jobs.NewRunner(store, logger, 2)
	sc := scanner.New(store, registry, logger, 0)
	runner.Register(domain.JobScan, func(ctx context.Context, payload string, progress jobs.ProgressFunc) error {
		var p scanner.JobPayload
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			return err
		}
		return sc.Scan(ctx, p.LibraryID, p.Full, scanner.ProgressFunc(progress))
	})

	router := NewRouter(Deps{
		Logger:  logger,
		DB:      db,
		Config:  config.Config{Mode: config.ModeServer},
		Library: library.New(store),
		Repo:    store,
		Runner:  runner,
		Reader:  readerSvc,
	})
	srv := httptest.NewServer(router)
	t.Cleanup(func() { srv.Close(); runner.Shutdown(); _ = db.Close() })
	return srv.URL, store
}

func TestScanEndpointToCatalog(t *testing.T) {
	srv, store := newScanServer(t)
	api := srv + "/api/v1"

	// A library root with one CBZ.
	root := t.TempDir()
	writeTestCBZ(t, filepath.Join(root, "Saga", "Saga 001 (2012).cbz"), map[string]string{
		"p1.jpg": "a", "p2.jpg": "b",
	})

	// Create the library.
	body, _ := json.Marshal(map[string]any{"name": "DC", "kind": "comic", "roots": []string{root}})
	res, err := http.Post(api+"/libraries", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create lib: %v", err)
	}
	var created libraryDTO
	decode(t, res, &created)

	// Start a full scan.
	res, err = http.Post(api+"/libraries/"+created.ID+"/scan", "application/json", bytes.NewBufferString(`{"mode":"full"}`))
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("scan status = %d, want 202", res.StatusCode)
	}
	var started struct {
		JobID string `json:"jobId"`
	}
	decode(t, res, &started)
	if started.JobID == "" {
		t.Fatal("no jobId returned")
	}

	// Poll the job to completion.
	waitJobDone(t, api, started.JobID)

	// Catalog should now have the series + book.
	series, _ := store.Series().ListByLibrary(context.Background(), created.ID)
	if len(series) != 1 || series[0].Name != "Saga" {
		t.Fatalf("series = %+v", series)
	}
	books, _ := store.Books().ListBySeries(context.Background(), series[0].ID)
	if len(books) != 1 || books[0].PageCount != 2 {
		t.Fatalf("books = %+v", books)
	}
}

func waitJobDone(t *testing.T, api, jobID string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		res, err := http.Get(api + "/jobs/" + jobID)
		if err != nil {
			t.Fatalf("get job: %v", err)
		}
		var j jobDTO
		decode(t, res, &j)
		switch j.State {
		case string(domain.JobDone):
			return
		case string(domain.JobFailed), string(domain.JobCanceled):
			t.Fatalf("job ended in state %q: %s", j.State, j.Error)
		}
		time.Sleep(15 * time.Millisecond)
	}
	t.Fatal("scan job did not finish in time")
}

func writeTestCBZ(t *testing.T, path string, entries map[string]string) {
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
