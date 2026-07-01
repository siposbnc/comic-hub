package http

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// A book with no progress row yet returns a 200 with a default "unread" progress (page 0,
// real page count) so the reader opening a fresh book doesn't have to special-case a 404.
// An unknown book is still a 404.
func TestGetProgressDefaultsWhenNone(t *testing.T) {
	srv, _ := newScanServer(t)
	api := srv + "/api/v1"

	root := t.TempDir()
	writeImageCBZ(t, filepath.Join(root, "Saga", "Saga 001.cbz"), map[string][]byte{
		"p1.png": makePNGBytes(60, 90),
		"p2.png": makePNGBytes(60, 90),
	})
	lib := createLibrary(t, api, root)
	startAndAwaitScan(t, api, lib.ID)

	var seriesList struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	getJSON(t, api+"/series?library="+lib.ID, &seriesList)
	var detail struct {
		Books []struct {
			ID string `json:"id"`
		} `json:"books"`
	}
	getJSON(t, api+"/series/"+seriesList.Items[0].ID, &detail)
	book := detail.Books[0].ID

	// No progress yet → 200 with sensible defaults, not 404.
	var got struct {
		BookID    string  `json:"bookId"`
		Page      int     `json:"page"`
		PageCount int     `json:"pageCount"`
		Status    string  `json:"status"`
		Percent   float64 `json:"percent"`
	}
	getJSON(t, api+"/me/progress/"+book, &got)
	if got.BookID != book || got.Page != 0 || got.Status != "unread" || got.PageCount != 2 || got.Percent != 0 {
		t.Fatalf("default progress = %+v, want page 0 / unread / pageCount 2 / 0%%", got)
	}

	// An unknown book is still a 404.
	resp, err := http.Get(api + "/me/progress/does-not-exist")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	mustStatus(t, resp, http.StatusNotFound)
}

// scanOneSeries scans a root with the given CBZs and returns the book ids, sorted by
// issue number.
func scanOneSeries(t *testing.T, api, root string, count int) []string {
	t.Helper()
	lib := createLibrary(t, api, root)
	startAndAwaitScan(t, api, lib.ID)

	var seriesList struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	getJSON(t, api+"/series?library="+lib.ID, &seriesList)
	var detail struct {
		Books []struct {
			ID string `json:"id"`
		} `json:"books"`
	}
	getJSON(t, api+"/series/"+seriesList.Items[0].ID, &detail)
	if len(detail.Books) != count {
		t.Fatalf("scan produced %d books, want %d", len(detail.Books), count)
	}
	ids := make([]string, count)
	for i, b := range detail.Books {
		ids[i] = b.ID
	}
	return ids
}

func putProgress(t *testing.T, api, bookID, body string) progressDTO {
	t.Helper()
	res := sendJSON(t, http.MethodPut, api+"/me/progress/"+bookID, body)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("put progress = %d, want 200", res.StatusCode)
	}
	var dto progressDTO
	if err := json.NewDecoder(res.Body).Decode(&dto); err != nil {
		t.Fatalf("decode progress: %v", err)
	}
	return dto
}

// TestProgressLastWriterWinsByUpdatedAt: a replayed write timestamped older than the
// stored row is rejected — the response carries the authoritative (newer) row — while
// newer-timestamped and untimestamped (interactive) writes apply. ADR-008.
func TestProgressLastWriterWinsByUpdatedAt(t *testing.T) {
	srv, _ := newScanServer(t)
	api := srv + "/api/v1"

	root := t.TempDir()
	writeImageCBZ(t, filepath.Join(root, "Saga", "Saga 001.cbz"), map[string][]byte{
		"p1.png": makePNGBytes(40, 60), "p2.png": makePNGBytes(40, 60),
	})
	book := scanOneSeries(t, api, root, 1)[0]

	// Device A read to page 1 at T2.
	const t1, t2 = 1_000_000, 2_000_000
	got := putProgress(t, api, book, `{"page":1,"status":"in_progress","device":"tablet","updatedAt":`+strconv.Itoa(t2)+`}`)
	if got.Page != 1 || got.UpdatedAt != t2 {
		t.Fatalf("write@T2 = %+v, want page 1 @ T2", got)
	}

	// Device B replays older progress (T1, page 0): rejected, authoritative row returned.
	got = putProgress(t, api, book, `{"page":0,"status":"in_progress","device":"phone","updatedAt":`+strconv.Itoa(t1)+`}`)
	if got.Page != 1 || got.UpdatedAt != t2 {
		t.Fatalf("stale write response = %+v, want authoritative page 1 @ T2", got)
	}
	var stored progressDTO
	getJSON(t, api+"/me/progress/"+book, &stored)
	if stored.Page != 1 || stored.UpdatedAt != t2 {
		t.Fatalf("stored after stale write = %+v, want page 1 @ T2 (unclobbered)", stored)
	}

	// An untimestamped (interactive) write always applies, stamped server-side.
	got = putProgress(t, api, book, `{"page":0,"status":"in_progress"}`)
	if got.Page != 0 || got.UpdatedAt <= t2 {
		t.Fatalf("interactive write = %+v, want page 0 with fresh updatedAt", got)
	}
}

// TestProgressBatchFlush: the reader's offline flush applies each item independently
// and reports which writes won.
func TestProgressBatchFlush(t *testing.T) {
	srv, _ := newScanServer(t)
	api := srv + "/api/v1"

	root := t.TempDir()
	writeImageCBZ(t, filepath.Join(root, "Saga", "Saga 001.cbz"), map[string][]byte{
		"p1.png": makePNGBytes(40, 60), "p2.png": makePNGBytes(40, 60),
	})
	writeImageCBZ(t, filepath.Join(root, "Saga", "Saga 002.cbz"), map[string][]byte{
		"p1.png": makePNGBytes(40, 60), "p2.png": makePNGBytes(40, 60),
	})
	books := scanOneSeries(t, api, root, 2)

	// Book 0 already has newer progress than the flush will carry.
	putProgress(t, api, books[0], `{"page":1,"status":"in_progress","updatedAt":5000000}`)

	res := sendJSON(t, http.MethodPost, api+"/me/progress/batch", `{"items":[
		{"bookId":"`+books[0]+`","page":0,"status":"in_progress","updatedAt":4000000},
		{"bookId":"`+books[1]+`","page":1,"status":"in_progress","updatedAt":4000000},
		{"bookId":"no-such-book","page":3}
	]}`)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("batch = %d, want 200", res.StatusCode)
	}
	var out struct {
		Items []struct {
			BookID   string       `json:"bookId"`
			Applied  bool         `json:"applied"`
			Progress *progressDTO `json:"progress"`
			Error    string       `json:"error"`
		} `json:"items"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatalf("decode batch: %v", err)
	}
	if len(out.Items) != 3 {
		t.Fatalf("batch items = %d, want 3", len(out.Items))
	}
	stale, fresh, missing := out.Items[0], out.Items[1], out.Items[2]
	if stale.Applied || stale.Progress == nil || stale.Progress.Page != 1 {
		t.Errorf("stale item = %+v, want applied=false with authoritative page 1", stale)
	}
	if !fresh.Applied || fresh.Progress == nil || fresh.Progress.Page != 1 || fresh.Progress.UpdatedAt != 4000000 {
		t.Errorf("fresh item = %+v, want applied page 1 @ 4000000", fresh)
	}
	if missing.Applied || missing.Error == "" || !strings.Contains(missing.Error, "not found") {
		t.Errorf("missing item = %+v, want not-found error", missing)
	}

	// An empty flush is a 400, not a silent no-op.
	res = sendJSON(t, http.MethodPost, api+"/me/progress/batch", `{"items":[]}`)
	mustStatus(t, res, http.StatusBadRequest)
}

// TestProgressBatchByContentHash: standalone-mode progress is keyed by the file's
// content hash (the reader has no book id); the flush resolves it to the catalog book.
func TestProgressBatchByContentHash(t *testing.T) {
	srv, store := newScanServer(t)
	api := srv + "/api/v1"

	root := t.TempDir()
	writeImageCBZ(t, filepath.Join(root, "Saga", "Saga 001.cbz"), map[string][]byte{
		"p1.png": makePNGBytes(40, 60), "p2.png": makePNGBytes(40, 60),
	})
	bookID := scanOneSeries(t, api, root, 1)[0]
	book, err := store.Books().Get(t.Context(), bookID)
	if err != nil || book.ContentHash == "" {
		t.Fatalf("book content hash unavailable: %v (hash %q)", err, book.ContentHash)
	}

	res := sendJSON(t, http.MethodPost, api+"/me/progress/batch", `{"items":[
		{"contentHash":"`+book.ContentHash+`","page":1,"status":"in_progress","updatedAt":4000000},
		{"contentHash":"not-a-real-hash","page":1},
		{"page":1}
	]}`)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("batch = %d, want 200", res.StatusCode)
	}
	var out struct {
		Items []struct {
			BookID   string       `json:"bookId"`
			Applied  bool         `json:"applied"`
			Progress *progressDTO `json:"progress"`
			Error    string       `json:"error"`
		} `json:"items"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatalf("decode batch: %v", err)
	}
	matched, missing, keyless := out.Items[0], out.Items[1], out.Items[2]
	if !matched.Applied || matched.BookID != bookID || matched.Progress == nil || matched.Progress.Page != 1 {
		t.Errorf("hash item = %+v, want applied to %s at page 1", matched, bookID)
	}
	if missing.Applied || missing.Error == "" {
		t.Errorf("unknown-hash item = %+v, want error", missing)
	}
	if keyless.Applied || keyless.Error == "" {
		t.Errorf("keyless item = %+v, want error", keyless)
	}

	// The flush landed as real progress on the resolved book.
	var stored progressDTO
	getJSON(t, api+"/me/progress/"+bookID, &stored)
	if stored.Page != 1 || stored.UpdatedAt != 4000000 {
		t.Fatalf("stored progress = %+v, want page 1 @ 4000000", stored)
	}
}
