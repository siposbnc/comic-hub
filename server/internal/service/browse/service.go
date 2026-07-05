// Package browse is the read side of the catalog: it composes series/book/progress
// data into the shapes the client's Home, library grid, series detail, and book detail
// screens need (docs/03-api.md §4, §8). It owns the JSON wire shapes for those reads.
package browse

import (
	"context"
	"errors"
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/siposbnc/comic-hub/server/internal/access"
	"github.com/siposbnc/comic-hub/server/internal/domain"
)

// SeriesCard is a series in the library grid.
type SeriesCard struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Year          int    `json:"year,omitempty"`
	ReadingDir    string `json:"readingDir"`
	BookCount     int    `json:"bookCount"`
	ReadCount     int    `json:"readCount"`
	CoverBookID   string `json:"coverBookId,omitempty"`
	MetadataState string `json:"metadataState,omitempty"`
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
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Year          int            `json:"year,omitempty"`
	Publisher     string         `json:"publisher,omitempty"`
	Summary       string         `json:"summary,omitempty"`
	ReadingDir    string         `json:"readingDir"`
	BookCount     int            `json:"bookCount"`
	ReadCount     int            `json:"readCount"`
	MetadataState string         `json:"metadataState,omitempty"`
	Genres        []string       `json:"genres,omitempty"`
	Characters    []string       `json:"characters,omitempty"`
	Volumes       []GroupingCard `json:"volumes,omitempty"`
	StoryArcs     []GroupingCard `json:"storyArcs,omitempty"`
	Books         []BookCard     `json:"books"`
}

// GroupingCard summarizes a browsable grouping (a story arc or a volume) on the series page.
type GroupingCard struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Year        int    `json:"year,omitempty"`
	IssueCount  int    `json:"issueCount"`
	Description string `json:"description,omitempty"`
}

// GroupingDetail is a story-arc/volume header plus its issues (the detail screen payload).
type GroupingDetail struct {
	ID          string     `json:"id"`
	Kind        string     `json:"kind"` // "arc" | "volume"
	Name        string     `json:"name"`
	SeriesID    string     `json:"seriesId"`
	SeriesName  string     `json:"seriesName"`
	Year        int        `json:"year,omitempty"`
	Description string     `json:"description,omitempty"`
	ReadingDir  string     `json:"readingDir"`
	IssueCount  int        `json:"issueCount"`
	ReadCount   int        `json:"readCount"`
	Books       []BookCard `json:"books"`
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

	// Memberships so the client can avoid offering to add the book where it already is.
	CollectionIDs  []string `json:"collectionIds,omitempty"`
	ReadingListIDs []string `json:"readingListIds,omitempty"`
}

// TagView is a tag on the book detail screen.
type TagView struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}

// NextUp is the next issue to read from the active reading list (Home screen).
type NextUp struct {
	Book     BookCard `json:"book"`
	ListID   string   `json:"listId"`
	ListName string   `json:"listName"`
}

// Discover is the Home feed.
type Discover struct {
	ContinueReading []BookCard `json:"continueReading"`
	RecentlyAdded   []BookCard `json:"recentlyAdded"`
	NextUp          *NextUp    `json:"nextUp,omitempty"`
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

// ListSeries returns the series cards for a library. For a restricted user, series are
// rebuilt from only the books they may see: a fully-restricted series is dropped entirely
// (treated as non-existent), and a partially-restricted one reports the visible count and a
// visible cover so nothing leaks.
func (s *Service) ListSeries(ctx context.Context, libraryID, userID string) ([]SeriesCard, error) {
	sums, err := s.repo.Series().Summaries(ctx, libraryID, userID)
	if err != nil {
		return nil, err
	}
	ceiling := access.CeilingFrom(ctx)
	cards := make([]SeriesCard, 0, len(sums))
	for _, su := range sums {
		card := SeriesCard{
			ID:            su.ID,
			Name:          su.Name,
			Year:          su.Year,
			ReadingDir:    string(su.ReadingDir),
			BookCount:     su.BookCount,
			ReadCount:     su.ReadCount,
			CoverBookID:   su.CoverBookID,
			MetadataState: string(su.MetadataState),
		}
		if ceiling != "" {
			visible := s.visible(ctx, s.booksOf(ctx, su.ID))
			if len(visible) == 0 {
				continue // hide series with no viewable issues
			}
			card.BookCount = len(visible)
			card.ReadCount = s.readCount(ctx, userID, visible)
			card.CoverBookID = visibleCover(su.CoverBookID, visible)
		}
		cards = append(cards, card)
	}
	return cards, nil
}

// booksOf returns a series' books straight from the repo (unfiltered).
func (s *Service) booksOf(ctx context.Context, seriesID string) []domain.Book {
	books, _ := s.repo.Books().ListBySeries(ctx, seriesID)
	return books
}

// readCount counts how many of the given books the user has finished.
func (s *Service) readCount(ctx context.Context, userID string, books []domain.Book) int {
	n := 0
	for _, b := range books {
		if p, err := s.repo.Progress().Get(ctx, userID, b.ID); err == nil && p.Status == domain.StatusRead {
			n++
		}
	}
	return n
}

// visibleCover keeps the configured cover if it's viewable, else falls back to the first
// visible book (books arrive in reading order).
func visibleCover(cover string, visible []domain.Book) string {
	for _, b := range visible {
		if b.ID == cover {
			return cover
		}
	}
	if len(visible) > 0 {
		return visible[0].ID
	}
	return ""
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
	books = s.visible(ctx, books) // hide issues above the acting user's content ceiling

	detail := SeriesDetail{
		ID:            ser.ID,
		Name:          ser.Name,
		Year:          ser.Year,
		Publisher:     ser.Publisher,
		Summary:       ser.Description,
		ReadingDir:    string(ser.ReadingDir),
		BookCount:     len(books),
		MetadataState: string(ser.MetadataState),
		Books:         make([]BookCard, 0, len(books)),
	}
	for _, b := range books {
		card := s.bookCard(ctx, b, userID)
		if card.Progress != nil && card.Progress.Status == string(domain.StatusRead) {
			detail.ReadCount++
		}
		detail.Books = append(detail.Books, card)
	}

	// Details-tab aggregates + browsable groupings (best-effort; absent until matched).
	detail.Genres, _ = s.repo.Metadata().SeriesGenres(ctx, seriesID)
	detail.Characters, _ = s.repo.Metadata().SeriesCharacters(ctx, seriesID)
	detail.Volumes = volumeCards(books)
	if arcs, err := s.repo.Metadata().SeriesStoryArcs(ctx, seriesID); err == nil {
		for _, a := range arcs {
			detail.StoryArcs = append(detail.StoryArcs, GroupingCard{
				ID: a.ID, Name: a.Name, IssueCount: a.IssueCount, Description: a.Description,
			})
		}
	}
	return detail, nil
}

// volumeCards derives browsable volumes from each book's volume number (0 = ungrouped).
// A "Volume N" lists the issues tagged with that number, earliest release year first.
func volumeCards(books []domain.Book) []GroupingCard {
	type agg struct {
		count int
		year  int
	}
	byVol := map[int]*agg{}
	var order []int
	for _, b := range books {
		if b.Volume <= 0 {
			continue
		}
		a := byVol[b.Volume]
		if a == nil {
			a = &agg{}
			byVol[b.Volume] = a
			order = append(order, b.Volume)
		}
		a.count++
		if y := yearOf(b.ReleaseDate); y > 0 && (a.year == 0 || y < a.year) {
			a.year = y
		}
	}
	sort.Ints(order)
	cards := make([]GroupingCard, 0, len(order))
	for _, v := range order {
		a := byVol[v]
		cards = append(cards, GroupingCard{
			ID:         strconv.Itoa(v),
			Name:       "Volume " + strconv.Itoa(v),
			Year:       a.year,
			IssueCount: a.count,
		})
	}
	return cards
}

// StoryArcDetail returns a story arc's header + its issues in reading order.
func (s *Service) StoryArcDetail(ctx context.Context, seriesID, arcID, userID string) (GroupingDetail, error) {
	arc, err := s.repo.Metadata().StoryArc(ctx, arcID)
	if err != nil {
		return GroupingDetail{}, err
	}
	if arc.SeriesID != seriesID {
		return GroupingDetail{}, domain.ErrNotFound
	}
	bookIDs, err := s.repo.Metadata().StoryArcBookIDs(ctx, arcID)
	if err != nil {
		return GroupingDetail{}, err
	}
	d := s.groupingDetail(ctx, seriesID, userID, "arc", arc.ID, arc.Name, 0, arc.Description, s.booksByID(ctx, bookIDs))
	return d, nil
}

// VolumeDetail returns a derived volume's header + its issues (books with that volume number).
func (s *Service) VolumeDetail(ctx context.Context, seriesID string, volume int, userID string) (GroupingDetail, error) {
	all, err := s.repo.Books().ListBySeries(ctx, seriesID)
	if err != nil {
		return GroupingDetail{}, err
	}
	all = s.visible(ctx, all)
	var books []domain.Book
	year := 0
	for _, b := range all {
		if b.Volume == volume {
			books = append(books, b)
			if y := yearOf(b.ReleaseDate); y > 0 && (year == 0 || y < year) {
				year = y
			}
		}
	}
	if len(books) == 0 {
		return GroupingDetail{}, domain.ErrNotFound
	}
	return s.groupingDetail(ctx, seriesID, userID, "volume", strconv.Itoa(volume), "Volume "+strconv.Itoa(volume), year, "", books), nil
}

// groupingDetail assembles a GroupingDetail header + its book cards (shared by arc/volume).
func (s *Service) groupingDetail(ctx context.Context, seriesID, userID, kind, id, name string, year int, desc string, books []domain.Book) GroupingDetail {
	d := GroupingDetail{
		ID: id, Kind: kind, Name: name, SeriesID: seriesID, Year: year, Description: desc,
		ReadingDir: string(domain.LTR), Books: make([]BookCard, 0, len(books)),
	}
	if ser, err := s.repo.Series().Get(ctx, seriesID); err == nil {
		d.SeriesName = ser.Name
		if ser.ReadingDir != "" {
			d.ReadingDir = string(ser.ReadingDir)
		}
	}
	for _, b := range books {
		card := s.bookCard(ctx, b, userID)
		if card.Progress != nil && card.Progress.Status == string(domain.StatusRead) {
			d.ReadCount++
		}
		d.Books = append(d.Books, card)
	}
	d.IssueCount = len(d.Books)
	return d
}

// booksByID loads books for the given ids, preserving order and skipping any that vanished.
func (s *Service) booksByID(ctx context.Context, ids []string) []domain.Book {
	out := make([]domain.Book, 0, len(ids))
	for _, id := range ids {
		if b, err := s.repo.Books().Get(ctx, id); err == nil {
			out = append(out, b)
		}
	}
	return s.visible(ctx, out)
}

// allowed reports whether the acting user (per the request's content ceiling) may see a book.
func (s *Service) allowed(ctx context.Context, b domain.Book) bool {
	return access.Allowed(access.CeilingFrom(ctx), b.AgeRating)
}

// visible drops books the acting user may not see (content restriction). A no-op for
// unrestricted users, so it stays cheap on the common path.
func (s *Service) visible(ctx context.Context, books []domain.Book) []domain.Book {
	if access.CeilingFrom(ctx) == "" {
		return books
	}
	out := make([]domain.Book, 0, len(books))
	for _, b := range books {
		if s.allowed(ctx, b) {
			out = append(out, b)
		}
	}
	return out
}

// yearOf extracts the year from an epoch-ms timestamp (0 when unset).
func yearOf(ms int64) int {
	if ms <= 0 {
		return 0
	}
	return time.UnixMilli(ms).UTC().Year()
}

// BookDetail returns the full book detail payload.
func (s *Service) BookDetail(ctx context.Context, bookID, userID string) (BookDetail, error) {
	b, err := s.repo.Books().Get(ctx, bookID)
	if err != nil {
		return BookDetail{}, err
	}
	if !s.allowed(ctx, b) {
		return BookDetail{}, domain.ErrNotFound // don't reveal restricted content exists
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

	collectionIDs, _ := s.repo.Collections().IDsForBook(ctx, bookID)
	readingListIDs, _ := s.repo.ReadingLists().IDsForBook(ctx, userID, bookID)

	return BookDetail{
		ID:             b.ID,
		SeriesID:       b.SeriesID,
		SeriesName:     seriesName,
		Number:         b.Number,
		Title:          b.Title,
		Volume:         b.Volume,
		PageCount:      b.PageCount,
		Format:         b.FileFormat,
		ReadingDir:     readingDir,
		ReleaseDate:    b.ReleaseDate,
		AgeRating:      b.AgeRating,
		Language:       b.Language,
		Summary:        b.Summary,
		IsCorrupt:      b.IsCorrupt,
		Progress:       s.progressView(ctx, b, userID),
		Credits:        credits,
		Genres:         genres,
		Characters:     characters,
		Tags:           tags,
		CollectionIDs:  collectionIDs,
		ReadingListIDs: readingListIDs,
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
	for _, b := range s.visible(ctx, books) {
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
		if err != nil || !s.allowed(ctx, b) {
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
	nextUp, _ := s.nextUpFromActiveList(ctx, userID) // best-effort; absent when no active list
	return Discover{ContinueReading: cont, RecentlyAdded: recent, NextUp: nextUp}, nil
}

// nextUpFromActiveList returns the first not-yet-read issue, in queue order, of the user's
// active reading list — the "next up" the Home screen offers. Nil when there's no active
// list or every issue is read.
func (s *Service) nextUpFromActiveList(ctx context.Context, userID string) (*NextUp, error) {
	list, err := s.repo.ReadingLists().GetActive(ctx, userID)
	if err != nil {
		return nil, err
	}
	items, err := s.repo.ReadingLists().Items(ctx, list.ID)
	if err != nil {
		return nil, err
	}
	for _, it := range items {
		if it.Stale() {
			continue // placeholder with no backing book — nothing to read
		}
		b, err := s.repo.Books().Get(ctx, it.BookID)
		if err != nil {
			continue
		}
		card := s.bookCard(ctx, b, userID)
		if card.Progress == nil || card.Progress.Status != string(domain.StatusRead) {
			return &NextUp{Book: card, ListID: list.ID, ListName: list.Name}, nil
		}
	}
	return nil, nil
}

// NextAfter returns the issue to read after bookID. context "readingList" follows the
// active list's order; anything else follows the series' issue order. Nil when there is no
// next issue (or no active list, for the list context).
func (s *Service) NextAfter(ctx context.Context, userID, bookID, context string) (*BookCard, error) {
	if context == "readingList" {
		list, err := s.repo.ReadingLists().GetActive(ctx, userID)
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		items, err := s.repo.ReadingLists().Items(ctx, list.ID)
		if err != nil {
			return nil, err
		}
		return s.cardAfter(ctx, userID, bookID, itemBookIDs(items))
	}

	cur, err := s.repo.Books().Get(ctx, bookID)
	if err != nil {
		return nil, err
	}
	books, err := s.repo.Books().ListBySeries(ctx, cur.SeriesID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(books))
	for i, b := range books {
		ids[i] = b.ID
	}
	return s.cardAfter(ctx, userID, bookID, ids)
}

// cardAfter finds bookID in an ordered id list and returns a card for the following one.
func (s *Service) cardAfter(ctx context.Context, userID, bookID string, ordered []string) (*BookCard, error) {
	for i, id := range ordered {
		if id != bookID {
			continue
		}
		if i+1 >= len(ordered) {
			return nil, nil
		}
		b, err := s.repo.Books().Get(ctx, ordered[i+1])
		if err != nil || !s.allowed(ctx, b) {
			return nil, nil
		}
		card := s.bookCard(ctx, b, userID)
		return &card, nil
	}
	return nil, nil
}

func itemBookIDs(items []domain.ReadingListItem) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		// Stale placeholders hold a queue slot but can't be read; the reading chain
		// skips over them to the next real issue.
		if !it.Stale() {
			out = append(out, it.BookID)
		}
	}
	return out
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
		if err != nil || !s.allowed(ctx, b) {
			continue // book vanished or above the content ceiling; skip
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
	if p.Status == domain.StatusRead {
		percent = 100 // a finished book is 100%, not (pageCount-1)/pageCount
	} else if p.PageCount > 0 {
		percent = math.Round(float64(p.Page)/float64(p.PageCount)*1000) / 10
	}
	return &ProgressView{Page: p.Page, Status: string(p.Status), Percent: percent, UpdatedAt: p.UpdatedAt}
}
