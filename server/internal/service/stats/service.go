// Package stats aggregates a user's reading into the dashboard summary (Phase 3,
// Milestone G): headline numbers, issues-per-month buckets, streaks, top genres and
// publishers, and recently finished issues. The repository returns raw counts and
// timestamps; the calendar math lives here (in server-local time) so the store SQL
// stays dialect-portable.
package stats

import (
	"context"
	"sort"
	"time"

	"github.com/siposbnc/comic-hub/server/internal/domain"
)

const (
	topLimit      = 8
	finishedLimit = 8
	monthsWindow  = 12
)

// MonthCount is one bar of the issues-per-month chart, oldest first.
type MonthCount struct {
	Label string // short month name, e.g. "Jul"
	Count int
}

// Summary is the per-user dashboard payload.
type Summary struct {
	BooksRead  int
	PagesRead  int
	ThisYear   int
	Streak     int // consecutive reading days ending today/yesterday
	BestStreak int // longest run this year
	Months     []MonthCount
	Genres     []domain.NameCount
	Publishers []domain.NameCount
	Finished   []domain.FinishedBook
}

// Service computes reading stats over the domain repository.
type Service struct {
	repo domain.Repository
	now  func() time.Time // injectable for tests
}

func New(repo domain.Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

// Summary assembles the dashboard for one user.
func (s *Service) Summary(ctx context.Context, userID string) (Summary, error) {
	now := s.now()
	out := Summary{
		Months:     make([]MonthCount, 0, monthsWindow),
		Genres:     []domain.NameCount{},
		Publishers: []domain.NameCount{},
		Finished:   []domain.FinishedBook{},
	}

	var err error
	if out.BooksRead, out.PagesRead, err = s.repo.Stats().ReadCounts(ctx, userID); err != nil {
		return Summary{}, err
	}

	// Month buckets: the current month and the 11 before it, oldest first.
	windowStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).
		AddDate(0, -(monthsWindow - 1), 0)
	finished, err := s.repo.Stats().FinishedTimes(ctx, userID, windowStart.UnixMilli())
	if err != nil {
		return Summary{}, err
	}
	buckets := map[string]int{}
	yearStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
	for _, ms := range finished {
		t := time.UnixMilli(ms).In(now.Location())
		buckets[t.Format("2006-01")]++
		if !t.Before(yearStart) {
			out.ThisYear++
		}
	}
	for i := 0; i < monthsWindow; i++ {
		m := windowStart.AddDate(0, i, 0)
		out.Months = append(out.Months, MonthCount{Label: m.Format("Jan"), Count: buckets[m.Format("2006-01")]})
	}

	// Streaks over the set of reading days. The catalog keeps each book's latest
	// progress (started/finished/updated), not a full diary, so this is a floor —
	// mid-book days between updates aren't recorded. Good enough for a calm stat.
	activity, err := s.repo.Stats().ActivityTimes(ctx, userID)
	if err != nil {
		return Summary{}, err
	}
	out.Streak, out.BestStreak = streaks(activity, now, yearStart)

	if out.Genres, err = s.repo.Stats().TopGenres(ctx, userID, topLimit); err != nil {
		return Summary{}, err
	}
	if out.Publishers, err = s.repo.Stats().TopPublishers(ctx, userID, topLimit); err != nil {
		return Summary{}, err
	}
	if out.Finished, err = s.repo.Stats().RecentlyFinished(ctx, userID, finishedLimit); err != nil {
		return Summary{}, err
	}
	if out.Genres == nil {
		out.Genres = []domain.NameCount{}
	}
	if out.Publishers == nil {
		out.Publishers = []domain.NameCount{}
	}
	if out.Finished == nil {
		out.Finished = []domain.FinishedBook{}
	}
	return out, nil
}

// streaks returns (current run ending today or yesterday, longest run starting within
// this year). Days are calendar days in now's location.
func streaks(activity []int64, now time.Time, yearStart time.Time) (current, best int) {
	if len(activity) == 0 {
		return 0, 0
	}
	daySet := map[int64]bool{}
	for _, ms := range activity {
		t := time.UnixMilli(ms).In(now.Location())
		d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, now.Location())
		daySet[d.Unix()] = true
	}
	days := make([]int64, 0, len(daySet))
	for d := range daySet {
		days = append(days, d)
	}
	sort.Slice(days, func(i, j int) bool { return days[i] < days[j] })

	const day = 24 * 60 * 60 // DST shifts don't matter at day granularity with Unix dates from midnight
	// Longest run whose first day is in this year.
	run := 1
	for i := 1; i <= len(days); i++ {
		if i < len(days) && days[i]-days[i-1] <= day+3600 && days[i] != days[i-1] {
			run++
			continue
		}
		start := days[i-run]
		if start >= yearStart.Unix() && run > best {
			best = run
		}
		run = 1
	}

	// Current run: walk back from today (or yesterday — reading later today shouldn't
	// show a broken streak at breakfast).
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	cursor := today
	if !daySet[cursor.Unix()] {
		cursor = cursor.AddDate(0, 0, -1)
	}
	for daySet[cursor.Unix()] {
		current++
		cursor = cursor.AddDate(0, 0, -1)
	}
	if current > best {
		best = current
	}
	return current, best
}
