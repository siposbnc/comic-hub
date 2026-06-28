package http

import (
	"net/http"
	"path/filepath"
	"testing"
)

// Exercises the active-reading-list queue: set active, Home "next up" skips read issues,
// and the per-book "next" endpoint follows both series and reading-list order.
func TestReadingQueue(t *testing.T) {
	srv, _ := newScanServer(t)
	api := srv + "/api/v1"

	root := t.TempDir()
	for _, n := range []string{"001", "002", "003"} {
		writeImageCBZ(t, filepath.Join(root, "Saga", "Saga "+n+".cbz"), map[string][]byte{
			"p1.png": makePNGBytes(60, 90),
		})
	}
	lib := createLibrary(t, api, root)
	startAndAwaitScan(t, api, lib.ID)

	var seriesList struct {
		Items []struct{ ID string } `json:"items"`
	}
	getJSON(t, api+"/series?library="+lib.ID, &seriesList)
	var detail struct {
		Books []struct {
			ID     string `json:"id"`
			Number string `json:"number"`
		} `json:"books"`
	}
	getJSON(t, api+"/series/"+seriesList.Items[0].ID, &detail)
	b := map[string]string{}
	for _, bk := range detail.Books {
		b[bk.Number] = bk.ID
	}

	// A reading list with the three issues in order, set active.
	var list readingListDTO
	decode(t, sendJSON(t, http.MethodPost, api+"/me/reading-lists", `{"name":"Queue"}`), &list)
	mustStatus(t, sendJSON(t, http.MethodPost, api+"/me/reading-lists/"+list.ID+"/items",
		`{"bookIds":["`+b["1"]+`","`+b["2"]+`","`+b["3"]+`"]}`), http.StatusNoContent)
	mustStatus(t, sendJSON(t, http.MethodPost, api+"/me/reading-lists/"+list.ID+"/active", ``),
		http.StatusNoContent)

	// Home "next up" is the first issue.
	type nextUp struct {
		NextUp *struct {
			Book     struct{ ID string } `json:"book"`
			ListName string              `json:"listName"`
		} `json:"nextUp"`
	}
	var disc nextUp
	getJSON(t, api+"/discover?library="+lib.ID, &disc)
	if disc.NextUp == nil || disc.NextUp.Book.ID != b["1"] || disc.NextUp.ListName != "Queue" {
		t.Fatalf("next up = %+v", disc.NextUp)
	}

	// Mark #1 read → next up advances to #2.
	mustStatus(t, sendJSON(t, http.MethodPost, api+"/me/books/"+b["1"]+"/mark", `{"status":"read"}`),
		http.StatusOK)
	getJSON(t, api+"/discover?library="+lib.ID, &disc)
	if disc.NextUp == nil || disc.NextUp.Book.ID != b["2"] {
		t.Fatalf("next up after read = %+v", disc.NextUp)
	}

	// next?context=readingList follows queue order; series order matches here too.
	var nb struct {
		Book *struct{ ID string } `json:"book"`
	}
	getJSON(t, api+"/me/books/"+b["2"]+"/next?context=readingList", &nb)
	if nb.Book == nil || nb.Book.ID != b["3"] {
		t.Fatalf("next in list after #2 = %+v", nb.Book)
	}
	getJSON(t, api+"/me/books/"+b["3"]+"/next?context=readingList", &nb)
	if nb.Book != nil {
		t.Fatalf("expected no next after last, got %+v", nb.Book)
	}
	getJSON(t, api+"/me/books/"+b["1"]+"/next?context=series", &nb)
	if nb.Book == nil || nb.Book.ID != b["2"] {
		t.Fatalf("next in series after #1 = %+v", nb.Book)
	}
}
