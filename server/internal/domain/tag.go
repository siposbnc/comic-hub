package domain

import "context"

// Tag is a free-form label applied to books (many-to-many). Color is an optional UI hint.
type Tag struct {
	ID        string
	Name      string
	Color     string
	BookCount int // populated by list reads; not a stored column
}

// TagRepository persists tags and their book assignments.
type TagRepository interface {
	Create(ctx context.Context, t Tag) (Tag, error)
	Get(ctx context.Context, id string) (Tag, error)
	// GetByName resolves a tag by its (unique) name, ErrNotFound if absent.
	GetByName(ctx context.Context, name string) (Tag, error)
	List(ctx context.Context) ([]Tag, error)
	Update(ctx context.Context, t Tag) error
	Delete(ctx context.Context, id string) error

	// AssignToBook tags a book with each tagID (idempotent; unknown tags are an error).
	AssignToBook(ctx context.Context, bookID string, tagIDs []string) error
	UnassignFromBook(ctx context.Context, bookID, tagID string) error
	// BookTags returns a book's tags, name-sorted.
	BookTags(ctx context.Context, bookID string) ([]Tag, error)
	// TaggedBookIDs returns the ids of books carrying a tag, newest-added first.
	TaggedBookIDs(ctx context.Context, tagID string) ([]string, error)
}
