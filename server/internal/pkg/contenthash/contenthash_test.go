package contenthash

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, data []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "f.bin")
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return p
}

func TestOfFileFullDeterministic(t *testing.T) {
	p := write(t, []byte("hello comic"))
	a, err := OfFile(p, 0)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	b, _ := OfFile(p, 0)
	if a != b {
		t.Fatalf("non-deterministic: %s != %s", a, b)
	}
	if len(a) != 16 {
		t.Fatalf("hash len = %d, want 16 hex chars", len(a))
	}
}

func TestOfFileDistinguishesContent(t *testing.T) {
	p1 := write(t, []byte("page one"))
	p2 := write(t, []byte("page two"))
	h1, _ := OfFile(p1, 0)
	h2, _ := OfFile(p2, 0)
	if h1 == h2 {
		t.Fatal("different content hashed to the same value")
	}
}

func TestOfFileSampledPath(t *testing.T) {
	// Force the sampled branch with a tiny threshold; still must be deterministic and
	// distinguish content of differing length.
	p := write(t, make([]byte, 4096))
	a, err := OfFile(p, 16)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	b, _ := OfFile(p, 16)
	if a != b {
		t.Fatalf("sampled hash non-deterministic: %s != %s", a, b)
	}
	bigger := write(t, make([]byte, 8192))
	c, _ := OfFile(bigger, 16)
	if a == c {
		t.Fatal("files of different size hashed equal under sampling")
	}
}
