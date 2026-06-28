package http

import (
	"bytes"
	"net/http"
	"path/filepath"
	"testing"
)

func TestCollectionsEndpoint(t *testing.T) {
	srv, _ := newScanServer(t)
	api := srv + "/api/v1"

	// Scan three issues so we have real book ids to organize.
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
	if len(detail.Books) != 3 {
		t.Fatalf("expected 3 books, got %d", len(detail.Books))
	}
	b := map[string]string{}
	for _, bk := range detail.Books {
		b[bk.Number] = bk.ID
	}

	// Create a collection.
	var created collectionDTO
	decode(t, sendJSON(t, http.MethodPost, api+"/collections", `{"name":"Faves"}`), &created)
	if created.ID == "" || created.Name != "Faves" || created.BookCount != 0 {
		t.Fatalf("create = %+v", created)
	}

	// Add all three issues, in order.
	mustStatus(t, sendJSON(t, http.MethodPost, api+"/collections/"+created.ID+"/items",
		`{"bookIds":["`+b["1"]+`","`+b["2"]+`","`+b["3"]+`"]}`), http.StatusNoContent)

	if got := collectionOrder(t, api, created.ID); !equalStr(got, []string{b["1"], b["2"], b["3"]}) {
		t.Fatalf("initial order = %v", got)
	}

	// Move issue 3 to the front (before issue 1).
	mustStatus(t, sendJSON(t, http.MethodPatch, api+"/collections/"+created.ID+"/items/reorder",
		`{"bookId":"`+b["3"]+`","beforeId":"`+b["1"]+`"}`), http.StatusNoContent)
	if got := collectionOrder(t, api, created.ID); !equalStr(got, []string{b["3"], b["1"], b["2"]}) {
		t.Fatalf("after reorder-to-front = %v", got)
	}

	// Move issue 3 back to the end (no beforeId).
	mustStatus(t, sendJSON(t, http.MethodPatch, api+"/collections/"+created.ID+"/items/reorder",
		`{"bookId":"`+b["3"]+`"}`), http.StatusNoContent)
	if got := collectionOrder(t, api, created.ID); !equalStr(got, []string{b["1"], b["2"], b["3"]}) {
		t.Fatalf("after reorder-to-end = %v", got)
	}

	// Remove the middle issue.
	mustStatus(t, sendJSON(t, http.MethodDelete, api+"/collections/"+created.ID+"/items/"+b["2"], ``),
		http.StatusNoContent)
	if got := collectionOrder(t, api, created.ID); !equalStr(got, []string{b["1"], b["3"]}) {
		t.Fatalf("after remove = %v", got)
	}

	// Rename via PATCH; the list reflects it with the right count.
	var updated collectionDTO
	decode(t, sendJSON(t, http.MethodPatch, api+"/collections/"+created.ID, `{"name":"Best of Saga"}`), &updated)
	if updated.Name != "Best of Saga" || updated.BookCount != 2 {
		t.Fatalf("update = %+v", updated)
	}

	// Delete; the collection is then gone.
	mustStatus(t, sendJSON(t, http.MethodDelete, api+"/collections/"+created.ID, ``), http.StatusNoContent)
	res, _ := http.Get(api + "/collections/" + created.ID)
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("get after delete = %d, want 404", res.StatusCode)
	}
	res.Body.Close()
}

// collectionOrder fetches a collection and returns its book ids in display order.
func collectionOrder(t *testing.T, api, id string) []string {
	t.Helper()
	var d struct {
		Books []struct {
			ID string `json:"id"`
		} `json:"books"`
	}
	getJSON(t, api+"/collections/"+id, &d)
	out := make([]string, len(d.Books))
	for i, bk := range d.Books {
		out[i] = bk.ID
	}
	return out
}

func sendJSON(t *testing.T, method, url, body string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(method, url, bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	return res
}

func mustStatus(t *testing.T, res *http.Response, want int) {
	t.Helper()
	defer res.Body.Close()
	if res.StatusCode != want {
		t.Fatalf("status = %d, want %d", res.StatusCode, want)
	}
}

func equalStr(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
