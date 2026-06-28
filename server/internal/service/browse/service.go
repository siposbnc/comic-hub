// Package browse is the read side of the catalog: it composes series/book/progress
// data into the shapes the client's Home, library grid, series detail, and book detail
// screens need (docs/03-api.md §4, §8). It owns the JSON wire shapes for those reads.
package browse

import (
	"context"
	"errors"
	"math"

	"github.com/siposbnc/comic-hub/server/internal/domain"
)

// SeriesCard is a series in the library grid.
type SeriesCard struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Year        int    `json:"year,omitempty"`
	ReadingDir  string `json:"readingDir"`
	BookCount   int    `json:"bookCount"`
	ReadCount   int    `json:"readCount"`
	CoverBookID string `json:"coverBookId,omitempty"`
}

// ProgressView is a user's progress for a book.
type ProgressView struct {
	Page      int     `json:"page"`
	Status    string  `json:"status"`
	Percent   float64 `json:"percent"`
	UpdatedAt int64   `json:"updatedAt"`
}

// BookCard is a book in a list/rail/grid.
type BookCard struct {
	ID        string        `json:"id"`
	SeriesID  string        `json:"seriesId"`
	Number    string        `json:"number,omitempty"`
	Title     string        `json:"title,omitempty"`
	PageCount int           `json:"pageCount"`
	Format    string        `json:"format"`
	IsCorrupt bool          `json:"isCorrupt,omitempty"`
	Progress  *ProgressView `json:"progress,omitempty"`
}

// SeriesDetail is a series header + its issues.
type SeriesDetail struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Year       int        `json:"year,omitempty"`
	Publisher  string     `json:"publisher,omitempty"`
	Summary    string     `json:"summary,omitempty"`
	ReadingDir string     `json:"readingDir"`
	BookCount  int        `json:"bookCount"`
	ReadCount  int        `json:"readCount"`
	Books      []BookCard `json:"books"`
}

// BookDetail is the book detail screen payload.
type BookDetail struct {
	ID          string        `json:"id"`
	SeriesID    string        `json:"seriesId"`
	SeriesName  string        `json:"seriesName"`
	Number      string        `json:"number,omitempty"`
	Title       string        `json:"title,omitempty"`
	Volume      int           `json:"volume,omitempty"`
	PageCount   int           `json:"pageCount"`
	Format      string        `json:"format"`
	ReadingDir  string        `json:"readingDir"`
	ReleaseDate int64         `json:"releaseDate,omitempty"`
	AgeRating   string        `json:"ageRating,omitempty"`
	Language    string        `json:"language,omitempty"`
	Summary     string        `json:"summary,omitempty"`
	IsCorrupt   bool          `json:"isCorrupt,omitempty"`
	Progress    *ProgressView `json:"progress,omitempty"`

	// Normalized metadata from an online match (omitted when absent).
	Credits    map[string][]string `json:"credits,omitempty"`
	Genres     []string            `json:"genres,omitempty"`
	Characters []string            `json:"characters,omitempty"`

	// User-applied organizational tags (omitted when none).
	Tags []TagView `json:"tags,omitempty"`
}

// TagView is a tag on the book detail screen.
type TagView struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}

// Discover is the Home feed.
type Discover struct {
	ContinueReading []BookCard `json:"continueReading"`
	RecentlyAdded   []BookCard `json:"recentlyAdded"`
}

const (
	continueLimit = 20
	recentLimit   = 24
)

// Service answers catalog read queries for a user.
type Service struct {
	repo domain.Repository
}

// New constructs the browse service.
func New(repo domain.Repository) *Service { return &Service{repo: repo} }

// ListSeries returns the series cards for a library.
func (s *Service) ListSeries(ctx context.Context, libraryID, userID string) ([]SeriesCard, error) {
	sums, err := s.repo.Series().Summaries(ctx, libraryID, userID)
	if err != nil {
		return nil, err
	}
	cards := make([]SeriesCard, 0, len(sums))
	for _, su := range sums {
		cards = append(cards, SeriesCard{
			ID:          su.ID,
			Name:        su.Name,
			Year:        su.Year,
			ReadingDir:  string(su.ReadingDir),
			BookCount:   su.BookCount,
			ReadCount:   su.ReadCount,
			CoverBookID: su.CoverBookID,
		})
	}
	return cards, nil
}

// SeriesDetail returns a series header + its issues with per-book progress.
func (s *Service) SeriesDetail(ctx context.Context, seriesID, userID string) (SeriesDetail, error) {
	ser, err := s.repo.Series().Get(ctx, seriesID)
	if err != nil {
		return SeriesDetail{}, err
	}
	books, err := s.repo.Books().ListBySeries(ctx, seriesID)
	if err != nil {
		return SeriesDetail{}, err
	}

	detail := SeriesDetail{
		ID:         ser.ID,
		Name:       ser.Name,
		Year:       ser.Year,
		Publisher:  ser.Publisher,
		Summary:    ser.Description,
		ReadingDir: string(ser.ReadingDir),
		BookCount:  len(books),
		Books:      make([]BookCard, 0, len(books)),
	}
	for _, b := range books {
		card := s.bookCard(ctx, b, userID)
		if card.Progress != nil && card.Progress.Status == string(domain.StatusRead) {
			detail.ReadCount++
		}
		detail.Books = append(detail.Books, card)
	}
	return detail, nil
}

// BookDetail returns the full book detail payload.
func (s *Service) BookDetail(ctx context.Context, bookID, userID string) (BookDetail, error) {
	b, err := s.repo.Books().Get(ctx, bookID)
	if err != nil {
		return BookDetail{}, err
	}

	seriesName := ""
	readingDir := string(domain.LTR)
	if ser, err := s.repo.Series().Get(ctx, b.SeriesID); err == nil {
		seriesName = ser.Name
		if ser.ReadingDir != "" {
			readingDir = string(ser.ReadingDir)
		}
	}

	// Normalized credits/genres/characters from an online match (best-effort).
	credits, _ := s.repo.Metadata().BookCredits(ctx, bookID)
	genres, _ := s.repo.Metadata().BookGenres(ctx, bookID)
	characters, _ := s.repo.Metadata().BookCharacters(ctx, bookID)

	var tags []TagView
	if ts, err := s.repo.Tags().BookTags(ctx, bookID); err == nil {
		for _, t := range ts {
			tags = append(tags, TagView{ID: t.ID, Name: t.Name, Color: t.Color})
		}
	}

	return BookDetail{
		ID:          b.ID,
		SeriesID:    b.SeriesID,
		SeriesName:  seriesName,
		Number:      b.Number,
		Title:       b.Title,
		Volume:      b.Volume,
		PageCount:   b.PageCount,
		Format:      b.FileFormat,
		ReadingDir:  readingDir,
		ReleaseDate: b.ReleaseDate,
		AgeRating:   b.AgeRating,
		Language:    b.Language,
		Summary:     b.Summary,
		IsCorrupt:   b.IsCorrupt,
		Progress:    s.progressView(ctx, b, userID),
		Credits:     credits,
		Genres:      genres,
		Characters:  characters,
		Tags:        tags,
	}, nil
}

// RecentBooks returns the most recently added books in a library (or all libraries when
// libraryID is empty).
func (s *Service) RecentBooks(ctx context.Context, libraryID, userID string, limit int) ([]BookCard, error) {
	if limit <= 0 {
		limit = recentLimit
	}
	books, err := s.repo.Books().ListRecent(ctx, libraryID, limit)
	if err != nil {
		return nil, err
	}
	cards := make([]BookCard, 0, len(books))
	for _, b := range books {
		cards = append(cards, s.bookCard(ctx, b, userID))
	}
	return cards, nil
}

// BooksByIDs returns book cards for the given ids, preserving the input order and
// silently skipping ids that no longer resolve (e.g. a book removed since it was listed).
// Used to render the ordered contents of a collection or reading list.
func (s *Service) BooksByIDs(ctx context.Context, ids []string, userID string) ([]BookCard, error) {
	cards := make([]BookCard, 0, len(ids))
	for _, id := range ids {
		b, err := s.repo.Books().Get(ctx, id)
		if err != nil {
			continue
		}
		cards = append(cards, s.bookCard(ctx, b, userID))
	}
	return cards, nil
}

// Discover builds the Home feed: Continue Reading + Recently Added.
func (s *Service) Discover(ctx context.Context, libraryID, userID string) (Discover, error) {
	cont, err := s.continueReading(ctx, userID)
	if err != nil {
		return Discover{}, err
	}
	recent, err := s.RecentBooks(ctx, libraryID, userID, recentLimit)
	if err != nil {
		return Discover{}, err
	}
	return Discover{ContinueReading: cont, RecentlyAdded: recent}, nil
}

// ContinueReading returns the user's in-progress books, most-recent first.
func (s *Service) ContinueReading(ctx context.Context, userID string) ([]BookCard, error) {
	return s.continueReading(ctx, userID)
}

func (s *Service) continueReading(ctx context.Context, userID string) ([]BookCard, error) {
	progs, err := s.repo.Progress().ContinueReading(ctx, userID, continueLimit)
	if err != nil {
		return nil, err
	}
	cards := make([]BookCard, 0, len(progs))
	for _, p := range progs {
		b, err := s.repo.Books().Get(ctx, p.BookID)
		if err != nil {
			continue // book vanished; skip
		}
		card := s.bookCard(ctx, b, userID)
		card.Progress = toProgressView(p)
		cards = append(cards, card)
	}
	return cards, nil
}

func (s *Service) bookCard(ctx context.Context, b domain.Book, userID string) BookCard {
	return BookCard{
		ID:        b.ID,
		SeriesID:  b.SeriesID,
		Number:    b.Number,
		Title:     b.Title,
		PageCount: b.PageCount,
		Format:    b.FileFormat,
		IsCorrupt: b.IsCorrupt,
		Progress:  s.progressView(ctx, b, userID),
	}
}

func (s *Service) progressView(ctx context.Context, b domain.Book, userID string) *ProgressView {
	p, err := s.repo.Progress().Get(ctx, userID, b.ID)
	if errors.Is(err, domain.ErrNotFound) || err != nil {
		return nil
	}
	return toProgressView(p)
}

func toProgressView(p domain.Progress) *ProgressView {
	percent := 0.0
	if p.PageCount > 0 {
		percent = math.Round(float64(p.Page)/float64(p.PageCount)*1000) / 10
	}
	return &ProgressView{Page: p.Page, Status: string(p.Status), Percent: percent, UpdatedAt: p.UpdatedAt}
}
