package http

import (
	"bytes"
	"net/http"
	"path/filepath"
	"testing"
)

func TestBrowseAndProgress(t *testing.T) {
	srv, _ := newScanServer(t)
	api := srv + "/api/v1"

	root := t.TempDir()
	writeImageCBZ(t, filepath.Join(root, "Saga", "Saga 001.cbz"), map[string][]byte{
		"p1.png": makePNGBytes(60, 90), "p2.png": makePNGBytes(60, 90),
	})
	writeImageCBZ(t, filepath.Join(root, "Saga", "Saga 002.cbz"), map[string][]byte{
		"p1.png": makePNGBytes(60, 90), "p2.png": makePNGBytes(60, 90), "p3.png": makePNGBytes(60, 90),
	})

	lib := createLibrary(t, api, root)
	startAndAwaitScan(t, api, lib.ID)

	// Library grid: one series with two books.
	var seriesList struct {
		Items []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			BookCount   int    `json:"bookCount"`
			ReadCount   int    `json:"readCount"`
			CoverBookID string `json:"coverBookId"`
		} `json:"items"`
	}
	getJSON(t, api+"/series?library="+lib.ID, &seriesList)
	if len(seriesList.Items) != 1 {
		t.Fatalf("series = %+v", seriesList.Items)
	}
	sc := seriesList.Items[0]
	if sc.Name != "Saga" || sc.BookCount != 2 || sc.CoverBookID == "" {
		t.Fatalf("series card = %+v", sc)
	}

	// Series detail: two issues.
	var detail struct {
		BookCount int `json:"bookCount"`
		Books     []struct {
			ID     string `json:"id"`
			Number string `json:"number"`
		} `json:"books"`
	}
	getJSON(t, api+"/series/"+sc.ID, &detail)
	if detail.BookCount != 2 || len(detail.Books) != 2 {
		t.Fatalf("series detail = %+v", detail)
	}
	bookID := detail.Books[0].ID

	// Book detail.
	var bd struct {
		ID         string `json:"id"`
		SeriesName string `json:"seriesName"`
		PageCount  int    `json:"pageCount"`
	}
	getJSON(t, api+"/books/"+bookID, &bd)
	if bd.ID != bookID || bd.SeriesName != "Saga" || bd.PageCount != 2 {
		t.Fatalf("book detail = %+v", bd)
	}

	// Update progress -> in_progress, then it appears in Continue Reading. (Explicit
	// status, since page 1 of a 2-page book would otherwise auto-derive to "read".)
	putJSON(t, api+"/me/progress/"+bookID, `{"page":1,"status":"in_progress"}`)
	var cont struct {
		Items []struct {
			ID       string `json:"id"`
			Progress *struct {
				Status string `json:"status"`
			} `json:"progress"`
		} `json:"items"`
	}
	getJSON(t, api+"/me/continue", &cont)
	if len(cont.Items) != 1 || cont.Items[0].ID != bookID {
		t.Fatalf("continue = %+v", cont.Items)
	}

	// Mark read.
	res, err := http.Post(api+"/me/books/"+bookID+"/mark", "application/json", bytes.NewBufferString(`{"status":"read"}`))
	if err != nil {
		t.Fatalf("mark: %v", err)
	}
	var marked progressDTO
	decode(t, res, &marked)
	if marked.Status != "read" {
		t.Fatalf("mark status = %q, want read", marked.Status)
	}

	// After reading, series detail shows readCount 1.
	getJSON(t, api+"/series/"+sc.ID, &detail)
	if detail.BookCount != 2 {
		t.Fatalf("series detail post-read = %+v", detail)
	}

	// Discover: recently added has both books.
	var disc struct {
		ContinueReading []any `json:"continueReading"`
		RecentlyAdded   []any `json:"recentlyAdded"`
	}
	getJSON(t, api+"/discover?library="+lib.ID, &disc)
	if len(disc.RecentlyAdded) != 2 {
		t.Fatalf("recently added = %d, want 2", len(disc.RecentlyAdded))
	}
}

func getJSON(t *testing.T, url string, dst any) {
	t.Helper()
	res, err := http.Get(url)
	if err != nil {
		t.Fatalf("get %s: %v", url, err)
	}
	if res.StatusCode != http.StatusOK {
		res.Body.Close()
		t.Fatalf("get %s status = %d", url, res.StatusCode)
	}
	decode(t, res, dst)
}

func putJSON(t *testing.T, url, body string) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPut, url, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("put %s: %v", url, err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("put %s status = %d", url, res.StatusCode)
	}
}
