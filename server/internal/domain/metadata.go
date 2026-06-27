package domain

import "context"

// BookMeta is the writable metadata envelope for a book: the scalar fields, the
// provider-id map (provider name -> external id, so a future re-match reuses the link),
// and the set of user-locked field names that an online match must never overwrite.
type BookMeta struct {
	Title        string
	Number       string
	Volume       int
	ReleaseDate  int64
	AgeRating    string
	Language     string
	Summary      string
	State        MetadataState
	ProviderIDs  map[string]string
	LockedFields []string
}

// MetadataRepository persists the per-book metadata envelope and the normalized
// credits/genres/characters that online matching produces. Scalar catalog fields are
// also writable via BookRepository.Upsert; this boundary exists for the match-apply path,
// which writes credits/locks/provider-ids the scanner does not touch.
type MetadataRepository interface {
	// WriteBookMeta replaces a book's scalar metadata, provider ids, and locked fields in
	// one statement. The caller supplies the final, lock-resolved values.
	WriteBookMeta(ctx context.Context, bookID string, m BookMeta) error
	// LockedBookFields returns the field names the user has pinned on a book.
	LockedBookFields(ctx context.Context, bookID string) ([]string, error)
	// BookProviderIDs returns the book's provider name -> external id map.
	BookProviderIDs(ctx context.Context, bookID string) (map[string]string, error)

	// ReplaceBookPeople swaps a book's credits (role -> names).
	ReplaceBookPeople(ctx context.Context, bookID string, credits map[string][]string) error
	// ReplaceBookGenres swaps a book's genres.
	ReplaceBookGenres(ctx context.Context, bookID string, names []string) error
	// ReplaceBookCharacters swaps a book's characters.
	ReplaceBookCharacters(ctx context.Context, bookID string, names []string) error

	BookCredits(ctx context.Context, bookID string) (map[string][]string, error)
	BookGenres(ctx context.Context, bookID string) ([]string, error)
	BookCharacters(ctx context.Context, bookID string) ([]string, error)
}
