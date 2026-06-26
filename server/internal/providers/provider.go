// Package providers defines the boundary for external metadata sources (Comic Vine,
// GCD, Metron, AniList). API keys live server-side only. See docs/04-server.md §6.
// Concrete providers and the matching engine arrive in Phase 2.
package providers

import "context"

// SeriesCandidate is a possible series match from a provider.
type SeriesCandidate struct {
	ProviderID string
	Name       string
	Year       int
	Publisher  string
	IssueCount int
	CoverURL   string
	Score      float64 // matcher confidence 0..1
}

// IssueCandidate is a possible issue match from a provider.
type IssueCandidate struct {
	ProviderID string
	Number     string
	Title      string
	CoverURL   string
}

// IssueMeta is the full metadata for a matched issue.
type IssueMeta struct {
	Title       string
	Number      string
	Volume      int
	Summary     string
	ReleaseDate int64
	AgeRating   string
	People      map[string][]string // role -> names
	Genres      []string
	Characters  []string
}

// Provider is an external metadata source.
type Provider interface {
	// Name identifies the provider (e.g. "comicvine").
	Name() string
	// SearchSeries finds candidate series for a query.
	SearchSeries(ctx context.Context, query string) ([]SeriesCandidate, error)
	// Issues lists issues for a matched series.
	Issues(ctx context.Context, seriesProviderID string) ([]IssueCandidate, error)
	// Issue fetches full metadata for one issue.
	Issue(ctx context.Context, issueProviderID string) (IssueMeta, error)
}
