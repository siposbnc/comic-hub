// Package providers defines the boundary for external metadata sources (Comic Vine,
// GCD, Metron, AniList). API keys live server-side only. See docs/04-server.md §6.
// Concrete providers and the matching engine arrive in Phase 2.
package providers

import (
	"context"
	"errors"
)

// ErrRateLimited marks a provider request rejected by the upstream rate limit (HTTP 429
// or equivalent) after the client's own retries were exhausted. The matching service uses
// it to leave a series resumable instead of failing opaquely.
var ErrRateLimited = errors.New("provider rate limited")

// SeriesCandidate is a possible series match from a provider.
type SeriesCandidate struct {
	ProviderID string  `json:"providerId"`
	Provider   string  `json:"provider"` // source provider name (e.g. "comicvine", "metron")
	Name       string  `json:"name"`
	Year       int     `json:"year"`
	Publisher  string  `json:"publisher"`
	IssueCount int     `json:"issueCount"`
	CoverURL   string  `json:"coverUrl"`
	Score      float64 `json:"score"` // matcher confidence 0..1
}

// IssueCandidate is a possible issue match from a provider.
type IssueCandidate struct {
	ProviderID string
	Number     string
	Title      string
	CoverURL   string
}

// SeriesMeta is the series-level (provider "volume") metadata for a matched series.
type SeriesMeta struct {
	Name        string
	Year        int
	Publisher   string
	Description string
	CoverURL    string
	Genres      []string
}

// ArcRef is a reference to a story arc an issue belongs to (id + display name).
type ArcRef struct {
	ProviderID string
	Name       string
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
	StoryArcs   []ArcRef
}

// Provider is an external metadata source.
type Provider interface {
	// Name identifies the provider (e.g. "comicvine").
	Name() string
	// SearchSeries finds candidate series for a query.
	SearchSeries(ctx context.Context, query string) ([]SeriesCandidate, error)
	// SeriesMeta fetches series-level detail (description, publisher, …) for a matched series.
	SeriesMeta(ctx context.Context, seriesProviderID string) (SeriesMeta, error)
	// Issues lists issues for a matched series.
	Issues(ctx context.Context, seriesProviderID string) ([]IssueCandidate, error)
	// Issue fetches full metadata for one issue.
	Issue(ctx context.Context, issueProviderID string) (IssueMeta, error)
}
