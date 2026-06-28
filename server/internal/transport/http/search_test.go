package http

import (
	"path/filepath"
	"testing"
)

func TestSearchEndpoint(t *testing.T) {
	srv, _ := newScanServer(t)
	api := srv + "/api/v1"

	root := t.TempDir()
	writeImageCBZ(t, filepath.Join(root, "Saga", "Saga 001.cbz"), map[string][]byte{
		"p1.png": makePNGBytes(60, 90),
	})
	writeImageCBZ(t, filepath.Join(root, "Batman", "Batman 001.cbz"), map[string][]byte{
		"p1.png": makePNGBytes(60, 90),
	})

	lib := createLibrary(t, api, root)
	startAndAwaitScan(t, api, lib.ID)

	type results struct {
		Query  string `json:"query"`
		Series []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"series"`
		Books []struct {
			ID string `json:"id"`
		} `json:"books"`
	}

	// Prefix match on the series name folder.
	var r results
	getJSON(t, api+"/search?q=sag", &r)
	if len(r.Series) != 1 || r.Series[0].Name != "Saga" {
		t.Fatalf("search 'sag' series = %+v", r.Series)
	}

	// type=series suppresses the books group even if titles matched.
	getJSON(t, api+"/search?q=bat&type=series", &r)
	if len(r.Series) != 1 || r.Series[0].Name != "Batman" {
		t.Fatalf("search 'bat' series = %+v", r.Series)
	}

	// A blank query returns empty groups, not an error.
	getJSON(t, api+"/search?q=", &r)
	if len(r.Series) != 0 || len(r.Books) != 0 {
		t.Fatalf("blank query returned %+v", r)
	}
}
