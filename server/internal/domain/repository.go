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
	Jobs() JobRepository
	Metadata() MetadataRepository
	Search() SearchRepository
	Collections() CollectionRepository
	ReadingLists() ReadingListRepository
	Tags() TagRepository
	SmartLists() SmartListRepository
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
	// GetByFolder resolves a series by its source folder within a library, so the
	// scanner can reuse an existing series id across rescans (ErrNotFound if absent).
	GetByFolder(ctx context.Context, libraryID, folderPath string) (Series, error)
	ListByLibrary(ctx context.Context, libraryID string) ([]Series, error)
	// Summaries lists a library's series with grid aggregates (book/read counts +
	// resolved cover book) for the given user, in one query.
	Summaries(ctx context.Context, libraryID, userID string) ([]SeriesSummary, error)
}

// BookRepository persists books and their pages.
type BookRepository interface {
	Upsert(ctx context.Context, b Book) (Book, error)
	Get(ctx context.Context, id string) (Book, error)
	// GetByPath resolves a book by its absolute file path, so the scanner can
	// change-detect and reuse ids across rescans (ErrNotFound if absent).
	GetByPath(ctx context.Context, filePath string) (Book, error)
	ReplacePages(ctx context.Context, bookID string, pages []Page) error
	// ListPages returns a book's pages in index order (the reader's manifest source).
	ListPages(ctx context.Context, bookID string) ([]Page, error)
	ListBySeries(ctx context.Context, seriesID string) ([]Book, error)
	// ListRecent returns the most recently added books, newest first. An empty
	// libraryID spans all libraries.
	ListRecent(ctx context.Context, libraryID string, limit int) ([]Book, error)
}

// ProgressRepository persists per-user reading progress.
type ProgressRepository interface {
	Upsert(ctx context.Context, p Progress) (Progress, error)
	Get(ctx context.Context, userID, bookID string) (Progress, error)
	ContinueReading(ctx context.Context, userID string, limit int) ([]Progress, error)
}
