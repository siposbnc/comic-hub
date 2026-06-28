package domain

import "context"

// Bookmark is a user's saved place in a book: a page with an optional short note.
// At most one bookmark exists per page (per user+book), so the reader can toggle the
// current page on/off. See docs/06-reader.md §6.
type Bookmark struct {
	ID        string
	UserID    string
	BookID    string
	Page      int
	Note      string
	CreatedAt int64
	UpdatedAt int64
}

// BookmarkRepository persists per-user, per-book bookmarks. Every method is scoped to
// the owning user; reads/writes for a bookmark not owned by userID return ErrNotFound.
type BookmarkRepository interface {
	// List returns a book's bookmarks for the user, ordered by page ascending.
	List(ctx context.Context, userID, bookID string) ([]Bookmark, error)
	// Get resolves a bookmark by id (ErrNotFound if absent or not owned by the user).
	Get(ctx context.Context, userID, id string) (Bookmark, error)
	// GetByPage resolves the bookmark at a page, so the service can toggle/upsert it
	// (ErrNotFound if the page is not bookmarked).
	GetByPage(ctx context.Context, userID, bookID string, page int) (Bookmark, error)
	// Create inserts a new bookmark.
	Create(ctx context.Context, b Bookmark) (Bookmark, error)
	// UpdateNote replaces a bookmark's note and bumps updated_at.
	UpdateNote(ctx context.Context, userID, id, note string, updatedAt int64) (Bookmark, error)
	// Delete removes a bookmark owned by the user (no error if already gone).
	Delete(ctx context.Context, userID, id string) error
}
