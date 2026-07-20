package organize

import (
	"testing"

	"github.com/siposbnc/comic-hub/server/internal/domain"
)

func TestSpecialName(t *testing.T) {
	cases := []struct {
		number string
		kind   domain.BookKind
		want   string
	}{
		{"Futures End 1", domain.KindSpecial, "Futures End"},
		{"Convergence 1", domain.KindSpecial, "Convergence"},
		{"Convergence 2", domain.KindSpecial, "Convergence"},
		{"Annual 2", domain.KindAnnual, "Annual"},
		{"One-Shot", domain.KindOneShot, "One-Shot"},
		{"1", domain.KindSpecial, "Special"}, // bare number → generic kind label
		{"", domain.KindTPB, "TPB"},
	}
	for _, c := range cases {
		if got := specialName(c.number, c.kind); got != c.want {
			t.Errorf("specialName(%q, %q) = %q, want %q", c.number, c.kind, got, c.want)
		}
	}
}

func TestLocalSpecialNumber(t *testing.T) {
	cases := []struct {
		number string
		want   float64
	}{
		{"Futures End 1", 1},
		{"Convergence 2", 2},
		{"Annual 3", 3},
		{"One-Shot", 0}, // no trailing figure
	}
	for _, c := range cases {
		if got := localSpecialNumber(c.number); got != c.want {
			t.Errorf("localSpecialNumber(%q) = %v, want %v", c.number, got, c.want)
		}
	}
}

// Distinct named specials that each start at #1 must split into separate rows instead of
// colliding — the Wonder Woman "Futures End #1" vs "Convergence #1" case.
func TestGroupSpecialsSplitsNamedEditions(t *testing.T) {
	books := []domain.Book{
		{ID: "b1", Number: "Futures End 1", Kind: domain.KindSpecial, SortNumber: 1_000_001},
		{ID: "b2", Number: "Convergence 2", Kind: domain.KindSpecial, SortNumber: 1_000_002},
		{ID: "b3", Number: "Convergence 1", Kind: domain.KindSpecial, SortNumber: 1_000_001},
		{ID: "b4", Number: "Annual 1", Kind: domain.KindAnnual, SortNumber: 1_000_001},
	}
	groups := groupSpecials(books)

	// Annual sorts first (kind order), then the two KindSpecial editions alphabetically.
	if len(groups) != 3 {
		t.Fatalf("got %d groups, want 3 (Annual, Convergence, Futures End)", len(groups))
	}
	if groups[0].name != "Annual" || groups[1].name != "Convergence" || groups[2].name != "Futures End" {
		t.Fatalf("group order = %q/%q/%q", groups[0].name, groups[1].name, groups[2].name)
	}
	// Convergence keeps both issues, in number order.
	conv := groups[1]
	if len(conv.books) != 2 || conv.books[0].ID != "b3" || conv.books[1].ID != "b2" {
		t.Fatalf("Convergence books = %+v, want b3 then b2", conv.books)
	}
	// Futures End is its own single-issue row, not folded into Convergence.
	if len(groups[2].books) != 1 || groups[2].books[0].ID != "b1" {
		t.Fatalf("Futures End books = %+v, want just b1", groups[2].books)
	}
	// Keys are unique so the tracker row ids don't collide.
	if groups[1].key == groups[2].key {
		t.Fatalf("Convergence and Futures End share a key %q", groups[1].key)
	}
}
