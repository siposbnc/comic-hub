package archive

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func writeTestCBZ(t *testing.T, path string, entries map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	for name, data := range entries {
		w, _ := zw.Create(name)
		_, _ = io.WriteString(w, data)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
}

func TestWriteCBZComicInfoReplacesAndPreserves(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Saga 001.cbz")
	// Tiny but valid PNGs aren't needed — the reader only inspects names/sizes here.
	writeTestCBZ(t, path, map[string]string{
		"01.jpg":        "imgone",
		"02.jpg":        "imgtwo",
		"ComicInfo.xml": `<?xml version="1.0"?><ComicInfo><Series>Stale</Series></ComicInfo>`,
	})

	xml := []byte(`<?xml version="1.0"?><ComicInfo><Series>Saga</Series><Number>1</Number></ComicInfo>`)
	if err := WriteCBZComicInfo(path, xml); err != nil {
		t.Fatalf("WriteCBZComicInfo: %v", err)
	}

	src, err := CBZ{}.Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer src.Close()

	if src.PageCount() != 2 {
		t.Fatalf("pages = %d, want 2 (preserved)", src.PageCount())
	}
	r, ok := src.Sidecar()
	if !ok {
		t.Fatal("expected a ComicInfo.xml sidecar after write")
	}
	data, _ := io.ReadAll(r)
	if got := string(data); !contains(got, "Saga") || contains(got, "Stale") {
		t.Fatalf("sidecar not replaced: %q", got)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
