package scanner

import (
	"testing"

	"github.com/siposbnc/comic-hub/server/internal/domain"
)

func TestClassifyKind(t *testing.T) {
	cases := []struct {
		number, path, format string
		want                 domain.BookKind
	}{
		// ComicInfo <Format> wins.
		{"1", `X\Batman 001.cbz`, "Annual", domain.KindAnnual},
		{"1", `X\Batman 001.cbz`, "Trade Paperback", domain.KindTPB},
		{"1", `X\Batman 001.cbz`, "Variant", domain.KindVariant},
		{"1", `X\Batman 001.cbz`, "Digital", domain.KindIssue}, // format says nothing about kind
		// Number label (from the filename parser).
		{"Annual 2", `X\Batman Annual 02.cbz`, "", domain.KindAnnual},
		{"One-Shot", `X\Batman Special.cbz`, "", domain.KindOneShot},
		{"TPB", `X\Batman TPB.cbz`, "", domain.KindTPB},
		// Filename heuristics: variant markers even on a numbered file.
		{"1", `X\Batman 001 (Variant).cbz`, "", domain.KindVariant},
		{"", `X\Batman 001 var.cbz`, "", domain.KindVariant},
		// "cover" only when unnumbered, so real titles aren't misread.
		{"", `X\Batman Covers.cbz`, "", domain.KindCover},
		{"3", `X\Undercover 003.cbz`, "", domain.KindIssue},
		// Plain issue.
		{"12", `X\Saga 012.cbz`, "", domain.KindIssue},
	}
	for _, c := range cases {
		if got := classifyKind(c.number, c.path, c.format); got != c.want {
			t.Errorf("classifyKind(%q, %q, %q) = %q, want %q", c.number, c.path, c.format, got, c.want)
		}
	}
}
