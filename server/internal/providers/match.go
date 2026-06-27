package providers

import (
	"regexp"
	"sort"
	"strings"
)

// LocalSeries is the catalog series we are trying to match to a provider's series. Year
// and IssueCount are optional signals (0 = unknown) gleaned from the scan.
type LocalSeries struct {
	Name       string
	Year       int
	IssueCount int
}

// Scoring weights. Name dominates; year and issue-count refine. When a signal is missing
// on either side it is skipped and its weight redistributed (see ScoreSeries).
const (
	weightName  = 0.7
	weightYear  = 0.2
	weightCount = 0.1
)

var (
	reParens     = regexp.MustCompile(`\([^)]*\)`)                    // "(2016)", "(of 6)"
	reVolume     = regexp.MustCompile(`(?i)\bvol(?:ume|\.)?\s*\d+\b`) // "Vol. 3", "volume 2"
	reIssueRange = regexp.MustCompile(`#?\b\d{1,4}\s*-\s*\d{1,4}\b`)  // "013-028", "#1-6"
	reTrailNum   = regexp.MustCompile(`\s+#?\d{1,4}\s*$`)             // trailing "001", "#5"
	reNonAlnum   = regexp.MustCompile(`[^a-z0-9]+`)
	reSpaceRuns  = regexp.MustCompile(`\s+`)
)

// CleanQuery prepares a series name for a provider name/free-text search: it drops the
// "(year)" / "Vol. N" qualifiers and the issue ranges and numbers a scanned folder name
// carries, then collapses whitespace — so "Batman (2016)", "Batman Vol. 3", and
// "Wonder Woman 013-028 (2017)" search as "Batman" / "Wonder Woman". Casing, word order,
// and punctuation are preserved (providers may substring-match, where "X-Men" must stay
// hyphenated). Returns the original (trimmed) string if cleaning would leave it empty.
func CleanQuery(name string) string {
	s := reParens.ReplaceAllString(name, " ")
	s = reVolume.ReplaceAllString(s, " ")
	s = reIssueRange.ReplaceAllString(s, " ")
	s = reTrailNum.ReplaceAllString(s, " ")
	s = strings.TrimSpace(reSpaceRuns.ReplaceAllString(s, " "))
	if s == "" {
		return strings.TrimSpace(name)
	}
	return s
}

// ScoreSeries returns a 0..1 confidence that the candidate is the same series as local.
// It blends normalized name similarity with publication-year and issue-count proximity,
// skipping (and redistributing the weight of) any signal that is unknown on either side.
func ScoreSeries(local LocalSeries, c SeriesCandidate) float64 {
	sum := weightName * nameSimilarity(local.Name, c.Name)
	total := weightName

	if local.Year > 0 && c.Year > 0 {
		sum += weightYear * yearProximity(local.Year, c.Year)
		total += weightYear
	}
	if local.IssueCount > 0 && c.IssueCount > 0 {
		sum += weightCount * countProximity(local.IssueCount, c.IssueCount)
		total += weightCount
	}
	if total == 0 {
		return 0
	}
	return sum / total
}

// RankSeries scores each candidate (filling its Score) and returns a copy sorted
// best-first. The input slice is not modified.
func RankSeries(local LocalSeries, cands []SeriesCandidate) []SeriesCandidate {
	out := make([]SeriesCandidate, len(cands))
	copy(out, cands)
	for i := range out {
		out[i].Score = ScoreSeries(local, out[i])
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	return out
}

// nameSimilarity is a token Dice coefficient over normalized series names: exact match is
// 1.0, no shared tokens is 0.0. Robust to punctuation, casing, a leading article, and the
// "(year)" / "Vol. N" qualifiers that year/count already capture.
func nameSimilarity(a, b string) float64 {
	ta, tb := normalizeName(a), normalizeName(b)
	if len(ta) == 0 || len(tb) == 0 {
		return 0
	}
	if strings.Join(ta, " ") == strings.Join(tb, " ") {
		return 1
	}

	setA := make(map[string]struct{}, len(ta))
	for _, t := range ta {
		setA[t] = struct{}{}
	}
	setB := make(map[string]struct{}, len(tb))
	inter := 0
	for _, t := range tb {
		if _, seen := setB[t]; seen {
			continue
		}
		setB[t] = struct{}{}
		if _, ok := setA[t]; ok {
			inter++
		}
	}
	return 2 * float64(inter) / float64(len(setA)+len(setB))
}

// normalizeName lowercases, strips parenthetical and volume qualifiers and punctuation,
// drops a leading article, and returns the remaining tokens.
func normalizeName(s string) []string {
	s = strings.ToLower(s)
	s = reParens.ReplaceAllString(s, " ")
	s = reVolume.ReplaceAllString(s, " ")
	s = reNonAlnum.ReplaceAllString(s, " ")
	s = strings.TrimSpace(reSpaceRuns.ReplaceAllString(s, " "))
	if s == "" {
		return nil
	}
	fields := strings.Fields(s)
	if len(fields) > 1 && fields[0] == "the" {
		fields = fields[1:]
	}
	return fields
}

// yearProximity rewards an exact publication year and tolerates the off-by-one/two drift
// common between catalog heuristics and provider data (relaunches score 0).
func yearProximity(a, b int) float64 {
	d := a - b
	if d < 0 {
		d = -d
	}
	switch d {
	case 0:
		return 1
	case 1:
		return 0.6
	case 2:
		return 0.3
	default:
		return 0
	}
}

// countProximity is the ratio of the smaller issue count to the larger (1.0 when equal).
func countProximity(a, b int) float64 {
	lo, hi := a, b
	if lo > hi {
		lo, hi = hi, lo
	}
	if hi == 0 {
		return 0
	}
	return float64(lo) / float64(hi)
}
