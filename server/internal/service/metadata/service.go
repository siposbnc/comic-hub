// Package metadata is the online-matching service: it searches metadata providers for a
// local series, ranks candidates, and applies a chosen match onto books — honoring the
// per-field locks the user has set so a match never clobbers hand-edited values.
package metadata

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/providers"
)

// Writable field names used by the /apply API and the locked-fields set.
const (
	FieldTitle       = "title"
	FieldNumber      = "number"
	FieldVolume      = "volume"
	FieldReleaseDate = "releaseDate"
	FieldAgeRating   = "ageRating"
	FieldSummary     = "summary"
	FieldCredits     = "credits"
	FieldGenres      = "genres"
	FieldCharacters  = "characters"
)

// Service applies provider metadata to the catalog.
type Service struct {
	repo      domain.Repository
	providers map[string]providers.Provider
	def       string // default provider name (first registered)
}

// New builds the service over the catalog repository and zero or more providers; the
// first provider registered is the default when a request doesn't name one.
func New(repo domain.Repository, provs ...providers.Provider) *Service {
	m := make(map[string]providers.Provider, len(provs))
	def := ""
	for _, p := range provs {
		if p == nil {
			continue
		}
		m[p.Name()] = p
		if def == "" {
			def = p.Name()
		}
	}
	return &Service{repo: repo, providers: m, def: def}
}

// Names lists the registered provider names.
func (s *Service) Names() []string {
	out := make([]string, 0, len(s.providers))
	for name := range s.providers {
		out = append(out, name)
	}
	return out
}

func (s *Service) provider(name string) (providers.Provider, error) {
	if name == "" {
		name = s.def
	}
	p, ok := s.providers[name]
	if !ok {
		return nil, fmt.Errorf("metadata: provider %q not configured", name)
	}
	return p, nil
}

// Candidates searches a provider for series matching the local series (or an explicit
// query), ranked best-first by the matching engine.
func (s *Service) Candidates(ctx context.Context, seriesID, providerName, query string) ([]providers.SeriesCandidate, error) {
	series, err := s.repo.Series().Get(ctx, seriesID)
	if err != nil {
		return nil, err
	}
	p, err := s.provider(providerName)
	if err != nil {
		return nil, err
	}
	q := strings.TrimSpace(query)
	if q == "" {
		q = series.Name
	}
	cands, err := p.SearchSeries(ctx, q)
	if err != nil {
		return nil, err
	}
	books, err := s.repo.Books().ListBySeries(ctx, seriesID)
	if err != nil {
		return nil, err
	}
	local := providers.LocalSeries{Name: series.Name, Year: series.Year, IssueCount: len(books)}
	return providers.RankSeries(local, cands), nil
}

// ApplyBook fetches one issue's metadata from a provider and writes it onto a book,
// honoring per-field locks. An empty fields slice applies every (unlocked) field.
func (s *Service) ApplyBook(ctx context.Context, bookID, providerName, issueProviderID string, fields []string) error {
	p, err := s.provider(providerName)
	if err != nil {
		return err
	}
	im, err := p.Issue(ctx, issueProviderID)
	if err != nil {
		return err
	}
	return s.applyIssueMeta(ctx, bookID, p.Name(), issueProviderID, im, fields)
}

// MatchSeries links a series to a provider volume and applies each provider issue's
// metadata to the local book with the matching issue number. progress, if non-nil, is
// called after each book (done, total).
func (s *Service) MatchSeries(ctx context.Context, seriesID, providerName, volumeProviderID string, fields []string, progress func(done, total int)) error {
	p, err := s.provider(providerName)
	if err != nil {
		return err
	}
	books, err := s.repo.Books().ListBySeries(ctx, seriesID)
	if err != nil {
		return err
	}
	issues, err := p.Issues(ctx, volumeProviderID)
	if err != nil {
		return err
	}
	byNumber := make(map[string]providers.IssueCandidate, len(issues))
	for _, iss := range issues {
		byNumber[normalizeNumber(iss.Number)] = iss
	}

	total := len(books)
	for i, b := range books {
		if err := ctx.Err(); err != nil {
			return err
		}
		if iss, ok := byNumber[normalizeNumber(b.Number)]; ok {
			im, err := p.Issue(ctx, iss.ProviderID)
			if err != nil {
				return err
			}
			if err := s.applyIssueMeta(ctx, b.ID, p.Name(), iss.ProviderID, im, fields); err != nil {
				return err
			}
		}
		if progress != nil {
			progress(i+1, total)
		}
	}
	return nil
}

// applyIssueMeta merges a provider IssueMeta onto a book: each writable field takes the
// matched value unless it is locked or excluded by `fields` (empty = all), and the
// provider link is always recorded so a later re-match reuses it.
func (s *Service) applyIssueMeta(ctx context.Context, bookID, providerName, issueProviderID string, im providers.IssueMeta, fields []string) error {
	book, err := s.repo.Books().Get(ctx, bookID)
	if err != nil {
		return err
	}
	locked, err := s.repo.Metadata().LockedBookFields(ctx, bookID)
	if err != nil {
		return err
	}
	providerIDs, err := s.repo.Metadata().BookProviderIDs(ctx, bookID)
	if err != nil {
		return err
	}

	lockedSet := toSet(locked)
	wantSet := toSet(fields)
	all := len(fields) == 0
	allow := func(field string) bool {
		return !lockedSet[field] && (all || wantSet[field])
	}

	meta := domain.BookMeta{
		Title:        book.Title,
		Number:       book.Number,
		Volume:       book.Volume,
		ReleaseDate:  book.ReleaseDate,
		AgeRating:    book.AgeRating,
		Language:     book.Language,
		Summary:      book.Summary,
		State:        domain.MetaMatched,
		ProviderIDs:  providerIDs,
		LockedFields: locked,
	}
	if allow(FieldTitle) && im.Title != "" {
		meta.Title = im.Title
	}
	if allow(FieldNumber) && im.Number != "" {
		meta.Number = im.Number
	}
	if allow(FieldVolume) && im.Volume != 0 {
		meta.Volume = im.Volume
	}
	if allow(FieldReleaseDate) && im.ReleaseDate != 0 {
		meta.ReleaseDate = im.ReleaseDate
	}
	if allow(FieldAgeRating) && im.AgeRating != "" {
		meta.AgeRating = im.AgeRating
	}
	if allow(FieldSummary) && im.Summary != "" {
		meta.Summary = im.Summary
	}
	meta.ProviderIDs[providerName] = issueProviderID

	if err := s.repo.Metadata().WriteBookMeta(ctx, bookID, meta); err != nil {
		return err
	}
	if allow(FieldCredits) && len(im.People) > 0 {
		if err := s.repo.Metadata().ReplaceBookPeople(ctx, bookID, im.People); err != nil {
			return err
		}
	}
	if allow(FieldGenres) && len(im.Genres) > 0 {
		if err := s.repo.Metadata().ReplaceBookGenres(ctx, bookID, im.Genres); err != nil {
			return err
		}
	}
	if allow(FieldCharacters) && len(im.Characters) > 0 {
		if err := s.repo.Metadata().ReplaceBookCharacters(ctx, bookID, im.Characters); err != nil {
			return err
		}
	}
	return nil
}

func toSet(xs []string) map[string]bool {
	m := make(map[string]bool, len(xs))
	for _, x := range xs {
		m[x] = true
	}
	return m
}

// normalizeNumber canonicalizes an issue number for matching ("001" and "1" and "1.0" all
// become "1"); non-numeric labels fall back to a trimmed lowercase compare.
func normalizeNumber(s string) string {
	s = strings.TrimSpace(s)
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}
	return strings.ToLower(s)
}
