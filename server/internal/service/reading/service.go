// Package reading owns per-user reading progress: upserting current page/status,
// marking read/unread, and feeding "Continue Reading". Writes are debounced/batched by
// the client and broadcast over the WS progress topic (docs/03-api.md §6).
package reading

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/siposbnc/comic-hub/server/internal/domain"
)

// ProgressNotifier is called after a successful progress write so the transport layer
// can broadcast it (e.g. over WebSocket). Optional.
type ProgressNotifier func(userID string, p domain.Progress)

// Service manages reading progress.
type Service struct {
	repo   domain.Repository
	notify ProgressNotifier
}

// New constructs the reading service. notify may be nil.
func New(repo domain.Repository, notify ProgressNotifier) *Service {
	return &Service{repo: repo, notify: notify}
}

// UpsertInput is a progress update from a reader/client.
type UpsertInput struct {
	Page   int
	Status string // optional; derived from page when empty
	Device string
}

// Get returns the user's progress for a book (ErrNotFound if none yet).
func (s *Service) Get(ctx context.Context, userID, bookID string) (domain.Progress, error) {
	return s.repo.Progress().Get(ctx, userID, bookID)
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
