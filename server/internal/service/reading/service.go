// Package reading owns per-user reading progress: upserting current page/status,
// marking read/unread, and feeding "Continue Reading". Writes are debounced/batched by
// the client and broadcast over the WS progress topic (docs/03-api.md §6).
package reading

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/pkg/ulid"
)

// ProgressNotifier is called after a successful progress write so the transport layer
// can broadcast it (e.g. over WebSocket). Optional.
type ProgressNotifier func(userID string, p domain.Progress)

// BookmarkNotifier is called after a bookmark add/update/delete so the transport layer
// can broadcast that a book's bookmarks changed. Optional.
type BookmarkNotifier func(userID, bookID string)

// Service manages reading progress.
type Service struct {
	repo     domain.Repository
	notify   ProgressNotifier
	bmNotify BookmarkNotifier
}

// New constructs the reading service. notify may be nil.
func New(repo domain.Repository, notify ProgressNotifier) *Service {
	return &Service{repo: repo, notify: notify}
}

// OnBookmarkChange registers a notifier fired after every bookmark add/update/delete.
func (s *Service) OnBookmarkChange(fn BookmarkNotifier) { s.bmNotify = fn }

// maxNoteLen caps a bookmark note (the UI keeps notes short; this is a safety bound).
const maxNoteLen = 280

// UpsertInput is a progress update from a reader/client.
type UpsertInput struct {
	Page   int
	Status string // optional; derived from page when empty
	Device string
}

// Get returns the user's progress for a book (ErrNotFound if none yet).
func (s *Service) Get(ctx context.Context, userID, bookID string) (domain.Progress, error) {
	p, err := s.repo.Progress().Get(ctx, userID, bookID)
	if err == nil {
		return p, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return domain.Progress{}, err
	}
	// No progress row yet: return a default "unread" progress (page 0) so a client opening
	// a fresh book gets a 200 with sensible defaults instead of a 404 it must special-case.
	// An unknown book is still a real 404.
	book, berr := s.repo.Books().Get(ctx, bookID)
	if berr != nil {
		return domain.Progress{}, berr
	}
	return domain.Progress{
		UserID:    userID,
		BookID:    bookID,
		Page:      0,
		PageCount: book.PageCount,
		Status:    domain.StatusUnread,
	}, nil
}

// Upsert records progress for a book, snapshotting the page count and deriving status
// when the caller doesn't supply one.
func (s *Service) Upsert(ctx context.Context, userID, bookID string, in UpsertInput) (domain.Progress, error) {
	book, err := s.repo.Books().Get(ctx, bookID)
	if err != nil {
		return domain.Progress{}, err
	}

	page := clamp(in.Page, 0, lastIndex(book.PageCount))
	status := domain.ReadStatus(in.Status)
	if status == "" {
		status = deriveStatus(page, book.PageCount)
	}

	existing, _ := s.repo.Progress().Get(ctx, userID, bookID)
	now := time.Now().UnixMilli()

	p := domain.Progress{
		UserID:    userID,
		BookID:    bookID,
		Page:      page,
		PageCount: book.PageCount,
		Status:    status,
		StartedAt: existing.StartedAt,
		UpdatedAt: now,
		Device:    in.Device,
	}
	if p.StartedAt == 0 && status != domain.StatusUnread {
		p.StartedAt = now
	}
	if status == domain.StatusRead {
		p.FinishedAt = now
		if existing.FinishedAt != 0 {
			p.FinishedAt = existing.FinishedAt
		}
	}

	return s.save(ctx, p)
}

// Mark sets a book read or unread (a convenience over Upsert).
func (s *Service) Mark(ctx context.Context, userID, bookID, status string) (domain.Progress, error) {
	book, err := s.repo.Books().Get(ctx, bookID)
	if err != nil {
		return domain.Progress{}, err
	}
	now := time.Now().UnixMilli()
	p := domain.Progress{
		UserID:    userID,
		BookID:    bookID,
		PageCount: book.PageCount,
		UpdatedAt: now,
	}
	switch status {
	case string(domain.StatusRead):
		p.Status = domain.StatusRead
		p.Page = lastIndex(book.PageCount)
		p.StartedAt = now
		p.FinishedAt = now
	case string(domain.StatusUnread):
		p.Status = domain.StatusUnread
		p.Page = 0
	default:
		return domain.Progress{}, errors.New("status must be \"read\" or \"unread\"")
	}
	return s.save(ctx, p)
}

// GetReaderPrefs returns the user's stored reader overrides for a book as a raw JSON
// object, or "{}" when none are saved. The shape is owned by the reader client.
func (s *Service) GetReaderPrefs(ctx context.Context, userID, bookID string) (json.RawMessage, error) {
	raw, err := s.repo.ReaderPrefs().Get(ctx, userID, bookID)
	if errors.Is(err, domain.ErrNotFound) {
		return json.RawMessage("{}"), nil
	}
	if err != nil {
		return nil, err
	}
	return json.RawMessage(raw), nil
}

// SetReaderPrefs stores the user's reader overrides for a book. The book must exist and
// settings must be a JSON object.
func (s *Service) SetReaderPrefs(ctx context.Context, userID, bookID string, settings json.RawMessage) error {
	if _, err := s.repo.Books().Get(ctx, bookID); err != nil {
		return err
	}
	if len(settings) == 0 || !json.Valid(settings) {
		return fmt.Errorf("%w: settings must be a JSON object", domain.ErrValidation)
	}
	return s.repo.ReaderPrefs().Put(ctx, userID, bookID, string(settings))
}

// ListBookmarks returns the user's bookmarks for a book, ordered by page ascending.
func (s *Service) ListBookmarks(ctx context.Context, userID, bookID string) ([]domain.Bookmark, error) {
	return s.repo.Bookmarks().List(ctx, userID, bookID)
}

// AddBookmark bookmarks a page (clamped to the book's range). If the page is already
// bookmarked, its note is updated instead — so the reader can safely "add" idempotently.
func (s *Service) AddBookmark(ctx context.Context, userID, bookID string, page int, note string) (domain.Bookmark, error) {
	book, err := s.repo.Books().Get(ctx, bookID)
	if err != nil {
		return domain.Bookmark{}, err
	}
	page = clamp(page, 0, lastIndex(book.PageCount))
	note = trimNote(note)
	now := time.Now().UnixMilli()

	existing, err := s.repo.Bookmarks().GetByPage(ctx, userID, bookID, page)
	switch {
	case err == nil:
		updated, uerr := s.repo.Bookmarks().UpdateNote(ctx, userID, existing.ID, note, now)
		if uerr != nil {
			return domain.Bookmark{}, uerr
		}
		s.notifyBookmark(userID, bookID)
		return updated, nil
	case errors.Is(err, domain.ErrNotFound):
		created, cerr := s.repo.Bookmarks().Create(ctx, domain.Bookmark{
			ID: ulid.New(), UserID: userID, BookID: bookID, Page: page, Note: note,
			CreatedAt: now, UpdatedAt: now,
		})
		if cerr != nil {
			return domain.Bookmark{}, cerr
		}
		s.notifyBookmark(userID, bookID)
		return created, nil
	default:
		return domain.Bookmark{}, err
	}
}

// UpdateBookmarkNote replaces a bookmark's note.
func (s *Service) UpdateBookmarkNote(ctx context.Context, userID, id, note string) (domain.Bookmark, error) {
	updated, err := s.repo.Bookmarks().UpdateNote(ctx, userID, id, trimNote(note), time.Now().UnixMilli())
	if err != nil {
		return domain.Bookmark{}, err
	}
	s.notifyBookmark(userID, updated.BookID)
	return updated, nil
}

// DeleteBookmark removes a bookmark owned by the user.
func (s *Service) DeleteBookmark(ctx context.Context, userID, id string) error {
	// Resolve first so we can name the affected book in the change notification.
	bm, err := s.repo.Bookmarks().Get(ctx, userID, id)
	if err != nil {
		return err
	}
	if err := s.repo.Bookmarks().Delete(ctx, userID, id); err != nil {
		return err
	}
	s.notifyBookmark(userID, bm.BookID)
	return nil
}

func (s *Service) notifyBookmark(userID, bookID string) {
	if s.bmNotify != nil {
		s.bmNotify(userID, bookID)
	}
}

func trimNote(note string) string {
	note = strings.TrimSpace(note)
	if len(note) > maxNoteLen {
		note = note[:maxNoteLen]
	}
	return note
}

func (s *Service) save(ctx context.Context, p domain.Progress) (domain.Progress, error) {
	saved, err := s.repo.Progress().Upsert(ctx, p)
	if err != nil {
		return domain.Progress{}, err
	}
	if s.notify != nil {
		s.notify(saved.UserID, saved)
	}
	return saved, nil
}

func deriveStatus(page, pageCount int) domain.ReadStatus {
	if pageCount > 0 && page >= lastIndex(pageCount) {
		return domain.StatusRead
	}
	return domain.StatusInProgress
}

func lastIndex(pageCount int) int {
	if pageCount <= 0 {
		return 0
	}
	return pageCount - 1
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
