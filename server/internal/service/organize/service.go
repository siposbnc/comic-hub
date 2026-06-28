// Package organize is the application service for the library's organizational layer:
// collections (curated, shared shelves) today, with reading lists, tags, and smart lists
// to follow. It owns id/timestamp assignment, validation, and fractional reorder math,
// delegating persistence to the domain.Repository.
package organize

import (
	"context"
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

	// The reference order excludes the item being moved.
	rest := make([]domain.CollectionItem, 0, len(items))
	found := false
	for _, it := range items {
		if it.BookID == bookID {
			found = true
			continue
		}
		rest = append(rest, it)
	}
	if !found {
		return fmt.Errorf("%w: book is not in this collection", domain.ErrValidation)
	}

	var newPos float64
	if beforeID == "" {
		if len(rest) == 0 {
			newPos = positionGap
		} else {
			newPos = rest[len(rest)-1].Position + positionGap
		}
	} else {
		idx := -1
		for i, it := range rest {
			if it.BookID == beforeID {
				idx = i
				break
			}
		}
		if idx < 0 {
			return fmt.Errorf("%w: beforeId is not in this collection", domain.ErrValidation)
		}
		if idx == 0 {
			newPos = rest[0].Position - positionGap
		} else {
			newPos = (rest[idx-1].Position + rest[idx].Position) / 2
		}
	}
	return s.repo.Collections().SetPosition(ctx, id, bookID, newPos)
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
