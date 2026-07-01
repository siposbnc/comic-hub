// Package presence tracks who is reading what right now ("now reading" — Phase 3,
// Milestone E). Presence is derived from progress writes, so it needs no extra client
// wiring: a user is present while their reader keeps reporting page turns, and their
// entry expires a TTL after the last one (or clears immediately when the book is
// finished or marked read/unread). State is in-memory only — it is ambient awareness,
// not a record, and an empty map after restart is correct.
package presence

import (
	"context"
	"sync"
	"time"

	"github.com/siposbnc/comic-hub/server/internal/domain"
)

// DefaultTTL is how long after the last page turn a reader still counts as reading.
const DefaultTTL = 5 * time.Minute

// Entry is one user's current reading activity, enriched for direct display (the
// client renders rows from this without extra lookups; covers come from bookId).
type Entry struct {
	UserID      string `json:"userId"`
	DisplayName string `json:"displayName"`
	BookID      string `json:"bookId"`
	BookTitle   string `json:"bookTitle"`
	SeriesID    string `json:"seriesId,omitempty"`
	SeriesTitle string `json:"seriesTitle,omitempty"`
	Page        int    `json:"page"`
	PageCount   int    `json:"pageCount"`
	// AgeRating is the book's rating, carried so the transport can hide the entry from
	// viewers whose content ceiling excludes it. Not serialized to clients.
	AgeRating string `json:"-"`
	UpdatedAt int64  `json:"updatedAt"` // unix ms of the last page turn
}

// Notifier observes presence changes. active=false means the entry was cleared
// (finished, marked, or expired); the Entry then names the user and their last book.
type Notifier func(e Entry, active bool)

// Tracker keeps the current presence set. Safe for concurrent use.
type Tracker struct {
	mu      sync.Mutex
	entries map[string]Entry // by user id — one book per user, the latest wins
	ttl     time.Duration
	notify  Notifier
	now     func() time.Time
}

// New builds a tracker; ttl <= 0 uses DefaultTTL.
func New(ttl time.Duration) *Tracker {
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	return &Tracker{entries: make(map[string]Entry), ttl: ttl, now: time.Now}
}

// OnChange registers the change observer (the WS broadcast). Call before use.
func (t *Tracker) OnChange(fn Notifier) { t.notify = fn }

// Touch records that a user is reading a book right now.
func (t *Tracker) Touch(e Entry) {
	if e.UserID == "" || e.BookID == "" {
		return
	}
	if e.UpdatedAt == 0 {
		e.UpdatedAt = t.now().UnixMilli()
	}
	t.mu.Lock()
	t.entries[e.UserID] = e
	t.mu.Unlock()
	if t.notify != nil {
		t.notify(e, true)
	}
}

// Clear drops a user's presence (finished the book, marked it, signed out). No-op —
// and no notification — when the user wasn't present.
func (t *Tracker) Clear(userID string) {
	t.mu.Lock()
	e, ok := t.entries[userID]
	if ok {
		delete(t.entries, userID)
	}
	t.mu.Unlock()
	if ok && t.notify != nil {
		t.notify(e, false)
	}
}

// Snapshot returns the current presence set (unexpired entries, unordered).
func (t *Tracker) Snapshot() []Entry {
	cutoff := t.now().Add(-t.ttl).UnixMilli()
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]Entry, 0, len(t.entries))
	for _, e := range t.entries {
		if e.UpdatedAt >= cutoff {
			out = append(out, e)
		}
	}
	return out
}

// ObserveProgress returns a progress notifier (the reading service's callback shape)
// that maintains presence: an in-progress write touches the user's entry — enriched
// from the repo for direct display — and anything else (finished, marked read/unread)
// clears it. This is the single seam between progress and presence; main and tests
// wire it identically.
func (t *Tracker) ObserveProgress(repo domain.Repository) func(userID string, p domain.Progress) {
	return func(userID string, p domain.Progress) {
		if p.Status != domain.StatusInProgress {
			t.Clear(userID)
			return
		}
		ctx := context.Background()
		e := Entry{
			UserID: userID, BookID: p.BookID,
			Page: p.Page, PageCount: p.PageCount, UpdatedAt: p.UpdatedAt,
		}
		if u, err := repo.Users().Get(ctx, userID); err == nil {
			e.DisplayName = u.DisplayName
		}
		if b, err := repo.Books().Get(ctx, p.BookID); err == nil {
			e.BookTitle = b.Title
			e.AgeRating = b.AgeRating
			e.SeriesID = b.SeriesID
			if s, err := repo.Series().Get(ctx, b.SeriesID); err == nil {
				e.SeriesTitle = s.Name
			}
		}
		t.Touch(e)
	}
}

// Run sweeps expired entries (notifying each) until ctx is cancelled.
func (t *Tracker) Run(ctx context.Context) {
	ticker := time.NewTicker(t.ttl / 4)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.sweep()
		}
	}
}

func (t *Tracker) sweep() {
	cutoff := t.now().Add(-t.ttl).UnixMilli()
	var expired []Entry
	t.mu.Lock()
	for id, e := range t.entries {
		if e.UpdatedAt < cutoff {
			delete(t.entries, id)
			expired = append(expired, e)
		}
	}
	t.mu.Unlock()
	if t.notify != nil {
		for _, e := range expired {
			t.notify(e, false)
		}
	}
}
