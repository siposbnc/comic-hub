package http

import (
	"net/http"
	"path/filepath"
	"testing"
)

func TestReadingListsEndpoint(t *testing.T) {
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
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
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

	// Create, then it shows up in the user's lists.
	var created readingListDTO
	decode(t, sendJSON(t, http.MethodPost, api+"/me/reading-lists", `{"name":"To Read"}`), &created)
	if created.ID == "" || created.Name != "To Read" {
		t.Fatalf("create = %+v", created)
	}
	var list struct {
		Items []readingListDTO `json:"items"`
	}
	getJSON(t, api+"/me/reading-lists", &list)
	if len(list.Items) != 1 || list.Items[0].ID != created.ID {
		t.Fatalf("list = %+v", list.Items)
	}

	// Add items, then reorder + remove like collections.
	mustStatus(t, sendJSON(t, http.MethodPost, api+"/me/reading-lists/"+created.ID+"/items",
		`{"bookIds":["`+b["1"]+`","`+b["2"]+`","`+b["3"]+`"]}`), http.StatusNoContent)
	if got := readingListOrder(t, api, created.ID); !equalStr(got, []string{b["1"], b["2"], b["3"]}) {
		t.Fatalf("initial order = %v", got)
	}

	mustStatus(t, sendJSON(t, http.MethodPatch, api+"/me/reading-lists/"+created.ID+"/items/reorder",
		`{"bookId":"`+b["3"]+`","beforeId":"`+b["1"]+`"}`), http.StatusNoContent)
	if got := readingListOrder(t, api, created.ID); !equalStr(got, []string{b["3"], b["1"], b["2"]}) {
		t.Fatalf("after reorder = %v", got)
	}

	mustStatus(t, sendJSON(t, http.MethodDelete, api+"/me/reading-lists/"+created.ID+"/items/"+b["1"], ``),
		http.StatusNoContent)
	if got := readingListOrder(t, api, created.ID); !equalStr(got, []string{b["3"], b["2"]}) {
		t.Fatalf("after remove = %v", got)
	}

	// Rename, then delete.
	var renamed readingListDTO
	decode(t, sendJSON(t, http.MethodPatch, api+"/me/reading-lists/"+created.ID, `{"name":"Up Next"}`), &renamed)
	if renamed.Name != "Up Next" || renamed.BookCount != 2 {
		t.Fatalf("rename = %+v", renamed)
	}
	mustStatus(t, sendJSON(t, http.MethodDelete, api+"/me/reading-lists/"+created.ID, ``), http.StatusNoContent)

	// A missing list 404s rather than erroring opaquely.
	res, _ := http.Get(api + "/me/reading-lists/" + created.ID)
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("get after delete = %d, want 404", res.StatusCode)
	}
	res.Body.Close()
}

func readingListOrder(t *testing.T, api, id string) []string {
	t.Helper()
	var d struct {
		Books []struct {
			ID string `json:"id"`
		} `json:"books"`
	}
	getJSON(t, api+"/me/reading-lists/"+id, &d)
	out := make([]string, len(d.Books))
	for i, bk := range d.Books {
		out[i] = bk.ID
	}
	return out
}
