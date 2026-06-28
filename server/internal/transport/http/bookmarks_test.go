package http

import (
	"net/http"
	"path/filepath"
	"testing"
)

func TestBookmarksEndpoint(t *testing.T) {
	srv, _ := newScanServer(t)
	api := srv + "/api/v1"

	root := t.TempDir()
	writeImageCBZ(t, filepath.Join(root, "Saga", "Saga 001.cbz"), map[string][]byte{
		"p1.png": makePNGBytes(60, 90),
		"p2.png": makePNGBytes(60, 90),
		"p3.png": makePNGBytes(60, 90),
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

	// No bookmarks yet.
	var list struct {
		Items []bookmarkDTO `json:"items"`
	}
	getJSON(t, api+"/me/books/"+book+"/bookmarks", &list)
	if len(list.Items) != 0 {
		t.Fatalf("expected no bookmarks, got %+v", list.Items)
	}

	// Add a bookmark with a note.
	var created bookmarkDTO
	decode(t, sendJSON(t, http.MethodPost, api+"/me/books/"+book+"/bookmarks",
		`{"page":1,"note":"cliffhanger"}`), &created)
	if created.ID == "" || created.Page != 1 || created.Note != "cliffhanger" {
		t.Fatalf("create = %+v", created)
	}

	// Re-adding the same page is idempotent: updates the note, keeps the id.
	var readd bookmarkDTO
	decode(t, sendJSON(t, http.MethodPost, api+"/me/books/"+book+"/bookmarks",
		`{"page":1,"note":"the big reveal"}`), &readd)
	if readd.ID != created.ID || readd.Note != "the big reveal" {
		t.Fatalf("re-add = %+v (want id %s)", readd, created.ID)
	}

	// A second bookmark on another page; list is ordered by page ascending.
	mustStatus(t, sendJSON(t, http.MethodPost, api+"/me/books/"+book+"/bookmarks",
		`{"page":0}`), http.StatusCreated)
	getJSON(t, api+"/me/books/"+book+"/bookmarks", &list)
	if len(list.Items) != 2 || list.Items[0].Page != 0 || list.Items[1].Page != 1 {
		t.Fatalf("list = %+v", list.Items)
	}

	// Edit a note via PATCH.
	var edited bookmarkDTO
	decode(t, sendJSON(t, http.MethodPatch, api+"/me/books/"+book+"/bookmarks/"+created.ID,
		`{"note":"act two"}`), &edited)
	if edited.Note != "act two" {
		t.Fatalf("edit = %+v", edited)
	}

	// Delete one, then it's gone.
	mustStatus(t, sendJSON(t, http.MethodDelete, api+"/me/books/"+book+"/bookmarks/"+created.ID, ``),
		http.StatusNoContent)
	getJSON(t, api+"/me/books/"+book+"/bookmarks", &list)
	if len(list.Items) != 1 || list.Items[0].Page != 0 {
		t.Fatalf("after delete = %+v", list.Items)
	}

	// Bookmarking a missing book 404s; editing a missing bookmark 404s.
	mustStatus(t, sendJSON(t, http.MethodPost, api+"/me/books/nope/bookmarks", `{"page":0}`),
		http.StatusNotFound)
	mustStatus(t, sendJSON(t, http.MethodPatch, api+"/me/books/"+book+"/bookmarks/nope", `{"note":"x"}`),
		http.StatusNotFound)
}
