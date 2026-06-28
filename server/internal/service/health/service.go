// Package health is the read side of library maintenance: it scans a library's catalog
// rows for problems — books flagged corrupt, files that have vanished from disk (orphans),
// issues with no metadata yet (unmatched), and identical files stored more than once
// (duplicates). See docs/03-api.md §7 (libraries health).
package health

import (
	"context"
	"os"

	"github.com/siposbnc/comic-hub/server/internal/domain"
)

// maxItems caps each problem list so a troubled library can't return a huge payload; the
// counts still reflect the true totals.
const maxItems = 200

// Item identifies a problem book compactly enough to navigate to it.
type Item struct {
	ID       string `json:"id"`
	SeriesID string `json:"seriesId"`
	Number   string `json:"number,omitempty"`
	Title    string `json:"title,omitempty"`
	Path     string `json:"path"`
}

// DuplicateGroup is a set of books whose files share a content hash.
type DuplicateGroup struct {
	ContentHash string `json:"contentHash"`
	Books       []Item `json:"books"`
}

// Counts are the totals across the whole library (independent of the capped lists).
type Counts struct {
	Books           int `json:"books"`
	Corrupt         int `json:"corrupt"`
	Orphans         int `json:"orphans"`
	Unmatched       int `json:"unmatched"`
	DuplicateGroups int `json:"duplicateGroups"`
}

// Report is a library's health snapshot.
type Report struct {
	LibraryID  string           `json:"libraryId"`
	Counts     Counts           `json:"counts"`
	Corrupt    []Item           `json:"corrupt"`
	Orphans    []Item           `json:"orphans"`
	Unmatched  []Item           `json:"unmatched"`
	Duplicates []DuplicateGroup `json:"duplicates"`
}

// Service computes health reports over the catalog.
type Service struct {
	repo   domain.Repository
	exists func(path string) bool
}

// Option configures the service.
type Option func(*Service)

// WithExistsFunc overrides on-disk existence checking (used in tests).
func WithExistsFunc(fn func(path string) bool) Option { return func(s *Service) { s.exists = fn } }

// New builds the health service over the catalog repository.
func New(repo domain.Repository, opts ...Option) *Service {
	s := &Service{repo: repo, exists: fileExists}
	for _, o := range opts {
		o(s)
	}
	return s
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Report inspects a library's books and returns the problem categories. The library must
// exist (ErrNotFound otherwise).
func (s *Service) Report(ctx context.Context, libraryID string) (Report, error) {
	if _, err := s.repo.Libraries().Get(ctx, libraryID); err != nil {
		return Report{}, err
	}
	books, err := s.repo.Books().ListByLibrary(ctx, libraryID)
	if err != nil {
		return Report{}, err
	}

	rep := Report{
		LibraryID:  libraryID,
		Corrupt:    []Item{},
		Orphans:    []Item{},
		Unmatched:  []Item{},
		Duplicates: []DuplicateGroup{},
	}
	rep.Counts.Books = len(books)
	byHash := map[string][]Item{}

	for _, b := range books {
		it := Item{ID: b.ID, SeriesID: b.SeriesID, Number: b.Number, Title: b.Title, Path: b.FilePath}
		if b.IsCorrupt {
			rep.Counts.Corrupt++
			appendCapped(&rep.Corrupt, it)
		}
		if b.MetadataState == domain.MetaNone {
			rep.Counts.Unmatched++
			appendCapped(&rep.Unmatched, it)
		}
		if !s.exists(b.FilePath) {
			rep.Counts.Orphans++
			appendCapped(&rep.Orphans, it)
		}
		if b.ContentHash != "" {
			byHash[b.ContentHash] = append(byHash[b.ContentHash], it)
		}
	}

	for hash, items := range byHash {
		if len(items) < 2 {
			continue
		}
		rep.Counts.DuplicateGroups++
		if len(rep.Duplicates) < maxItems {
			rep.Duplicates = append(rep.Duplicates, DuplicateGroup{ContentHash: hash, Books: items})
		}
	}
	return rep, nil
}

func appendCapped(list *[]Item, it Item) {
	if len(*list) < maxItems {
		*list = append(*list, it)
	}
}
