package domain

import "context"

// SeriesSearchHit is a series matched by full-text search.
type SeriesSearchHit struct {
	ID          string
	Name        string
	Year        int
	CoverBookID string
}

// BookSearchHit is a book matched by full-text search, carrying its series name for display.
type BookSearchHit struct {
	ID         string
	SeriesID   string
	SeriesName string
	Number     string
	Title      string
	Format     string
}

// SearchRepository runs full-text queries over the catalog. The `match` argument is a
// ready-to-use FTS5 MATCH expression (the service builds it from the user's raw query); an
// empty libraryID spans all libraries.
type SearchRepository interface {
	SearchSeries(ctx context.Context, libraryID, match string, limit int) ([]SeriesSearchHit, error)
	SearchBooks(ctx context.Context, libraryID, match string, limit int) ([]BookSearchHit, error)
}
