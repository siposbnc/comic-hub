// Package access enforces per-user content restrictions: a "restricted" user has an
// age-rating ceiling, and books rated above it are hidden from listings and refused by the
// reader. The acting user's ceiling rides on the request context so services can apply it
// without threading it through every signature.
package access

import (
	"context"
	"strings"
)

// tier maps a comic age rating to an ordinal severity. Unrated / unknown values are tier 0
// (always allowed), so a restriction never hides an unrated library. Values cover the
// ComicInfo AgeRating vocabulary plus common synonyms.
var tier = map[string]int{
	"early childhood": 1,
	"everyone":        1,
	"g":               1,
	"kids to adults":  1,
	"everyone 10+":    2,
	"pg":              2,
	"teen":            3,
	"t":               3,
	"m":               4,
	"ma15+":           4,
	"mature 17+":      4,
	"mature":          4,
	"adults only 18+": 5,
	"r18+":            5,
	"x18+":            5,
	"adult":           5,
}

// Tier returns the severity of a rating (0 for unrated/unknown).
func Tier(rating string) int {
	return tier[strings.ToLower(strings.TrimSpace(rating))]
}

// Allowed reports whether content with the given rating is viewable under the ceiling. An
// empty ceiling means unrestricted (every rating allowed).
func Allowed(ceiling, rating string) bool {
	if strings.TrimSpace(ceiling) == "" {
		return true
	}
	return Tier(rating) <= Tier(ceiling)
}

type ctxKey int

const ceilingKey ctxKey = iota

// WithCeiling attaches the acting user's age-rating ceiling to the context.
func WithCeiling(ctx context.Context, ceiling string) context.Context {
	return context.WithValue(ctx, ceilingKey, ceiling)
}

// CeilingFrom returns the ceiling set on the context (empty = unrestricted).
func CeilingFrom(ctx context.Context) string {
	c, _ := ctx.Value(ceilingKey).(string)
	return c
}
