package domain

import "context"

// ReadingList is a personal, ordered list of books owned by one user — the per-user
// counterpart to a (shared) Collection. Items use the same fractional positioning.
type ReadingList struct {
	ID        string
	UserID    string
	Name      string
	BookCount int // populated by reads; not a stored column
	CreatedAt int64
	UpdatedAt int64
}

// ReadingListItem is a book's membership in a reading list, with its sort position.
type ReadingListItem struct {
	BookID   string
	Position float64
}

// ReadingListRepository persists per-user reading lists and their ordered items. Every
// method is scoped to the owning user; reads/writes for a list not owned by userID return
// ErrNotFound.
type ReadingListRepository interface {
	Create(ctx context.Context, l ReadingList) (ReadingList, error)
	Get(ctx context.Context, userID, id string) (ReadingList, error)
	List(ctx context.Context, userID string) ([]ReadingList, error)
	Update(ctx context.Context, l ReadingList) error
	Delete(ctx context.Context, userID, id string) error

	Items(ctx context.Context, listID string) ([]ReadingListItem, error)
	AddItems(ctx context.Context, listID string, bookIDs []string) error
	RemoveItem(ctx context.Context, listID, bookID string) error
	SetPosition(ctx context.Context, listID, bookID string, position float64) error
}
