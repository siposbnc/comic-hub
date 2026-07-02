package domain

import "context"

// NameCount is a ranked aggregate row (top genres / publishers).
type NameCount struct {
	Name  string
	Count int
}

// FinishedBook is a recently completed issue for the stats dashboard's cover rail.
type FinishedBook struct {
	BookID     string
	Title      string
	Number     string
	SeriesID   string
	SeriesName string
	FinishedAt int64
}

// StatsRepository serves the per-user reading-stats aggregates (Phase 3, Milestone G).
// It returns raw counts and timestamp sets; the stats service does the calendar math
// (month buckets, streaks) in Go so the SQL stays dialect-portable.
type StatsRepository interface {
	// ReadCounts returns the user's all-time finished-book count and pages read
	// (full page count for finished books, current page for in-progress ones).
	ReadCounts(ctx context.Context, userID string) (books, pages int, err error)
	// FinishedTimes returns finished_at (unix ms) of the user's finished books since
	// the given time, for month bucketing and this-year counts.
	FinishedTimes(ctx context.Context, userID string, since int64) ([]int64, error)
	// ActivityTimes returns every day-relevant timestamp on the user's progress rows
	// (started/finished/updated). Streaks are approximated from these — the catalog
	// keeps only the latest update per book, not a full reading diary.
	ActivityTimes(ctx context.Context, userID string) ([]int64, error)
	// TopGenres ranks genres across the user's finished books.
	TopGenres(ctx context.Context, userID string, limit int) ([]NameCount, error)
	// TopPublishers ranks series publishers across the user's finished books.
	TopPublishers(ctx context.Context, userID string, limit int) ([]NameCount, error)
	// RecentlyFinished returns the user's latest finished books, newest first.
	RecentlyFinished(ctx context.Context, userID string, limit int) ([]FinishedBook, error)
}
