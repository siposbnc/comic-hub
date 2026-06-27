package archive

import "testing"

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

// TestCBRDecode is deferred: exercising the RAR decode path needs a real .cbr fixture,
// and there is no Go RAR writer (the format is proprietary, read-only here). The binary
// fixture lands with the scanner's sample-archive corpus (docs/04-server.md §10), at
// which point this asserts page ordering + sidecar parity with CBZ.
func TestCBRDecode(t *testing.T) {
	t.Skip("needs a .cbr fixture — added with the scanner corpus in S3")
}
