package domain

import "context"

// Collection is a curated, ordered set of books (a shared "shelf"), distinct from a
// per-user reading list. Items are positioned by a fractional `position` so a single book
// can be reordered without renumbering the rest.
type Collection struct {
	ID          string
	Name        string
	Description string
	CoverBookID string
	OwnerID     string
	BookCount   int // populated by list/get reads; not a stored column
	CreatedAt   int64
	UpdatedAt   int64
}

// CollectionItem is a book's membership in a collection, with its sort position.
type CollectionItem struct {
	BookID   string
	Position float64
}

// CollectionRepository persists collections and their ordered items.
type CollectionRepository interface {
	Create(ctx context.Context, c Collection) (Collection, error)
	Get(ctx context.Context, id string) (Collection, error)
	List(ctx context.Context) ([]Collection, error)
	// Update writes the editable fields (name, description, cover, updated_at).
	Update(ctx context.Context, c Collection) error
	Delete(ctx context.Context, id string) error

	// Items returns the collection's items ordered by position.
	Items(ctx context.Context, collectionID string) ([]CollectionItem, error)
	// AddItems appends books not already present, after the current last item.
	AddItems(ctx context.Context, collectionID string, bookIDs []string) error
	RemoveItem(ctx context.Context, collectionID, bookID string) error
	// SetPosition repositions one existing item.
	SetPosition(ctx context.Context, collectionID, bookID string, position float64) error
	// IDsForBook returns the ids of collections that already contain a book.
	IDsForBook(ctx context.Context, bookID string) ([]string, error)
}
