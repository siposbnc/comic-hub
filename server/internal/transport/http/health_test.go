package http

import (
	"path/filepath"
	"testing"
)

func TestLibraryHealthEndpoint(t *testing.T) {
	srv, _ := newScanServer(t)
	api := srv + "/api/v1"

	root := t.TempDir()
	writeImageCBZ(t, filepath.Join(root, "Saga", "Saga 001.cbz"), map[string][]byte{
		"p1.png": makePNGBytes(60, 90),
	})
	lib := createLibrary(t, api, root)
	startAndAwaitScan(t, api, lib.ID)

	var rep struct {
		Counts struct {
			Books     int `json:"books"`
			Corrupt   int `json:"corrupt"`
			Orphans   int `json:"orphans"`
			Unmatched int `json:"unmatched"`
		} `json:"counts"`
	}
	getJSON(t, api+"/libraries/"+lib.ID+"/health", &rep)

	// One freshly-scanned, on-disk, never-matched book: present, not corrupt/orphaned,
	// but unmatched (no metadata yet).
	if rep.Counts.Books != 1 || rep.Counts.Corrupt != 0 || rep.Counts.Orphans != 0 ||
		rep.Counts.Unmatched != 1 {
		t.Fatalf("health counts = %+v", rep.Counts)
	}
}
