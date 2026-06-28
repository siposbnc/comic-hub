package http

import (
	"net/http"
	"path/filepath"
	"testing"
)

func TestSmartListsEndpoint(t *testing.T) {
	srv, _ := newScanServer(t)
	api := srv + "/api/v1"

	root := t.TempDir()
	writeImageCBZ(t, filepath.Join(root, "Batman", "Batman 001.cbz"), map[string][]byte{
		"p1.png": makePNGBytes(60, 90),
	})
	writeImageCBZ(t, filepath.Join(root, "Saga", "Saga 001.cbz"), map[string][]byte{
		"p1.png": makePNGBytes(60, 90),
	})
	lib := createLibrary(t, api, root)
	startAndAwaitScan(t, api, lib.ID)

	// Create a smart list: series is Batman.
	var created smartListDTO
	decode(t, sendJSON(t, http.MethodPost, api+"/smart-lists",
		`{"name":"Bat books","rules":{"match":"all","rules":[{"field":"series","op":"is","value":"Batman"}]}}`),
		&created)
	if created.ID == "" || created.Name != "Bat books" {
		t.Fatalf("create = %+v", created)
	}

	// It appears in the list with a live count.
	var list struct {
		Items []smartListDTO `json:"items"`
	}
	getJSON(t, api+"/smart-lists", &list)
	if len(list.Items) != 1 || list.Items[0].BookCount != 1 {
		t.Fatalf("list = %+v", list.Items)
	}

	// Results return the matching book(s).
	var results struct {
		SmartList smartListDTO `json:"smartList"`
		Books     []struct {
			ID string `json:"id"`
		} `json:"books"`
	}
	getJSON(t, api+"/smart-lists/"+created.ID+"/results", &results)
	if len(results.Books) != 1 {
		t.Fatalf("results = %+v", results.Books)
	}

	// Invalid rules are rejected.
	mustStatus(t, sendJSON(t, http.MethodPost, api+"/smart-lists",
		`{"name":"bad","rules":{"match":"all","rules":[{"field":"nope","op":"is","value":"x"}]}}`),
		http.StatusBadRequest)
	mustStatus(t, sendJSON(t, http.MethodPost, api+"/smart-lists",
		`{"name":"empty","rules":{"match":"all","rules":[]}}`), http.StatusBadRequest)

	// Update the rule to match nothing, then delete.
	var updated smartListDTO
	decode(t, sendJSON(t, http.MethodPatch, api+"/smart-lists/"+created.ID,
		`{"rules":{"match":"all","rules":[{"field":"format","op":"is","value":"cbr"}]}}`), &updated)
	getJSON(t, api+"/smart-lists/"+created.ID+"/results", &results)
	if len(results.Books) != 0 {
		t.Fatalf("after update results = %+v", results.Books)
	}

	mustStatus(t, sendJSON(t, http.MethodDelete, api+"/smart-lists/"+created.ID, ``), http.StatusNoContent)
	getJSON(t, api+"/smart-lists", &list)
	if len(list.Items) != 0 {
		t.Fatalf("after delete = %+v", list.Items)
	}
}
