package domain

import "context"

// Track is a standalone (manual) tracker row: a series the user follows that is in no
// library. Library series are not stored as tracks — they are projected live from the
// catalog. A track only groups the user's overlay issues (issues with no backing file).
type Track struct {
	ID        string
	UserID    string
	Name      string
	CreatedAt int64
	UpdatedAt int64
}

// TrackIssue is an overlay issue on the tracker: an issue with no backing book. It hangs
// off either a standalone Track (TrackID set) or a library series (SeriesID set) — exactly
// one is non-empty. Its Read flag is independent of any file, so an issue read elsewhere
// can be marked read even though ComicHub holds no `.cbz` for it.
type TrackIssue struct {
	ID        string
	UserID    string
	TrackID   string // set for a standalone-track issue (empty otherwise)
	SeriesID  string // set for an overlay on a library series (empty otherwise)
	Number    string
	Sort      float64
	Read      bool
	ReadAt    int64
	CreatedAt int64
}

// TrackRepository persists the tracker's user-owned overlay: standalone tracks and the
// manual issues attached to tracks or library series. Every method is scoped to the owning
// user; reads/writes for another user's row return ErrNotFound.
type TrackRepository interface {
	CreateTrack(ctx context.Context, t Track) (Track, error)
	GetTrack(ctx context.Context, userID, id string) (Track, error)
	ListTracks(ctx context.Context, userID string) ([]Track, error)
	RenameTrack(ctx context.Context, t Track) error
	DeleteTrack(ctx context.Context, userID, id string) error

	// OverlayIssues returns every overlay issue the user has added — both series-attached
	// and track-attached — so the tracker view assembles in a single read.
	OverlayIssues(ctx context.Context, userID string) ([]TrackIssue, error)
	// AddIssues inserts overlay issues, skipping any whose number already exists in the
	// same track/series (the unique index), so re-adding a range is idempotent.
	AddIssues(ctx context.Context, issues []TrackIssue) error
	GetIssue(ctx context.Context, userID, id string) (TrackIssue, error)
	RemoveIssue(ctx context.Context, userID, id string) error
	// SetIssueRead flips an overlay issue's read flag (at = read timestamp, 0 when clearing).
	SetIssueRead(ctx context.Context, userID, id string, read bool, at int64) error
}
