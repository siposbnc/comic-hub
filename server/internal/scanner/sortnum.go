package scanner

import (
	"regexp"
	"strconv"
)

// specialBase pushes specials (Annual, one-shots, lettered issues) after all regular
// numbered issues while preserving a deterministic order among themselves.
const specialBase = 1_000_000

var (
	reLeadingFloat = regexp.MustCompile(`^(\d+(?:\.\d+)?)`)
	reTrailingNum  = regexp.MustCompile(`(\d+(?:\.\d+)?)\s*$`)
)

// SortNumber derives the numeric sort key (book.sort_number) from a messy issue
// "number" string. Leading-numeric issues sort by value ("1.MU" -> 1, "1.5" -> 1.5);
// specials ("Annual 2") sort after regular issues by their trailing number; anything
// unrecognized sorts last. See docs/02-data-model.md §5.
func SortNumber(number string) float64 {
	if number == "" {
		return 0
	}
	if m := reLeadingFloat.FindStringSubmatch(number); m != nil {
		f, _ := strconv.ParseFloat(m[1], 64)
		return f
	}
	if m := reTrailingNum.FindStringSubmatch(number); m != nil {
		f, _ := strconv.ParseFloat(m[1], 64)
		return specialBase + f
	}
	return specialBase
}
