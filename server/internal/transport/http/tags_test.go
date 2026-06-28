package http

import (
	"net/http"
	"path/filepath"
	"testing"
)

func TestTagsEndpoint(t *testing.T) {
	srv, _ := newScanServer(t)
	api := srv + "/api/v1"

	root := t.TempDir()
	for _, n := range []string{"001", "002"} {
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

	// Create a tag; a duplicate name is rejected.
	var tag tagDTO
	decode(t, sendJSON(t, http.MethodPost, api+"/tags", `{"name":"Favorites","color":"#ff0066"}`), &tag)
	if tag.ID == "" || tag.Name != "Favorites" {
		t.Fatalf("create tag = %+v", tag)
	}
	mustStatus(t, sendJSON(t, http.MethodPost, api+"/tags", `{"name":"favorites"}`), http.StatusBadRequest)

	// Assign to both books; the tag's count and book listing reflect it.
	mustStatus(t, sendJSON(t, http.MethodPost, api+"/books/"+b["1"]+"/tags", `{"tagIds":["`+tag.ID+`"]}`),
		http.StatusNoContent)
	mustStatus(t, sendJSON(t, http.MethodPost, api+"/books/"+b["2"]+"/tags", `{"tagIds":["`+tag.ID+`"]}`),
		http.StatusNoContent)

	var tagList struct {
		Items []tagDTO `json:"items"`
	}
	getJSON(t, api+"/tags", &tagList)
	if len(tagList.Items) != 1 || tagList.Items[0].BookCount != 2 {
		t.Fatalf("tag list = %+v", tagList.Items)
	}

	var tagged struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	getJSON(t, api+"/tags/"+tag.ID+"/books", &tagged)
	if len(tagged.Items) != 2 {
		t.Fatalf("tagged books = %+v", tagged.Items)
	}

	// Book detail surfaces the tag.
	var bd struct {
		Tags []tagDTO `json:"tags"`
	}
	getJSON(t, api+"/books/"+b["1"], &bd)
	if len(bd.Tags) != 1 || bd.Tags[0].Name != "Favorites" {
		t.Fatalf("book detail tags = %+v", bd.Tags)
	}

	// Assigning an unknown tag id is a clear 404.
	mustStatus(t, sendJSON(t, http.MethodPost, api+"/books/"+b["1"]+"/tags", `{"tagIds":["missing"]}`),
		http.StatusNotFound)

	// Unassign, rename, then delete.
	mustStatus(t, sendJSON(t, http.MethodDelete, api+"/books/"+b["1"]+"/tags/"+tag.ID, ``), http.StatusNoContent)
	getJSON(t, api+"/tags", &tagList)
	if tagList.Items[0].BookCount != 1 {
		t.Fatalf("count after unassign = %d, want 1", tagList.Items[0].BookCount)
	}

	var renamed tagDTO
	decode(t, sendJSON(t, http.MethodPatch, api+"/tags/"+tag.ID, `{"name":"Top Picks"}`), &renamed)
	if renamed.Name != "Top Picks" {
		t.Fatalf("rename = %+v", renamed)
	}

	mustStatus(t, sendJSON(t, http.MethodDelete, api+"/tags/"+tag.ID, ``), http.StatusNoContent)
	getJSON(t, api+"/tags", &tagList)
	if len(tagList.Items) != 0 {
		t.Fatalf("tags after delete = %+v", tagList.Items)
	}
}
