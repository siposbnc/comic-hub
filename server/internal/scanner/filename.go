// Package scanner turns a folder tree into catalog rows: it walks library roots,
// classifies and change-detects files, parses archives (via the archive registry) and
// their metadata, and upserts series/books/pages. See docs/04-server.md §3.
package scanner

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// ParsedName is the metadata derived from a file's name + folder when no ComicInfo.xml
// is present (docs/04-server.md §3.2). Fields are best-effort; the scanner overrides
// them with sidecar/online metadata where available.
type ParsedName struct {
	Series string
	Number string // normalized, e.g. "1", "1.5", "Annual 2" (empty if unknown)
	Volume int
	Year   int
}

var (
	reYear          = regexp.MustCompile(`\((\d{4})\)`)
	reVolume        = regexp.MustCompile(`(?i)\bv(?:ol(?:ume)?)?\.?\s*(\d{1,4})\b`)
	reParenBracket  = regexp.MustCompile(`[\(\[][^\)\]]*[\)\]]`)
	reTrailingIssue = regexp.MustCompile(`(?i)^(.*\S)\s+#?(\d{1,5}(?:\.\d+)?)$`)
	reJustNumber    = regexp.MustCompile(`^#?(\d{1,5}(?:\.\d+)?)$`)
	reSpecialTail   = regexp.MustCompile(`(?i)^(.*?)\s+(annual|special|one[- ]?shot|tpb|gn)$`)
	reLeadingZeros  = regexp.MustCompile(`^0+(\d)`)
	// Decimal issue number mid-name with a subtitle after it (New 52 villain-month style:
	// "Wonder Woman 023.1 - Cheetah"). Only decimals — a bare integer mid-name is far more
	// likely part of the series ("Spider-Man 2099") than an issue number, but a decimal
	// never is. Tried before the trailing rule so the subtitle's own trailing digits
	// ("… - Cheetah 001") don't win.
	reDecimalIssue = regexp.MustCompile(`^(.*\S)\s+#?(\d{1,5}\.\d+)(?:\s*[-–—]\s*|\s+)\S.*$`)
)

// ParseFilename derives series/number/volume/year from a file path. folder (the file's
// parent directory) supplies the series name when the filename alone has none.
func ParseFilename(filePath string) ParsedName {
	base := filepath.Base(filePath)
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	folder := filepath.Base(filepath.Dir(filePath))

	work := collapse(strings.ReplaceAll(stem, "_", " "))

	var p ParsedName

	// Year: the first parenthesized 4-digit group.
	if m := reYear.FindStringSubmatch(work); m != nil {
		p.Year, _ = strconv.Atoi(m[1])
	}
	// Volume: vN / vol N / volume N.
	if m := reVolume.FindStringSubmatch(work); m != nil {
		p.Volume, _ = strconv.Atoi(m[1])
		work = reVolume.ReplaceAllString(work, " ")
	}
	// Drop all parenthesized/bracketed noise (year, scan group, "Digital", …).
	work = collapse(reParenBracket.ReplaceAllString(work, " "))

	series := work
	number := ""
	if m := reDecimalIssue.FindStringSubmatch(work); m != nil {
		series = strings.TrimSpace(m[1])
		number = normalizeNumber(m[2])
	} else if m := reTrailingIssue.FindStringSubmatch(work); m != nil {
		series = strings.TrimSpace(m[1])
		number = normalizeNumber(m[2])
	} else if m := reJustNumber.FindStringSubmatch(work); m != nil {
		// The whole name is just an issue number (series comes from the folder).
		series = ""
		number = normalizeNumber(m[1])
	}

	// A trailing special word ("Annual", …) belongs with the number, not the series.
	if m := reSpecialTail.FindStringSubmatch(series); m != nil {
		special := titleCase(m[2])
		series = strings.TrimSpace(m[1])
		if number != "" {
			number = special + " " + number
		} else {
			number = special
		}
	}

	series = cleanTitle(series)
	if series == "" {
		series = cleanTitle(folder)
	}

	p.Series = series
	p.Number = number
	return p
}

func collapse(s string) string {
	return strings.TrimSpace(strings.Join(strings.Fields(s), " "))
}

// cleanTitle trims trailing separators left behind by token removal.
func cleanTitle(s string) string {
	return strings.TrimRight(collapse(s), " -–—_.")
}

func titleCase(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "one-shot", "one shot", "oneshot":
		return "One-Shot"
	case "tpb":
		return "TPB"
	case "gn":
		return "GN"
	}
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// normalizeNumber strips insignificant leading zeros ("001" -> "1", "0" -> "0").
func normalizeNumber(n string) string {
	return reLeadingZeros.ReplaceAllString(n, "$1")
}
