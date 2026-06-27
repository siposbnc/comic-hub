package archive

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// makeCBZ writes a CBZ with the given entries (name -> contents) to a temp file and
// returns its path. Map iteration order is random, which usefully checks that Open
// sorts entries regardless of stored order.
func makeCBZ(t *testing.T, entries map[string][]byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "test.cbz")
	f, err := os.Create(p)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	for name, data := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create entry %q: %v", name, err)
		}
		if _, err := w.Write(data); err != nil {
			t.Fatalf("write entry %q: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return p
}

func TestCBZOpenOrdersPagesAndReadsSidecar(t *testing.T) {
	path := makeCBZ(t, map[string][]byte{
		"page10.png":    []byte("ten"),
		"page2.png":     []byte("two"),
		"page1.png":     []byte("one"),
		"ComicInfo.xml": []byte("<ComicInfo><Number>1</Number></ComicInfo>"),
		"notes.txt":     []byte("ignored"), // non-image, skipped
		"../evil.png":   []byte("traversal"),
	})

	src, err := CBZ{}.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer src.Close()

	if src.PageCount() != 3 {
		t.Fatalf("PageCount = %d, want 3 (non-image and traversal entries excluded)", src.PageCount())
	}

	// Natural order: page1, page2, page10.
	wantOrder := []string{"page1.png", "page2.png", "page10.png"}
	wantBytes := []string{"one", "two", "ten"}
	for i, wantName := range wantOrder {
		rc, info, err := src.Page(i)
		if err != nil {
			t.Fatalf("page %d: %v", i, err)
		}
		if info.FileName != wantName {
			t.Errorf("page %d name = %q, want %q", i, info.FileName, wantName)
		}
		data, _ := io.ReadAll(rc)
		rc.Close()
		if string(data) != wantBytes[i] {
			t.Errorf("page %d bytes = %q, want %q", i, data, wantBytes[i])
		}
	}

	// Sidecar.
	r, ok := src.Sidecar()
	if !ok {
		t.Fatal("expected ComicInfo.xml sidecar")
	}
	data, _ := io.ReadAll(r)
	if len(data) == 0 {
		t.Fatal("sidecar empty")
	}
}

func TestCBZPageOutOfRange(t *testing.T) {
	path := makeCBZ(t, map[string][]byte{"p1.jpg": []byte("x")})
	src, err := CBZ{}.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer src.Close()
	if _, _, err := src.Page(5); err == nil {
		t.Fatal("expected out-of-range error")
	}
}

func TestCBZNoSidecar(t *testing.T) {
	path := makeCBZ(t, map[string][]byte{"p1.jpg": []byte("x")})
	src, err := CBZ{}.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer src.Close()
	if _, ok := src.Sidecar(); ok {
		t.Fatal("expected no sidecar")
	}
}
