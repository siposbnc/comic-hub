package domain

import "context"

// ReadingList is a personal, ordered list of books owned by one user — the per-user
// counterpart to a (shared) Collection. Items use the same fractional positioning.
type ReadingList struct {
	ID        string
	UserID    string
	Name      string
	Active    bool // the user's current reading queue (at most one)
	BookCount int  // populated by reads; not a stored column
	CreatedAt int64
	UpdatedAt int64
}

// ReadingListItem is an entry in a reading list. A linked entry carries a display snapshot
// (series name, number, title, content hash) captured when the item was added, so the
// entry survives — and stays renderable — even after the underlying book is deleted.
// BookID is empty for such stale entries, and for placeholders added manually for issues
// not (yet) in the library. Stale entries keep the list's order intact but can't be read.
//
// An entry can instead be a collection reference: CollectionID is set (BookID empty) and
// the row stands in — as a single ordered slot — for the whole collection, expanded live
// into its current books when the list is read (rendered as a named group).
type ReadingListItem struct {
	ID           string
	BookID       string // empty = stale placeholder or collection reference
	CollectionID string // non-empty = a collection-reference entry
	Position     float64
	AddedAt      int64
	SeriesName   string
	Number       string
	Title        string
	ContentHash  string // hash of the book the entry pointed at; drives auto-relink on rescan
}

// IsCollection reports whether the entry references a whole collection (a live group).
func (it ReadingListItem) IsCollection() bool { return it.CollectionID != "" }

// Stale reports whether the entry has no backing book and is not a collection reference
// (i.e. a deleted book, or a manual placeholder).
func (it ReadingListItem) Stale() bool { return it.BookID == "" && it.CollectionID == "" }

// ManualListItem describes a placeholder entry for an issue that isn't in the library.
type ManualListItem struct {
	SeriesName string
	Number     string
	Title      string
}

// ReadingListRepository persists per-user reading lists and their ordered items. Every
// method is scoped to the owning user; reads/writes for a list not owned by userID return
// ErrNotFound. Item mutations that take a `ref` accept either an item id or a (linked)
// book id, so callers keyed on books keep working.
type ReadingListRepository interface {
	Create(ctx context.Context, l ReadingList) (ReadingList, error)
	Get(ctx context.Context, userID, id string) (ReadingList, error)
	List(ctx context.Context, userID string) ([]ReadingList, error)
	Update(ctx context.Context, l ReadingList) error
	Delete(ctx context.Context, userID, id string) error

	// SetActive marks one list active for the user, clearing any previous active list.
	SetActive(ctx context.Context, userID, id string) error
	// GetActive returns the user's active list (ErrNotFound if none).
	GetActive(ctx context.Context, userID string) (ReadingList, error)

	Items(ctx context.Context, listID string) ([]ReadingListItem, error)
	// ExpandedBookIDs returns the list's readable book ids in display order, with each
	// collection reference expanded inline into the collection's books (in collection
	// order). Stale placeholders are omitted. This is the flattened reading order.
	ExpandedBookIDs(ctx context.Context, listID string) ([]string, error)
	AddItems(ctx context.Context, listID string, bookIDs []string) error
	// AddCollectionRef appends a collection-reference entry (one ordered slot standing in
	// for the whole collection). No-op if the collection is already referenced by the list.
	AddCollectionRef(ctx context.Context, listID, collectionID string) error
	// AddManualItems appends stale placeholder entries (no backing book).
	AddManualItems(ctx context.Context, listID string, entries []ManualListItem) error
	RemoveItem(ctx context.Context, listID, ref string) error
	SetPosition(ctx context.Context, listID, ref string, position float64) error
	// Relink points an entry at a (new) book and refreshes its snapshot from that book.
	// ErrNotFound if the entry doesn't exist; ErrValidation if the book is already in the list.
	Relink(ctx context.Context, listID, itemID, bookID string) error
	// RelinkStaleByHash re-attaches stale entries (across all lists) whose snapshot hash
	// matches a book — called by the scanner when a book (re)appears. Returns rows relinked.
	RelinkStaleByHash(ctx context.Context, contentHash, bookID string) (int, error)
	// IDsForBook returns the ids of the user's reading lists that already contain a book.
	IDsForBook(ctx context.Context, userID, bookID string) ([]string, error)
}
