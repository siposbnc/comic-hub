package http

import (
	"net/http"
	"path/filepath"
	"testing"
)

func TestReaderPrefsEndpoint(t *testing.T) {
	srv, _ := newScanServer(t)
	api := srv + "/api/v1"

	root := t.TempDir()
	writeImageCBZ(t, filepath.Join(root, "Saga", "Saga 001.cbz"), map[string][]byte{
		"p1.png": makePNGBytes(60, 90),
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

	// No prefs yet → empty object.
	var got struct {
		Settings map[string]any `json:"settings"`
	}
	getJSON(t, api+"/me/books/"+book+"/reader-prefs", &got)
	if len(got.Settings) != 0 {
		t.Fatalf("expected empty prefs, got %+v", got.Settings)
	}

	// Save, then read back.
	mustStatus(t, sendJSON(t, http.MethodPut, api+"/me/books/"+book+"/reader-prefs",
		`{"settings":{"layout":"continuous","background":"sepia"}}`), http.StatusNoContent)
	getJSON(t, api+"/me/books/"+book+"/reader-prefs", &got)
	if got.Settings["layout"] != "continuous" || got.Settings["background"] != "sepia" {
		t.Fatalf("prefs round-trip = %+v", got.Settings)
	}

	// Saving prefs for a missing book 404s.
	mustStatus(t, sendJSON(t, http.MethodPut, api+"/me/books/nope/reader-prefs",
		`{"settings":{"a":1}}`), http.StatusNotFound)
}
