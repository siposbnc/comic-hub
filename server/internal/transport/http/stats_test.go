package http

import (
	"path/filepath"
	"testing"
)

// TestMyStatsEndpoint: the dashboard endpoint aggregates real progress writes.
func TestMyStatsEndpoint(t *testing.T) {
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

	// Finish one book, leave one mid-read.
	putProgress(t, api, books[0], `{"page":1,"status":"read"}`)
	putProgress(t, api, books[1], `{"page":1,"status":"in_progress"}`)

	var got struct {
		BooksRead int `json:"booksRead"`
		PagesRead int `json:"pagesRead"`
		ThisYear  int `json:"thisYear"`
		Streak    int `json:"streak"`
		Months    []struct {
			Label string `json:"m"`
			Count int    `json:"n"`
		} `json:"months"`
		Finished []struct {
			BookID     string `json:"bookId"`
			SeriesName string `json:"seriesName"`
		} `json:"finished"`
		Genres []any `json:"genres"`
	}
	getJSON(t, api+"/me/stats", &got)

	if got.BooksRead != 1 || got.PagesRead != 2+1 || got.ThisYear != 1 {
		t.Fatalf("headline = %d/%d/%d, want 1 book / 3 pages / 1 this year", got.BooksRead, got.PagesRead, got.ThisYear)
	}
	if got.Streak != 1 {
		t.Fatalf("streak = %d, want 1 (read today)", got.Streak)
	}
	if len(got.Months) != 12 || got.Months[11].Count != 1 {
		t.Fatalf("months = %+v, want 12 buckets with 1 in the newest", got.Months)
	}
	if len(got.Finished) != 1 || got.Finished[0].BookID != books[0] || got.Finished[0].SeriesName != "Saga" {
		t.Fatalf("finished = %+v, want the read book with its series name", got.Finished)
	}
	if got.Genres == nil {
		t.Fatal("genres must be [] (not null) when empty")
	}
}
