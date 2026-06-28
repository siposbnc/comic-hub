// Package library is the application service for managing libraries: creating them
// from a name + root folders, listing, fetching, and removing them. It owns id and
// timestamp assignment and input validation, delegating persistence to the
// domain.Repository. Scanning is a separate concern (see the scanner milestone).
package library

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/pkg/ulid"
)

// Service handles library use cases.
type Service struct {
	repo     domain.Repository
	onCreate func(domain.Library)
	onDelete func(id string)
}

// New constructs the library service over a repository.
func New(repo domain.Repository) *Service {
	return &Service{repo: repo}
}

// OnCreate registers a hook run after a library is created (e.g. start watching its
// folders). OnDelete registers one run after a library is removed.
func (s *Service) OnCreate(fn func(domain.Library)) { s.onCreate = fn }
func (s *Service) OnDelete(fn func(id string))      { s.onDelete = fn }

// CreateInput is the validated request to create a library.
type CreateInput struct {
	Name  string
	Kind  string
	Roots []string
}

// Create validates the input, normalizes root paths, and persists a new library.
func (s *Service) Create(ctx context.Context, in CreateInput) (domain.Library, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return domain.Library{}, fmt.Errorf("%w: name is required", domain.ErrValidation)
	}

	kind := strings.TrimSpace(in.Kind)
	if kind == "" {
		kind = "comic"
	}
	if kind != "comic" && kind != "manga" {
		return domain.Library{}, fmt.Errorf("%w: kind must be \"comic\" or \"manga\"", domain.ErrValidation)
	}

	roots, err := normalizeRoots(in.Roots)
	if err != nil {
		return domain.Library{}, err
	}

	now := time.Now().UnixMilli()
	lib := domain.Library{
		ID:        ulid.New(),
		Name:      name,
		Kind:      kind,
		Roots:     roots,
		CreatedAt: now,
		UpdatedAt: now,
	}
	created, err := s.repo.Libraries().Create(ctx, lib)
	if err == nil && s.onCreate != nil {
		s.onCreate(created)
	}
	return created, err
}

// List returns all libraries.
func (s *Service) List(ctx context.Context) ([]domain.Library, error) {
	return s.repo.Libraries().List(ctx)
}

// Get returns one library by id (domain.ErrNotFound if absent).
func (s *Service) Get(ctx context.Context, id string) (domain.Library, error) {
	return s.repo.Libraries().Get(ctx, id)
}

// Delete removes a library from the catalog. The files on disk are untouched.
func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.repo.Libraries().Delete(ctx, id); err != nil {
		return err
	}
	if s.onDelete != nil {
		s.onDelete(id)
	}
	return nil
}

// normalizeRoots cleans and de-duplicates root paths, requiring at least one. Paths
// are made absolute so the catalog stores canonical locations regardless of the
// caller's working directory.
func normalizeRoots(in []string) ([]string, error) {
	seen := make(map[string]struct{}, len(in))
	var roots []string
	for _, raw := range in {
		p := strings.TrimSpace(raw)
		if p == "" {
			continue
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid root path %q", domain.ErrValidation, raw)
		}
		abs = filepath.Clean(abs)
		if _, dup := seen[abs]; dup {
			continue
		}
		seen[abs] = struct{}{}
		roots = append(roots, abs)
	}
	if len(roots) == 0 {
		return nil, fmt.Errorf("%w: at least one root folder is required", domain.ErrValidation)
	}
	return roots, nil
}
