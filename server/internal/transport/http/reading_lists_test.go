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

// TestReadingListCollectionGroup covers referencing a whole collection into a reading list:
// it renders as a group, its books expand into the flat order, the reading chain walks
// through them, re-adding is a no-op, and deleting the collection drops the group.
func TestReadingListCollectionGroup(t *testing.T) {
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

	// A collection of two issues.
	var col collectionDTO
	decode(t, sendJSON(t, http.MethodPost, api+"/collections", `{"name":"Crisis"}`), &col)
	mustStatus(t, sendJSON(t, http.MethodPost, api+"/collections/"+col.ID+"/items",
		`{"bookIds":["`+b["2"]+`","`+b["3"]+`"]}`), http.StatusNoContent)

	// A list with one individual issue, then the whole collection as a group.
	var rl readingListDTO
	decode(t, sendJSON(t, http.MethodPost, api+"/me/reading-lists", `{"name":"Order"}`), &rl)
	mustStatus(t, sendJSON(t, http.MethodPost, api+"/me/reading-lists/"+rl.ID+"/items",
		`{"bookIds":["`+b["1"]+`"]}`), http.StatusNoContent)
	mustStatus(t, sendJSON(t, http.MethodPost, api+"/me/reading-lists/"+rl.ID+"/collections",
		`{"collectionIds":["`+col.ID+`"]}`), http.StatusNoContent)
	// Re-adding the same collection is a no-op (partial unique index).
	mustStatus(t, sendJSON(t, http.MethodPost, api+"/me/reading-lists/"+rl.ID+"/collections",
		`{"collectionIds":["`+col.ID+`"]}`), http.StatusNoContent)

	getList := func() struct {
		Items []readingListEntryDTO `json:"items"`
		Books []struct {
			ID string `json:"id"`
		} `json:"books"`
	} {
		var d struct {
			Items []readingListEntryDTO `json:"items"`
			Books []struct {
				ID string `json:"id"`
			} `json:"books"`
		}
		getJSON(t, api+"/me/reading-lists/"+rl.ID, &d)
		return d
	}

	d := getList()
	if len(d.Items) != 2 {
		t.Fatalf("want 2 entries (1 book + 1 group), got %d", len(d.Items))
	}
	if d.Items[0].Kind != "book" || d.Items[0].Book == nil || d.Items[0].Book.ID != b["1"] {
		t.Fatalf("entry 0 = %+v", d.Items[0])
	}
	grp := d.Items[1]
	if grp.Kind != "collection" || grp.Collection == nil || grp.Collection.Name != "Crisis" {
		t.Fatalf("entry 1 = %+v", grp)
	}
	if got := []string{grp.Collection.Books[0].ID, grp.Collection.Books[1].ID}; !equalStr(got, []string{b["2"], b["3"]}) {
		t.Fatalf("group books = %v", got)
	}
	// The flat books array includes the group's members.
	if len(d.Books) != 3 {
		t.Fatalf("want 3 flat books, got %d", len(d.Books))
	}

	// The list's bookCount reflects the expanded reading order (1 individual + 2 members),
	// not the raw entry count (which would be 2).
	var lists struct {
		Items []readingListDTO `json:"items"`
	}
	getJSON(t, api+"/me/reading-lists", &lists)
	if len(lists.Items) != 1 || lists.Items[0].BookCount != 3 {
		t.Fatalf("list bookCount = %+v, want 3", lists.Items)
	}

	// The reading chain walks the group in order: 1 → 2 → 3.
	mustStatus(t, sendJSON(t, http.MethodPost, api+"/me/reading-lists/"+rl.ID+"/active", ``), http.StatusNoContent)
	next := func(from string) string {
		var nb struct {
			Book *struct {
				ID string `json:"id"`
			} `json:"book"`
		}
		getJSON(t, api+"/me/books/"+from+"/next?context=readingList", &nb)
		if nb.Book == nil {
			return ""
		}
		return nb.Book.ID
	}
	if got := next(b["1"]); got != b["2"] {
		t.Fatalf("next after 1 = %q, want 2", got)
	}
	if got := next(b["2"]); got != b["3"] {
		t.Fatalf("next after 2 = %q, want 3", got)
	}

	// Deleting the collection drops its group from the list, leaving the individual issue.
	mustStatus(t, sendJSON(t, http.MethodDelete, api+"/collections/"+col.ID, ``), http.StatusNoContent)
	d = getList()
	if len(d.Items) != 1 || d.Items[0].Book == nil || d.Items[0].Book.ID != b["1"] {
		t.Fatalf("after collection delete, entries = %+v", d.Items)
	}
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
