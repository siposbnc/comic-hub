package archive

import (
	"io"
	"path/filepath"
	"testing"
)

func TestCBRExtensions(t *testing.T) {
	got := CBR{}.Extensions()
	want := map[string]bool{"cbr": true, "rar": true}
	if len(got) != len(want) {
		t.Fatalf("extensions = %v", got)
	}
	for _, e := range got {
		if !want[e] {
			t.Errorf("unexpected extension %q", e)
		}
	}
}

// TestCBRDecode exercises the real RAR decode path against a committed fixture
// (testdata/sample.cbr, generated with WinRAR: pages p1/p2/p10 + ComicInfo.xml). It
// mirrors the CBZ test so both formats are held to the same behavior.
func TestCBRDecode(t *testing.T) {
	src, err := CBR{}.Open(filepath.Join("testdata", "sample.cbr"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer src.Close()

	if src.PageCount() != 3 {
		t.Fatalf("PageCount = %d, want 3 (ComicInfo.xml excluded)", src.PageCount())
	}

	// Natural order: p1, p2, p10 (not lexical p1, p10, p2).
	wantName := []string{"p1.jpg", "p2.jpg", "p10.jpg"}
	wantBytes := []string{"ONE", "TWO", "TEN"}
	for i := range wantName {
		rc, info, err := src.Page(i)
		if err != nil {
			t.Fatalf("page %d: %v", i, err)
		}
		if info.FileName != wantName[i] {
			t.Errorf("page %d name = %q, want %q", i, info.FileName, wantName[i])
		}
		data, _ := io.ReadAll(rc)
		rc.Close()
		if string(data) != wantBytes[i] {
			t.Errorf("page %d bytes = %q, want %q", i, data, wantBytes[i])
		}
	}

	// Pages() metadata listing must agree with Page() and not decode.
	if pages := src.Pages(); len(pages) != 3 || pages[2].FileName != "p10.jpg" {
		t.Fatalf("Pages() = %+v", src.Pages())
	}

	// Sidecar present.
	if _, ok := src.Sidecar(); !ok {
		t.Fatal("expected ComicInfo.xml sidecar")
	}
}
