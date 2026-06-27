package archive

import "testing"

func TestRegistrySupports(t *testing.T) {
	r := DefaultRegistry()
	cases := map[string]bool{
		`C:\comics\Saga 001.cbz`: true,
		"saga.CBZ":               true, // case-insensitive
		"saga.zip":               true,
		"saga.cbr":               true,
		"saga.rar":               true,
		"saga.cb7":               true,
		"saga.7z":                true,
		"saga.cbt":               true,
		"saga.tar":               true,
		"saga.pdf":               false, // not yet (Phase 2)
		"saga.txt":               false,
		"noext":                  false,
	}
	for path, want := range cases {
		if got := r.Supports(path); got != want {
			t.Errorf("Supports(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestRegistryOpenUnsupported(t *testing.T) {
	if _, err := DefaultRegistry().Open("book.pdf"); err == nil {
		t.Fatal("expected error opening unsupported format")
	}
}
