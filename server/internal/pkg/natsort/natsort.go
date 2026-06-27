// Package natsort provides natural ("human") ordering of strings, where embedded
// runs of digits compare by numeric value so "page2" sorts before "page10". It is
// used to order the image entries inside a comic archive into reading order (see
// docs/04-server.md §3.1) and is reused by the scanner.
package natsort

import (
	"sort"
	"strings"
)

// Less reports whether a should sort before b in natural order. Comparison is
// case-insensitive, with a case-sensitive tiebreak for deterministic ordering.
func Less(a, b string) bool { return compare(a, b) < 0 }

// SliceStable sorts s in place using natural ordering, preserving the order of
// elements that compare equal.
func SliceStable(s []string) {
	sort.SliceStable(s, func(i, j int) bool { return Less(s[i], s[j]) })
}

func compare(a, b string) int {
	la, lb := strings.ToLower(a), strings.ToLower(b)
	i, j := 0, 0
	for i < len(la) && j < len(lb) {
		if isDigit(la[i]) && isDigit(lb[j]) {
			// Consume both digit runs and compare them numerically.
			ni := i
			for ni < len(la) && isDigit(la[ni]) {
				ni++
			}
			nj := j
			for nj < len(lb) && isDigit(lb[nj]) {
				nj++
			}
			// Strip leading zeros: more digits => larger number (same-length => lexical).
			da := strings.TrimLeft(la[i:ni], "0")
			db := strings.TrimLeft(lb[j:nj], "0")
			switch {
			case len(da) != len(db):
				if len(da) < len(db) {
					return -1
				}
				return 1
			case da != db:
				if da < db {
					return -1
				}
				return 1
			case (ni - i) != (nj - j):
				// Equal value but different zero-padding: fewer chars first, for stability.
				if (ni - i) < (nj - j) {
					return -1
				}
				return 1
			}
			i, j = ni, nj
			continue
		}
		if la[i] != lb[j] {
			if la[i] < lb[j] {
				return -1
			}
			return 1
		}
		i++
		j++
	}
	switch {
	case len(la)-i < len(lb)-j:
		return -1
	case len(la)-i > len(lb)-j:
		return 1
	}
	// Case-insensitively equal; fall back to a stable case-sensitive comparison.
	return strings.Compare(a, b)
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }
