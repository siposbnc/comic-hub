package organize

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/siposbnc/comic-hub/server/internal/access"
	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/pkg/ulid"
)

// TrackerIssue is one cell in the tracker matrix. Library issues carry a BookID (their read
// state lives in read_progress); overlay issues carry an ID (a track_issue id) and no book.
type TrackerIssue struct {
	ID     string // track_issue id (overlay only; empty for a library issue)
	Number string
	Sort   float64
	BookID string // library file id (empty for an overlay issue)
	State  string // "read" | "reading" | "unread"
	Page   int
	Pages  int
	Source string // "library" | "manual"
}

// TrackerTrack is one row: a library series (Link "library", SeriesID set) or a standalone
// track (Link "manual"). ID is prefixed ("series:<id>" / "track:<id>") so the two id spaces
// never collide on the client.
type TrackerTrack struct {
	ID        string
	SeriesID  string
	LibraryID string // owning library (empty for a standalone track)
	Name      string
	Special   string // special-edition label ("Annual", "One-Shot", …); empty for a normal row
	Link      string // "library" | "manual"
	Issues    []TrackerIssue
}

// Tracker assembles the per-user reading matrix: every library series (projected live from
// the catalog + progress) merged with the user's overlay issues, plus their standalone
// tracks. Content-restricted users only see issues within their age ceiling.
func (s *Service) Tracker(ctx context.Context, userID string) ([]TrackerTrack, error) {
	overlay, err := s.repo.Tracks().OverlayIssues(ctx, userID)
	if err != nil {
		return nil, err
	}
	bySeries := map[string][]domain.TrackIssue{}
	byTrack := map[string][]domain.TrackIssue{}
	for _, it := range overlay {
		if it.SeriesID != "" {
			bySeries[it.SeriesID] = append(bySeries[it.SeriesID], it)
		} else if it.TrackID != "" {
			byTrack[it.TrackID] = append(byTrack[it.TrackID], it)
		}
	}

	ceiling := access.CeilingFrom(ctx)
	tracks := make([]TrackerTrack, 0)

	// Library series, one track each.
	libs, err := s.repo.Libraries().List(ctx)
	if err != nil {
		return nil, err
	}
	for _, lib := range libs {
		series, err := s.repo.Series().ListByLibrary(ctx, lib.ID)
		if err != nil {
			return nil, err
		}
		for _, ser := range series {
			books, err := s.repo.Books().ListBySeries(ctx, ser.ID)
			if err != nil {
				return nil, err
			}
			main := make([]TrackerIssue, 0, len(books))
			seen := map[float64]bool{}
			specials := map[domain.BookKind][]domain.Book{}
			for _, b := range books {
				if !access.Allowed(ceiling, b.AgeRating) {
					continue
				}
				if b.Kind.IsExtra() {
					continue // variants/covers aren't issues — keep them out of the matrix
				}
				if b.Kind.IsSpecial() {
					specials[b.Kind] = append(specials[b.Kind], b)
					continue
				}
				sortKey := issueSort(b.SortNumber, b.Number)
				seen[sortKey] = true
				main = append(main, s.libraryIssue(ctx, userID, b, sortKey, displayNumber(b.Number, sortKey)))
			}
			// Merge the user's overlay (gap) issues into the main run, skipping numbers a real
			// book already occupies.
			for _, it := range bySeries[ser.ID] {
				if seen[it.Sort] {
					continue
				}
				seen[it.Sort] = true
				main = append(main, overlayIssue(it))
			}
			if len(main) > 0 {
				sortIssues(main)
				tracks = append(tracks, TrackerTrack{
					ID:        "series:" + ser.ID,
					SeriesID:  ser.ID,
					LibraryID: lib.ID,
					Name:      ser.Name,
					Link:      "library",
					Issues:    main,
				})
			}
			// Each special edition (annual, one-shot, …) becomes its own sub-row, re-numbered
			// locally from 1 — exactly as a collector's wall-chart splits them out.
			for _, kind := range specialKindOrder {
				group := specials[kind]
				if len(group) == 0 {
					continue
				}
				sort.SliceStable(group, func(i, j int) bool { return group[i].SortNumber < group[j].SortNumber })
				issues := make([]TrackerIssue, 0, len(group))
				for i, b := range group {
					local := parseSort(b.Number)
					if local == 0 {
						local = float64(i + 1)
					}
					issues = append(issues, s.libraryIssue(ctx, userID, b, local, trimFloat(local)))
				}
				tracks = append(tracks, TrackerTrack{
					ID:        "series:" + ser.ID + ":" + string(kind),
					SeriesID:  ser.ID,
					LibraryID: lib.ID,
					Name:      ser.Name,
					Special:   specialLabel(kind),
					Link:      "library",
					Issues:    issues,
				})
			}
		}
	}

	// Standalone tracks.
	standalone, err := s.repo.Tracks().ListTracks(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, t := range standalone {
		raw := byTrack[t.ID]
		issues := make([]TrackerIssue, 0, len(raw))
		for _, it := range raw {
			issues = append(issues, overlayIssue(it))
		}
		sortIssues(issues)
		tracks = append(tracks, TrackerTrack{
			ID:       "track:" + t.ID,
			SeriesID: "",
			Name:     t.Name,
			Link:     "manual",
			Issues:   issues,
		})
	}

	sort.SliceStable(tracks, func(i, j int) bool {
		ai, aj := strings.ToLower(tracks[i].Name), strings.ToLower(tracks[j].Name)
		if ai != aj {
			return ai < aj
		}
		// Same series name: the main row sorts before its special sub-rows.
		return tracks[i].Special == "" && tracks[j].Special != ""
	})
	return tracks, nil
}

// specialKindOrder is the order special sub-rows appear under their parent series.
var specialKindOrder = []domain.BookKind{
	domain.KindAnnual, domain.KindOneShot, domain.KindSpecial, domain.KindTPB, domain.KindGN,
}

// specialLabel is the human sub-row suffix for a special kind ("{Series} — Annual").
func specialLabel(kind domain.BookKind) string {
	switch kind {
	case domain.KindAnnual:
		return "Annual"
	case domain.KindOneShot:
		return "One-Shot"
	case domain.KindSpecial:
		return "Special"
	case domain.KindTPB:
		return "TPB"
	case domain.KindGN:
		return "GN"
	default:
		return string(kind)
	}
}

// libraryIssue builds a tracker cell for a library book at the given sort key / display label.
func (s *Service) libraryIssue(ctx context.Context, userID string, b domain.Book, sortKey float64, number string) TrackerIssue {
	return TrackerIssue{
		Number: number,
		Sort:   sortKey,
		BookID: b.ID,
		State:  s.bookState(ctx, userID, b.ID),
		Pages:  b.PageCount,
		Source: "library",
		Page:   s.bookPage(ctx, userID, b.ID),
	}
}

// bookState maps a user's progress on a book to a tracker cell state.
func (s *Service) bookState(ctx context.Context, userID, bookID string) string {
	p, err := s.repo.Progress().Get(ctx, userID, bookID)
	if err != nil {
		return "unread"
	}
	switch p.Status {
	case domain.StatusRead:
		return "read"
	case domain.StatusInProgress:
		return "reading"
	default:
		return "unread"
	}
}

func (s *Service) bookPage(ctx context.Context, userID, bookID string) int {
	p, err := s.repo.Progress().Get(ctx, userID, bookID)
	if err != nil {
		return 0
	}
	return p.Page
}

func overlayIssue(it domain.TrackIssue) TrackerIssue {
	state := "unread"
	if it.Read {
		state = "read"
	}
	return TrackerIssue{
		ID:     it.ID,
		Number: it.Number,
		Sort:   it.Sort,
		State:  state,
		Source: "manual",
	}
}

func sortIssues(issues []TrackerIssue) {
	sort.SliceStable(issues, func(i, j int) bool { return issues[i].Sort < issues[j].Sort })
}

// issueSort resolves a book's numeric sort key, falling back to parsing its number string
// when the scanner left sort_number at 0.
func issueSort(sortNumber float64, number string) float64 {
	if sortNumber != 0 {
		return sortNumber
	}
	return parseSort(number)
}

// displayNumber prefers the stored number string, falling back to the sort key so a cell is
// never blank.
func displayNumber(number string, sortKey float64) string {
	number = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(number), "#"))
	if number != "" {
		return number
	}
	return trimFloat(sortKey)
}

// parseSort extracts a leading number from an issue label ("23.1" → 23.1, "#7" → 7,
// "Annual 2" → 2), returning 0 when there's no number.
func parseSort(number string) float64 {
	number = strings.TrimSpace(number)
	number = strings.TrimPrefix(number, "#")
	var b strings.Builder
	for _, r := range number {
		if (r >= '0' && r <= '9') || r == '.' {
			b.WriteRune(r)
		} else if b.Len() > 0 {
			break
		}
	}
	f, _ := strconv.ParseFloat(b.String(), 64)
	return f
}

func trimFloat(f float64) string {
	s := strconv.FormatFloat(f, 'f', -1, 64)
	return s
}

// ── Tracker mutations ────────────────────────────────────────────────────────────────

// CreateTrack adds a standalone (manual) track owned by userID.
func (s *Service) CreateTrack(ctx context.Context, userID, name string) (domain.Track, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.Track{}, fmt.Errorf("%w: name is required", domain.ErrValidation)
	}
	now := time.Now().UnixMilli()
	t := domain.Track{ID: ulid.New(), UserID: userID, Name: name, CreatedAt: now, UpdatedAt: now}
	return s.repo.Tracks().CreateTrack(ctx, t)
}

// RenameTrack updates a standalone track's name (its only editable field).
func (s *Service) RenameTrack(ctx context.Context, userID, id, name string) (domain.Track, error) {
	t, err := s.repo.Tracks().GetTrack(ctx, userID, id)
	if err != nil {
		return domain.Track{}, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.Track{}, fmt.Errorf("%w: name cannot be empty", domain.ErrValidation)
	}
	t.Name = name
	t.UpdatedAt = time.Now().UnixMilli()
	if err := s.repo.Tracks().RenameTrack(ctx, t); err != nil {
		return domain.Track{}, err
	}
	return t, nil
}

// DeleteTrack removes a standalone track and its overlay issues.
func (s *Service) DeleteTrack(ctx context.Context, userID, id string) error {
	return s.repo.Tracks().DeleteTrack(ctx, userID, id)
}

// AddTrackIssues adds overlay issues to a standalone track (trackID set) or a library series
// (seriesID set) — exactly one target. Numbers are issue labels ("24", "23.1", "24-52" is
// expanded by the caller); each becomes a gap cell with its parsed sort key.
func (s *Service) AddTrackIssues(ctx context.Context, userID, trackID, seriesID string, numbers []string) error {
	trackID = strings.TrimSpace(trackID)
	seriesID = strings.TrimSpace(seriesID)
	if (trackID == "") == (seriesID == "") {
		return fmt.Errorf("%w: exactly one of trackId or seriesId is required", domain.ErrValidation)
	}
	if trackID != "" {
		if _, err := s.repo.Tracks().GetTrack(ctx, userID, trackID); err != nil {
			return err
		}
	} else {
		if _, err := s.repo.Series().Get(ctx, seriesID); err != nil {
			return err
		}
	}
	now := time.Now().UnixMilli()
	issues := make([]domain.TrackIssue, 0, len(numbers))
	for _, n := range numbers {
		n = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(n), "#"))
		if n == "" {
			continue
		}
		issues = append(issues, domain.TrackIssue{
			ID:        ulid.New(),
			UserID:    userID,
			TrackID:   trackID,
			SeriesID:  seriesID,
			Number:    n,
			Sort:      parseSort(n),
			CreatedAt: now,
		})
	}
	if len(issues) == 0 {
		return fmt.Errorf("%w: at least one issue number is required", domain.ErrValidation)
	}
	return s.repo.Tracks().AddIssues(ctx, issues)
}

// RemoveTrackIssue deletes one overlay issue.
func (s *Service) RemoveTrackIssue(ctx context.Context, userID, id string) error {
	return s.repo.Tracks().RemoveIssue(ctx, userID, id)
}

// MarkTrackIssue flips an overlay issue's read flag.
func (s *Service) MarkTrackIssue(ctx context.Context, userID, id string, read bool) error {
	if _, err := s.repo.Tracks().GetIssue(ctx, userID, id); err != nil {
		return err
	}
	at := int64(0)
	if read {
		at = time.Now().UnixMilli()
	}
	return s.repo.Tracks().SetIssueRead(ctx, userID, id, read, at)
}
