package http

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func makePNGBytes(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 100, 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func writeImageCBZ(t *testing.T, path string, pages map[string][]byte) {
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
	for name, data := range pages {
		w, _ := zw.Create(name)
		_, _ = w.Write(data)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip: %v", err)
	}
}

func createLibrary(t *testing.T, api, root string) libraryDTO {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"name": "DC", "kind": "comic", "roots": []string{root}})
	res, err := http.Post(api+"/libraries", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create lib: %v", err)
	}
	var lib libraryDTO
	decode(t, res, &lib)
	return lib
}

func startAndAwaitScan(t *testing.T, api, libID string) {
	t.Helper()
	res, err := http.Post(api+"/libraries/"+libID+"/scan", "application/json", bytes.NewBufferString(`{"mode":"full"}`))
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	var started struct {
		JobID string `json:"jobId"`
	}
	decode(t, res, &started)
	waitJobDone(t, api, started.JobID)
}

func TestImageEndpoints(t *testing.T) {
	srv, store := newScanServer(t)
	api := srv + "/api/v1"
	ctx := context.Background()

	root := t.TempDir()
	writeImageCBZ(t, filepath.Join(root, "Saga", "Saga 001.cbz"), map[string][]byte{
		"p1.png": makePNGBytes(100, 150),
		"p2.png": makePNGBytes(80, 120),
	})

	// Create + scan.
	created := createLibrary(t, api, root)
	startAndAwaitScan(t, api, created.ID)

	// Resolve the book id via the store.
	series, _ := store.Series().ListByLibrary(ctx, created.ID)
	if len(series) != 1 {
		t.Fatalf("series = %d", len(series))
	}
	books, _ := store.Books().ListBySeries(ctx, series[0].ID)
	if len(books) != 1 {
		t.Fatalf("books = %d", len(books))
	}
	bookID := books[0].ID

	// Manifest.
	res, _ := http.Get(api + "/books/" + bookID + "/manifest")
	var m struct {
		PageCount int `json:"pageCount"`
		Pages     []struct {
			Idx int `json:"idx"`
		} `json:"pages"`
		ReadingDir string `json:"readingDir"`
	}
	decode(t, res, &m)
	if m.PageCount != 2 || len(m.Pages) != 2 || m.ReadingDir != "ltr" {
		t.Fatalf("manifest = %+v", m)
	}

	// Cover, resized to 50 -> JPEG 50px wide.
	assertImage(t, api+"/books/"+bookID+"/cover?w=50", "image/jpeg", 50)

	// Original page bytes (PNG passthrough, full size).
	assertImage(t, api+"/books/"+bookID+"/pages/0", "image/png", 100)

	// Resized page -> JPEG at requested width.
	assertImage(t, api+"/books/"+bookID+"/pages/0?w=40", "image/jpeg", 40)

	// Thumb -> JPEG (page is 100 wide, so capped at 100).
	assertImage(t, api+"/books/"+bookID+"/pages/1/thumb", "image/jpeg", 80)

	// ETag round-trip -> 304.
	first, _ := http.Get(api + "/books/" + bookID + "/pages/0")
	etag := first.Header.Get("ETag")
	first.Body.Close()
	if etag == "" {
		t.Fatal("no ETag on page response")
	}
	req, _ := http.NewRequest(http.MethodGet, api+"/books/"+bookID+"/pages/0", nil)
	req.Header.Set("If-None-Match", etag)
	cond, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("conditional get: %v", err)
	}
	cond.Body.Close()
	if cond.StatusCode != http.StatusNotModified {
		t.Fatalf("If-None-Match status = %d, want 304", cond.StatusCode)
	}

	// Prefetch accepted.
	pf, _ := http.Post(api+"/books/"+bookID+"/prefetch", "application/json", bytes.NewBufferString(`{"from":0,"count":2}`))
	if pf.StatusCode != http.StatusAccepted {
		t.Fatalf("prefetch status = %d, want 202", pf.StatusCode)
	}
}

// assertImage GETs url and checks the content type + decoded width.
func assertImage(t *testing.T, url, wantType string, wantWidth int) {
	t.Helper()
	res, err := http.Get(url)
	if err != nil {
		t.Fatalf("get %s: %v", url, err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("%s status = %d", url, res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct != wantType {
		t.Errorf("%s content-type = %q, want %q", url, ct, wantType)
	}
	cfg, _, err := image.DecodeConfig(res.Body)
	if err != nil {
		t.Fatalf("%s decode: %v", url, err)
	}
	if cfg.Width != wantWidth {
		t.Errorf("%s width = %d, want %d", url, cfg.Width, wantWidth)
	}
}
