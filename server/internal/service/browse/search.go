package browse

import (
	"context"
	"regexp"
	"strings"

	"github.com/siposbnc/comic-hub/server/internal/access"
)

// SearchHitType groups search results by kind.
const (
	SearchSeries = "series"
	SearchBook   = "book"
	SearchAll    = "all"
)

// defaultSearchLimit caps each result group; tuned for a type-ahead dropdown.
const defaultSearchLimit = 10

// SeriesHit is a series in a search result.
type SeriesHit struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Year        int    `json:"year,omitempty"`
	CoverBookID string `json:"coverBookId,omitempty"`
}

// BookHit is a book in a search result.
type BookHit struct {
	ID         string `json:"id"`
	SeriesID   string `json:"seriesId"`
	SeriesName string `json:"seriesName,omitempty"`
	Number     string `json:"number,omitempty"`
	Title      string `json:"title,omitempty"`
	Format     string `json:"format"`
}

// SearchResults is the grouped search payload.
type SearchResults struct {
	Query  string      `json:"query"`
	Series []SeriesHit `json:"series"`
	Books  []BookHit   `json:"books"`
}

// reSearchToken matches a run of letters or digits; everything else (FTS5 operators,
// punctuation) is treated as a separator and dropped, so user input can never form a
// malformed MATCH expression.
var reSearchToken = regexp.MustCompile(`[\p{L}\p{N}]+`)

// buildMatch turns a raw user query into a safe FTS5 MATCH expression: each word becomes a
// prefix term (so "bat" matches "Batman") AND-ed together. Returns "" when there's nothing
// searchable, which the caller treats as an empty result.
func buildMatch(query string) string {
	tokens := reSearchToken.FindAllString(query, -1)
	if len(tokens) == 0 {
		return ""
	}
	for i, t := range tokens {
		tokens[i] = t + "*"
	}
	return strings.Join(tokens, " ")
}

// Search runs a full-text query across series and books, grouped and ranked best-first.
// typeFilter is one of SearchAll/SearchSeries/SearchBook ("" means all); limit<=0 uses a
// type-ahead default.
func (s *Service) Search(ctx context.Context, libraryID, query, typeFilter string, limit int) (SearchResults, error) {
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	out := SearchResults{Query: query, Series: []SeriesHit{}, Books: []BookHit{}}

	match := buildMatch(query)
	if match == "" {
		return out, nil
	}

	wantSeries := typeFilter == "" || typeFilter == SearchAll || typeFilter == SearchSeries
	wantBooks := typeFilter == "" || typeFilter == SearchAll || typeFilter == SearchBook

	if wantSeries {
		hits, err := s.repo.Search().SearchSeries(ctx, libraryID, match, limit)
		if err != nil {
			return SearchResults{}, err
		}
		for _, h := range hits {
			out.Series = append(out.Series, SeriesHit{
				ID:          h.ID,
				Name:        h.Name,
				Year:        h.Year,
				CoverBookID: h.CoverBookID,
			})
		}
	}
	if wantBooks {
		hits, err := s.repo.Search().SearchBooks(ctx, libraryID, match, limit)
		if err != nil {
			return SearchResults{}, err
		}
		ceiling := access.CeilingFrom(ctx)
		for _, h := range hits {
			// Drop hits above a restricted user's content ceiling (search results expose
			// titles, so they must respect the restriction too).
			if ceiling != "" {
				if b, err := s.repo.Books().Get(ctx, h.ID); err != nil ||
					!access.Allowed(ceiling, b.AgeRating) {
					continue
				}
			}
			out.Books = append(out.Books, BookHit{
				ID:         h.ID,
				SeriesID:   h.SeriesID,
				SeriesName: h.SeriesName,
				Number:     h.Number,
				Title:      h.Title,
				Format:     h.Format,
			})
		}
	}
	return out, nil
}
