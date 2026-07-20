package scanner

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/siposbnc/comic-hub/server/internal/domain"
)

var (
	// "variant" / "var" / "cvr" as whole words — safe markers even on a numbered file
	// (a genuine issue rarely carries these), so a "Batman 001 (Variant)" is caught.
	reVariantFile = regexp.MustCompile(`(?i)(\bvariant\b|\bcvr\b|\bvar\b)`)
	// "cover(s)" is only trusted when the file has no resolvable issue number, so legit
	// titles ("Undercover 003") aren't mistaken for cover art.
	reCoverFile = regexp.MustCompile(`(?i)\bcovers?\b`)
)

// ClassifyKind decides a book's kind, in priority order: an explicit ComicInfo <Format>,
// then the parsed issue-number label ("Annual 2", "One-Shot", …), then filename heuristics
// for variant/cover art. Defaults to a normal issue.
func ClassifyKind(number, filePath, ciFormat string) domain.BookKind {
	if k := kindFromFormat(ciFormat); k != "" {
		return k
	}
	if k := kindFromNumberLabel(number); k != "" {
		return k
	}
	base := strings.ToLower(filepath.Base(filePath))
	if reVariantFile.MatchString(base) {
		return domain.KindVariant
	}
	if number == "" && reCoverFile.MatchString(base) {
		return domain.KindCover
	}
	// A labeled number the explicit tables don't know ("Futures End 1", lettered issues):
	// SortNumber already files it after the numbered run, so the kind must agree — a plain
	// "issue" here would collide with the real issue of the same trailing number.
	if number != "" && SortNumber(number) >= specialBase {
		return domain.KindSpecial
	}
	return domain.KindIssue
}

// kindFromFormat maps a ComicInfo <Format> string to a kind ("" when it says nothing about
// kind, e.g. "Digital"/"Web").
func kindFromFormat(format string) domain.BookKind {
	f := strings.ToLower(strings.TrimSpace(format))
	switch {
	case f == "":
		return ""
	case strings.Contains(f, "annual"):
		return domain.KindAnnual
	case strings.Contains(f, "one-shot"), strings.Contains(f, "one shot"), strings.Contains(f, "oneshot"):
		return domain.KindOneShot
	case strings.Contains(f, "trade"), f == "tpb":
		return domain.KindTPB
	case strings.Contains(f, "graphic novel"), f == "gn":
		return domain.KindGN
	case strings.Contains(f, "variant"):
		return domain.KindVariant
	case strings.Contains(f, "cover"):
		return domain.KindCover
	case strings.Contains(f, "special"):
		return domain.KindSpecial
	default:
		return ""
	}
}

// kindFromNumberLabel classifies from the special word the filename parser folds into the
// issue number ("Annual 2", "One-Shot", "TPB", "GN", "Special").
func kindFromNumberLabel(number string) domain.BookKind {
	n := strings.ToLower(strings.TrimSpace(number))
	switch {
	case strings.HasPrefix(n, "annual"):
		return domain.KindAnnual
	case strings.HasPrefix(n, "one-shot"), strings.HasPrefix(n, "one shot"):
		return domain.KindOneShot
	case strings.HasPrefix(n, "tpb"):
		return domain.KindTPB
	case strings.HasPrefix(n, "gn"):
		return domain.KindGN
	case strings.HasPrefix(n, "special"):
		return domain.KindSpecial
	default:
		return ""
	}
}
