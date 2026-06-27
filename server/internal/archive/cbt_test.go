package archive

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// makeCBT writes a CBT (tar) with the given entries (name -> contents) to a temp file and
// returns its path. Map iteration order is random, which usefully checks that Open sorts
// entries regardless of stored order.
func makeCBT(t *testing.T, entries map[string][]byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "test.cbt")
	f, err := os.Create(p)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer f.Close()
	tw := tar.NewWriter(f)
	for name, data := range entries {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(data))}); err != nil {
			t.Fatalf("write header %q: %v", name, err)
		}
		if _, err := tw.Write(data); err != nil {
			t.Fatalf("write entry %q: %v", name, err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	return p
}

func TestCBTExtensions(t *testing.T) {
	got := CBT{}.Extensions()
	want := map[string]bool{"cbt": true, "tar": true}
	if len(got) != len(want) {
		t.Fatalf("extensions = %v", got)
	}
	for _, e := range got {
		if !want[e] {
			t.Errorf("unexpected extension %q", e)
		}
	}
}

func TestCBTOpenOrdersPagesAndReadsSidecar(t *testing.T) {
	path := makeCBT(t, map[string][]byte{
		"page10.png":    []byte("ten"),
		"page2.png":     []byte("two"),
		"page1.png":     []byte("one"),
		"ComicInfo.xml": []byte("<ComicInfo><Number>1</Number></ComicInfo>"),
		"notes.txt":     []byte("ignored"), // non-image, skipped
	})

	src, err := CBT{}.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer src.Close()

	if src.PageCount() != 3 {
		t.Fatalf("PageCount = %d, want 3 (non-image entries excluded)", src.PageCount())
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

	// Pages() metadata listing must agree with Page() and not decode.
	if pages := src.Pages(); len(pages) != 3 || pages[2].FileName != "page10.png" {
		t.Fatalf("Pages() = %+v", src.Pages())
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

func TestCBTPageOutOfRange(t *testing.T) {
	path := makeCBT(t, map[string][]byte{"p1.jpg": []byte("x")})
	src, err := CBT{}.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer src.Close()
	if _, _, err := src.Page(5); err == nil {
		t.Fatal("expected out-of-range error")
	}
}

func TestCBTNoSidecar(t *testing.T) {
	path := makeCBT(t, map[string][]byte{"p1.jpg": []byte("x")})
	src, err := CBT{}.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer src.Close()
	if _, ok := src.Sidecar(); ok {
		t.Fatal("expected no sidecar")
	}
}
