package ulid

import (
	"strings"
	"testing"
	"time"
)

func TestNewFormat(t *testing.T) {
	id := New()
	if len(id) != 26 {
		t.Fatalf("ULID length = %d, want 26 (%q)", len(id), id)
	}
	for i, c := range id {
		if !strings.ContainsRune(crockford, c) {
			t.Fatalf("ULID has non-Crockford char %q at %d (%q)", c, i, id)
		}
	}
	// First char carries only 2 bits, so it is one of 0..7.
	if id[0] < '0' || id[0] > '7' {
		t.Fatalf("ULID first char = %q, want 0..7 (%q)", id[0], id)
	}
}

func TestUniqueness(t *testing.T) {
	const n = 10000
	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		id := New()
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate ULID generated: %q", id)
		}
		seen[id] = struct{}{}
	}
}

func TestTimeOrdering(t *testing.T) {
	earlier := New()
	time.Sleep(2 * time.Millisecond)
	later := New()
	if !(earlier < later) {
		t.Fatalf("expected earlier ULID %q to sort before later %q", earlier, later)
	}
}
