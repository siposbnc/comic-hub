// Package metadata is the online-matching service: it searches metadata providers for a
// local series, ranks candidates, and applies a chosen match onto books — honoring the
// per-field locks the user has set so a match never clobbers hand-edited values.
package metadata

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

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

// Service applies provider metadata to the catalog. Its provider set can be reconfigured
// at runtime (when credentials change in settings), so reads are guarded by a mutex.
type Service struct {
	repo domain.Repository

	mu        sync.RWMutex
	providers map[string]providers.Provider
	order     []string // registration order; order[0] is the default
	def       string   // default provider name (first registered)

	// afterApply, if set, runs after a book's metadata is written (e.g. to export a
	// ComicInfo.xml sidecar). Optional.
	afterApply func(ctx context.Context, bookID string)
}

// OnApply registers a hook fired after each book's metadata is applied.
func (s *Service) OnApply(fn func(ctx context.Context, bookID string)) { s.afterApply = fn }

// New builds the service over the catalog repository and zero or more providers; the
// first provider registered is the default when a request doesn't name one.
func New(repo domain.Repository, provs ...providers.Provider) *Service {
	s := &Service{repo: repo}
	s.Configure(provs...)
	return s
}

// Configure replaces the service's provider set (used at startup and whenever provider
// credentials change). Safe to call concurrently with matching.
func (s *Service) Configure(provs ...providers.Provider) {
	m := make(map[string]providers.Provider, len(provs))
	var order []string
	def := ""
	for _, p := range provs {
		if p == nil {
			continue
		}
		if _, dup := m[p.Name()]; !dup {
			order = append(order, p.Name())
		}
		m[p.Name()] = p
		if def == "" {
			def = p.Name()
		}
	}
	s.mu.Lock()
	s.providers, s.order, s.def = m, order, def
	s.mu.Unlock()
}

// Names lists the registered provider names.
func (s *Service) Names() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0, len(s.providers))
	for name := range s.providers {
		out = append(out, name)
	}
	return out
}

// Has reports whether a named provider is currently configured.
func (s *Service) Has(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.providers[name]
	return ok
}

func (s *Service) provider(name string) (providers.Provider, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if name == "" {
		name = s.def
	}
	p, ok := s.providers[name]
	if !ok {
		return nil, fmt.Errorf("metadata: provider %q not configured", name)
	}
	return p, nil
}

// Candidates searches for series matching the local series (or an explicit query), ranked
// best-first. With an empty providerName it searches every configured provider and merges
// the results (each candidate tagged with its source), so the picker shows the best hit
// across providers; a non-empty providerName restricts to that one. A provider that errors
// (e.g. transient network) is skipped rather than failing the whole search.
func (s *Service) Candidates(ctx context.Context, seriesID, providerName, query string) ([]providers.SeriesCandidate, error) {
	series, err := s.repo.Series().Get(ctx, seriesID)
	if err != nil {
		return nil, err
	}

	var search []providers.Provider
	if providerName == "" {
		search = s.allProviders()
		if len(search) == 0 {
			return nil, fmt.Errorf("metadata: no provider configured")
		}
	} else {
		p, err := s.provider(providerName)
		if err != nil {
			return nil, err
		}
		search = []providers.Provider{p}
	}

	q := strings.TrimSpace(query)
	if q == "" {
		q = series.Name
	}

	var merged []providers.SeriesCandidate
	var firstErr error
	for _, p := range search {
		cands, err := p.SearchSeries(ctx, q)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		for i := range cands {
			cands[i].Provider = p.Name()
		}
		merged = append(merged, cands...)
	}
	// Surface an error only when every provider failed (so one provider's outage doesn't
	// hide another's results).
	if len(merged) == 0 && firstErr != nil {
		return nil, firstErr
	}

	books, err := s.repo.Books().ListBySeries(ctx, seriesID)
	if err != nil {
		return nil, err
	}
	local := providers.LocalSeries{Name: series.Name, Year: series.Year, IssueCount: len(books)}
	return providers.RankSeries(local, merged), nil
}

// allProviders returns the configured providers in registration order.
func (s *Service) allProviders() []providers.Provider {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]providers.Provider, 0, len(s.order))
	for _, name := range s.order {
		if p, ok := s.providers[name]; ok {
			out = append(out, p)
		}
	}
	return out
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

// MatchSeries links a series to a provider volume: it writes the series-level metadata
// (publisher/year/description + the provider link, state=matched) and applies each provider
// issue's metadata to the local book with the matching issue number. progress, if non-nil,
// is called after each book (done, total).
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

	// Series-level metadata (best-effort: a provider that can't supply it still matches issues).
	sm, smErr := p.SeriesMeta(ctx, volumeProviderID)
	if smErr != nil {
		return smErr
	}
	if err := s.repo.Series().WriteMatch(ctx, seriesID, domain.SeriesMatch{
		Publisher:   sm.Publisher,
		Year:        sm.Year,
		Description: sm.Description,
		State:       domain.MetaMatched,
		Provider:    p.Name(),
		ProviderID:  volumeProviderID,
	}); err != nil {
		return err
	}

	byNumber := make(map[string]providers.IssueCandidate, len(issues))
	for _, iss := range issues {
		byNumber[normalizeNumber(iss.Number)] = iss
	}

	// Accumulate story-arc membership across the matched issues (de-duped by provider arc
	// id, encounter order preserved) to rebuild the series' arcs after applying issues.
	arcByID := make(map[string]*domain.StoryArcInput)
	var arcOrder []string

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
			// Genres are a series-level concept for most providers (e.g. Metron exposes them
			// on the series, not the issue) — fall back to the series' genres so the Details
			// tab is populated.
			if len(im.Genres) == 0 {
				im.Genres = sm.Genres
			}
			if err := s.applyIssueMeta(ctx, b.ID, p.Name(), iss.ProviderID, im, fields); err != nil {
				return err
			}
			for _, a := range im.StoryArcs {
				ai := arcByID[a.ProviderID]
				if ai == nil {
					ai = &domain.StoryArcInput{ProviderID: a.ProviderID, Name: a.Name}
					arcByID[a.ProviderID] = ai
					arcOrder = append(arcOrder, a.ProviderID)
				}
				ai.BookIDs = append(ai.BookIDs, b.ID)
			}
		}
		if progress != nil {
			progress(i+1, total)
		}
	}

	// Rebuild the series' story arcs (always — an empty slice clears stale arcs).
	arcs := make([]domain.StoryArcInput, 0, len(arcOrder))
	for _, id := range arcOrder {
		arcs = append(arcs, *arcByID[id])
	}
	return s.repo.Metadata().ReplaceSeriesStoryArcs(ctx, seriesID, arcs)
}

// HasProviders reports whether any metadata provider is configured (online matching is
// possible). The scan pipeline checks this before enqueuing an auto-match.
func (s *Service) HasProviders() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.providers) > 0
}

// autoMatchThreshold is the minimum candidate score to auto-apply without user confirmation.
// 1.0 means an exact, fully-confident match (see providers.ScoreSeries).
const autoMatchThreshold = 1.0

// AutoMatchSeries searches every configured provider for a series and, if the best
// candidate across them is a 100%-confidence match, applies it (from whichever provider it
// came); otherwise the series is flagged incomplete so the user can match it manually. A
// series with no configured provider is left untouched.
func (s *Service) AutoMatchSeries(ctx context.Context, seriesID string) error {
	if !s.HasProviders() {
		return nil
	}
	cands, err := s.Candidates(ctx, seriesID, "", "")
	if err != nil {
		return err
	}
	if len(cands) > 0 && cands[0].Score >= autoMatchThreshold {
		return s.MatchSeries(ctx, seriesID, cands[0].Provider, cands[0].ProviderID, nil, nil)
	}
	return s.repo.Series().SetMetadataState(ctx, seriesID, domain.MetaIncomplete)
}

// AutoMatchLibrary auto-matches every not-yet-matched series in a library (state "none"):
// already-matched series and ones already flagged incomplete are skipped so repeated scans
// don't re-hit the provider. progress, if non-nil, is called after each series.
func (s *Service) AutoMatchLibrary(ctx context.Context, libraryID string, progress func(done, total int)) error {
	if !s.HasProviders() {
		return nil
	}
	all, err := s.repo.Series().ListByLibrary(ctx, libraryID)
	if err != nil {
		return err
	}
	pending := make([]domain.Series, 0, len(all))
	for _, ser := range all {
		if ser.MetadataState == "" || ser.MetadataState == domain.MetaNone {
			pending = append(pending, ser)
		}
	}
	total := len(pending)
	for i, ser := range pending {
		if err := ctx.Err(); err != nil {
			return err
		}
		// A single series' failure (e.g. a transient provider error) shouldn't abort the
		// whole library; leave it as-is to retry on a later scan.
		_ = s.AutoMatchSeries(ctx, ser.ID)
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
	if s.afterApply != nil {
		s.afterApply(ctx, bookID)
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
