// Package organize is the application service for the library's organizational layer:
// collections (curated, shared shelves) today, with reading lists, tags, and smart lists
// to follow. It owns id/timestamp assignment, validation, and fractional reorder math,
// delegating persistence to the domain.Repository.
package organize

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/pkg/ulid"
)

// positionGap mirrors the store's spacing; new items land a gap past the last one and
// reorders bisect, so positions rarely need renumbering.
const positionGap = 1024.0

// Service handles organization use cases.
type Service struct {
	repo domain.Repository
}

// New constructs the organize service over a repository.
func New(repo domain.Repository) *Service { return &Service{repo: repo} }

// CollectionInput is the validated request to create a collection.
type CollectionInput struct {
	Name        string
	Description string
}

// CollectionPatch carries optional edits; nil fields are left unchanged.
type CollectionPatch struct {
	Name        *string
	Description *string
	CoverBookID *string
}

// CreateCollection validates and persists a new collection owned by the acting user.
func (s *Service) CreateCollection(ctx context.Context, ownerID string, in CollectionInput) (domain.Collection, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return domain.Collection{}, fmt.Errorf("%w: name is required", domain.ErrValidation)
	}
	now := time.Now().UnixMilli()
	c := domain.Collection{
		ID:          ulid.New(),
		Name:        name,
		Description: strings.TrimSpace(in.Description),
		OwnerID:     ownerID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	return s.repo.Collections().Create(ctx, c)
}

// ListCollections returns all collections (with item counts).
func (s *Service) ListCollections(ctx context.Context) ([]domain.Collection, error) {
	return s.repo.Collections().List(ctx)
}

// GetCollection returns one collection (ErrNotFound if absent).
func (s *Service) GetCollection(ctx context.Context, id string) (domain.Collection, error) {
	return s.repo.Collections().Get(ctx, id)
}

// CollectionItems returns the collection's book ids in display order.
func (s *Service) CollectionItems(ctx context.Context, id string) ([]string, error) {
	items, err := s.repo.Collections().Items(ctx, id)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(items))
	for i, it := range items {
		ids[i] = it.BookID
	}
	return ids, nil
}

// UpdateCollection applies a partial edit (read-modify-write) and returns the result.
func (s *Service) UpdateCollection(ctx context.Context, id string, patch CollectionPatch) (domain.Collection, error) {
	c, err := s.repo.Collections().Get(ctx, id)
	if err != nil {
		return domain.Collection{}, err
	}
	if patch.Name != nil {
		name := strings.TrimSpace(*patch.Name)
		if name == "" {
			return domain.Collection{}, fmt.Errorf("%w: name cannot be empty", domain.ErrValidation)
		}
		c.Name = name
	}
	if patch.Description != nil {
		c.Description = strings.TrimSpace(*patch.Description)
	}
	if patch.CoverBookID != nil {
		c.CoverBookID = strings.TrimSpace(*patch.CoverBookID)
	}
	c.UpdatedAt = time.Now().UnixMilli()
	if err := s.repo.Collections().Update(ctx, c); err != nil {
		return domain.Collection{}, err
	}
	return c, nil
}

// DeleteCollection removes a collection and its items.
func (s *Service) DeleteCollection(ctx context.Context, id string) error {
	return s.repo.Collections().Delete(ctx, id)
}

// AddItems appends books to a collection (existing members keep their place).
func (s *Service) AddItems(ctx context.Context, id string, bookIDs []string) error {
	if _, err := s.repo.Collections().Get(ctx, id); err != nil {
		return err
	}
	clean := dedupeNonEmpty(bookIDs)
	if len(clean) == 0 {
		return fmt.Errorf("%w: bookIds is required", domain.ErrValidation)
	}
	return s.repo.Collections().AddItems(ctx, id, clean)
}

// RemoveItem drops one book from a collection.
func (s *Service) RemoveItem(ctx context.Context, id, bookID string) error {
	return s.repo.Collections().RemoveItem(ctx, id, bookID)
}

// Reorder moves bookID to sit immediately before beforeID; an empty beforeID moves it to
// the end. Positions bisect between neighbours, so only the moved row is rewritten.
func (s *Service) Reorder(ctx context.Context, id, bookID, beforeID string) error {
	items, err := s.repo.Collections().Items(ctx, id)
	if err != nil {
		return err
	}
	cur := make([]positioned, len(items))
	for i, it := range items {
		cur[i] = positioned{bookID: it.BookID, position: it.Position}
	}
	newPos, err := reorderPosition(cur, bookID, beforeID)
	if err != nil {
		return err
	}
	return s.repo.Collections().SetPosition(ctx, id, bookID, newPos)
}

// positioned is a book id with its fractional sort position — the input to the shared
// reorder math used by collections and reading lists.
type positioned struct {
	bookID   string
	position float64
}

// reorderPosition computes the new fractional position for moving bookID before beforeID
// (empty = move to the end), bisecting between the neighbours in the current order.
func reorderPosition(items []positioned, bookID, beforeID string) (float64, error) {
	rest := make([]positioned, 0, len(items))
	found := false
	for _, it := range items {
		if it.bookID == bookID {
			found = true
			continue
		}
		rest = append(rest, it)
	}
	if !found {
		return 0, fmt.Errorf("%w: book is not in this list", domain.ErrValidation)
	}

	if beforeID == "" {
		if len(rest) == 0 {
			return positionGap, nil
		}
		return rest[len(rest)-1].position + positionGap, nil
	}
	for i, it := range rest {
		if it.bookID == beforeID {
			if i == 0 {
				return rest[0].position - positionGap, nil
			}
			return (rest[i-1].position + rest[i].position) / 2, nil
		}
	}
	return 0, fmt.Errorf("%w: beforeId is not in this list", domain.ErrValidation)
}

// ── Reading lists (per-user) ───────────────────────────────────────────────────────

// CreateReadingList validates and persists a new list owned by userID.
func (s *Service) CreateReadingList(ctx context.Context, userID, name string) (domain.ReadingList, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.ReadingList{}, fmt.Errorf("%w: name is required", domain.ErrValidation)
	}
	now := time.Now().UnixMilli()
	l := domain.ReadingList{
		ID:        ulid.New(),
		UserID:    userID,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return s.repo.ReadingLists().Create(ctx, l)
}

// ListReadingLists returns the user's lists (with item counts).
func (s *Service) ListReadingLists(ctx context.Context, userID string) ([]domain.ReadingList, error) {
	return s.repo.ReadingLists().List(ctx, userID)
}

// GetReadingList returns one of the user's lists (ErrNotFound if absent or not owned).
func (s *Service) GetReadingList(ctx context.Context, userID, id string) (domain.ReadingList, error) {
	return s.repo.ReadingLists().Get(ctx, userID, id)
}

// ReadingListItems returns the list's book ids in display order (owner-checked).
func (s *Service) ReadingListItems(ctx context.Context, userID, id string) ([]string, error) {
	if _, err := s.repo.ReadingLists().Get(ctx, userID, id); err != nil {
		return nil, err
	}
	items, err := s.repo.ReadingLists().Items(ctx, id)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(items))
	for i, it := range items {
		ids[i] = it.BookID
	}
	return ids, nil
}

// RenameReadingList updates a list's name (the only editable field).
func (s *Service) RenameReadingList(ctx context.Context, userID, id, name string) (domain.ReadingList, error) {
	l, err := s.repo.ReadingLists().Get(ctx, userID, id)
	if err != nil {
		return domain.ReadingList{}, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.ReadingList{}, fmt.Errorf("%w: name cannot be empty", domain.ErrValidation)
	}
	l.Name = name
	l.UpdatedAt = time.Now().UnixMilli()
	if err := s.repo.ReadingLists().Update(ctx, l); err != nil {
		return domain.ReadingList{}, err
	}
	return l, nil
}

// DeleteReadingList removes a user's list and its items.
func (s *Service) DeleteReadingList(ctx context.Context, userID, id string) error {
	return s.repo.ReadingLists().Delete(ctx, userID, id)
}

// AddReadingListItems appends books to a user's list (existing members keep their place).
func (s *Service) AddReadingListItems(ctx context.Context, userID, id string, bookIDs []string) error {
	if _, err := s.repo.ReadingLists().Get(ctx, userID, id); err != nil {
		return err
	}
	clean := dedupeNonEmpty(bookIDs)
	if len(clean) == 0 {
		return fmt.Errorf("%w: bookIds is required", domain.ErrValidation)
	}
	return s.repo.ReadingLists().AddItems(ctx, id, clean)
}

// RemoveReadingListItem drops one book from a user's list.
func (s *Service) RemoveReadingListItem(ctx context.Context, userID, id, bookID string) error {
	if _, err := s.repo.ReadingLists().Get(ctx, userID, id); err != nil {
		return err
	}
	return s.repo.ReadingLists().RemoveItem(ctx, id, bookID)
}

// ReorderReadingList moves bookID before beforeID within a user's list (empty = to end).
func (s *Service) ReorderReadingList(ctx context.Context, userID, id, bookID, beforeID string) error {
	if _, err := s.repo.ReadingLists().Get(ctx, userID, id); err != nil {
		return err
	}
	items, err := s.repo.ReadingLists().Items(ctx, id)
	if err != nil {
		return err
	}
	cur := make([]positioned, len(items))
	for i, it := range items {
		cur[i] = positioned{bookID: it.BookID, position: it.Position}
	}
	newPos, err := reorderPosition(cur, bookID, beforeID)
	if err != nil {
		return err
	}
	return s.repo.ReadingLists().SetPosition(ctx, id, bookID, newPos)
}

// ── Tags ─────────────────────────────────────────────────────────────────────────────

// CreateTag adds a tag with a unique (case-insensitive) name and optional color.
func (s *Service) CreateTag(ctx context.Context, name, color string) (domain.Tag, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.Tag{}, fmt.Errorf("%w: name is required", domain.ErrValidation)
	}
	if _, err := s.repo.Tags().GetByName(ctx, name); err == nil {
		return domain.Tag{}, fmt.Errorf("%w: a tag named %q already exists", domain.ErrValidation, name)
	} else if !errors.Is(err, domain.ErrNotFound) {
		return domain.Tag{}, err
	}
	t := domain.Tag{ID: ulid.New(), Name: name, Color: strings.TrimSpace(color)}
	return s.repo.Tags().Create(ctx, t)
}

// ListTags returns all tags (with book counts), name-sorted.
func (s *Service) ListTags(ctx context.Context) ([]domain.Tag, error) {
	return s.repo.Tags().List(ctx)
}

// TagPatch carries optional tag edits; nil fields are left unchanged.
type TagPatch struct {
	Name  *string
	Color *string
}

// UpdateTag renames/recolors a tag, keeping names unique.
func (s *Service) UpdateTag(ctx context.Context, id string, patch TagPatch) (domain.Tag, error) {
	t, err := s.repo.Tags().Get(ctx, id)
	if err != nil {
		return domain.Tag{}, err
	}
	if patch.Name != nil {
		name := strings.TrimSpace(*patch.Name)
		if name == "" {
			return domain.Tag{}, fmt.Errorf("%w: name cannot be empty", domain.ErrValidation)
		}
		if existing, err := s.repo.Tags().GetByName(ctx, name); err == nil && existing.ID != id {
			return domain.Tag{}, fmt.Errorf("%w: a tag named %q already exists", domain.ErrValidation, name)
		} else if err != nil && !errors.Is(err, domain.ErrNotFound) {
			return domain.Tag{}, err
		}
		t.Name = name
	}
	if patch.Color != nil {
		t.Color = strings.TrimSpace(*patch.Color)
	}
	if err := s.repo.Tags().Update(ctx, t); err != nil {
		return domain.Tag{}, err
	}
	return t, nil
}

// DeleteTag removes a tag and all its book assignments.
func (s *Service) DeleteTag(ctx context.Context, id string) error {
	return s.repo.Tags().Delete(ctx, id)
}

// TaggedBookIDs returns the ids of books carrying a tag (newest-added first).
func (s *Service) TaggedBookIDs(ctx context.Context, tagID string) ([]string, error) {
	if _, err := s.repo.Tags().Get(ctx, tagID); err != nil {
		return nil, err
	}
	return s.repo.Tags().TaggedBookIDs(ctx, tagID)
}

// AssignTags tags a book with the given tag ids (idempotent).
func (s *Service) AssignTags(ctx context.Context, bookID string, tagIDs []string) error {
	clean := dedupeNonEmpty(tagIDs)
	if len(clean) == 0 {
		return fmt.Errorf("%w: tagIds is required", domain.ErrValidation)
	}
	// Validate each tag exists so an unknown id is a clear 400, not an FK error.
	for _, id := range clean {
		if _, err := s.repo.Tags().Get(ctx, id); err != nil {
			return err
		}
	}
	return s.repo.Tags().AssignToBook(ctx, bookID, clean)
}

// UnassignTag removes one tag from a book.
func (s *Service) UnassignTag(ctx context.Context, bookID, tagID string) error {
	return s.repo.Tags().UnassignFromBook(ctx, bookID, tagID)
}

func dedupeNonEmpty(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, dup := seen[s]; dup {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
