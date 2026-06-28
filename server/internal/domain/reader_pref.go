package domain

import "context"

// ReaderPrefRepository persists per-user, per-book reader overrides. The settings value is
// an opaque JSON object the reader client defines; the server stores and returns it as-is.
type ReaderPrefRepository interface {
	// Get returns the stored settings JSON for a user+book, or ErrNotFound if none.
	Get(ctx context.Context, userID, bookID string) (string, error)
	// Put upserts the settings JSON for a user+book.
	Put(ctx context.Context, userID, bookID, settings string) error
}
