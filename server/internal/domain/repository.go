package domain

import "context"

// Repository is the storage boundary for the catalog. SQLite implements it today;
// Postgres can implement the same interface for large remote installs (ADR-005).
//
// This is intentionally a stub in Phase 0 — methods are added as services are built
// in Phase 1. Keeping the seam here lets the domain stay storage-agnostic from day one.
type Repository interface {
	Libraries() LibraryRepository
	Series() SeriesRepository
	Books() BookRepository
	Progress() ProgressRepository
}

// LibraryRepository persists libraries and their roots.
type LibraryRepository interface {
	Create(ctx context.Context, lib Library) (Library, error)
	Get(ctx context.Context, id string) (Library, error)
	List(ctx context.Context) ([]Library, error)
	Delete(ctx context.Context, id string) error
}

// SeriesRepository persists series.
type SeriesRepository interface {
	Upsert(ctx context.Context, s Series) (Series, error)
	Get(ctx context.Context, id string) (Series, error)
	ListByLibrary(ctx context.Context, libraryID string) ([]Series, error)
}

// BookRepository persists books and their pages.
type BookRepository interface {
	Upsert(ctx context.Context, b Book) (Book, error)
	Get(ctx context.Context, id string) (Book, error)
	ReplacePages(ctx context.Context, bookID string, pages []Page) error
	ListBySeries(ctx context.Context, seriesID string) ([]Book, error)
}

// ProgressRepository persists per-user reading progress.
type ProgressRepository interface {
	Upsert(ctx context.Context, p Progress) (Progress, error)
	Get(ctx context.Context, userID, bookID string) (Progress, error)
	ContinueReading(ctx context.Context, userID string, limit int) ([]Progress, error)
}
