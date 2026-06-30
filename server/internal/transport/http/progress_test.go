package http

import (
	"net/http"
	"path/filepath"
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
