package stats

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/pkg/ulid"
	"github.com/siposbnc/comic-hub/server/internal/store/sqlstore"
)

func day(now time.Time, offset int) time.Time { return now.AddDate(0, 0, offset) }

func TestStreaks(t *testing.T) {
	now := time.Date(2026, 7, 2, 15, 0, 0, 0, time.Local)
	yearStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)
	ms := func(t time.Time) int64 { return t.UnixMilli() }

	t.Run("empty", func(t *testing.T) {
		if c, b := streaks(nil, now, yearStart); c != 0 || b != 0 {
			t.Fatalf("empty = %d/%d, want 0/0", c, b)
		}
	})

	t.Run("run ending today", func(t *testing.T) {
		act := []int64{ms(day(now, 0)), ms(day(now, -1)), ms(day(now, -2)), ms(day(now, -5))}
		if c, b := streaks(act, now, yearStart); c != 3 || b != 3 {
			t.Fatalf("= %d/%d, want 3/3", c, b)
		}
	})

	t.Run("grace for yesterday", func(t *testing.T) {
		act := []int64{ms(day(now, -1)), ms(day(now, -2))}
		if c, _ := streaks(act, now, yearStart); c != 2 {
			t.Fatalf("current = %d, want 2 (yesterday keeps the streak alive)", c)
		}
	})

	t.Run("broken streak", func(t *testing.T) {
		act := []int64{ms(day(now, -3)), ms(day(now, -4))}
		c, b := streaks(act, now, yearStart)
		if c != 0 || b != 2 {
			t.Fatalf("= %d/%d, want current 0 / best 2", c, b)
		}
	})

	t.Run("best requires this year", func(t *testing.T) {
		// A 4-day run last November, nothing this year.
		old := time.Date(2025, 11, 10, 12, 0, 0, 0, time.Local)
		act := []int64{ms(old), ms(day(old, 1)), ms(day(old, 2)), ms(day(old, 3))}
		if c, b := streaks(act, now, yearStart); c != 0 || b != 0 {
			t.Fatalf("= %d/%d, want 0/0 (run predates this year)", c, b)
		}
	})
}

// TestSummary runs the whole aggregation against a real (SQLite) store.
func TestSummary(t *testing.T) {
	ctx := context.Background()
	db, err := sqlstore.OpenSQLite("file:" + filepath.Join(t.TempDir(), "s.db") +
		"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := sqlstore.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	store := sqlstore.NewStore(db)

	now := time.Date(2026, 7, 2, 15, 0, 0, 0, time.Local)
	lastMonth := now.AddDate(0, -1, 0)

	lib, _ := store.Libraries().Create(ctx, domain.Library{
		ID: ulid.New(), Name: "DC", Kind: "comic", Roots: []string{"/c"}, CreatedAt: 1, UpdatedAt: 1,
	})
	ser, _ := store.Series().Upsert(ctx, domain.Series{
		ID: ulid.New(), LibraryID: lib.ID, Name: "Gotham Central", SortName: "Gotham Central",
		Publisher: "DC Comics", CreatedAt: 1, UpdatedAt: 1,
	})
	mkBook := func(n string, pages int) domain.Book {
		b, err := store.Books().Upsert(ctx, domain.Book{
			ID: ulid.New(), SeriesID: ser.ID, LibraryID: lib.ID, FilePath: "/c/" + n + ".cbz",
			FileFormat: "cbz", PageCount: pages, Number: n, AddedAt: 1, UpdatedAt: 1,
		})
		if err != nil {
			t.Fatalf("book %s: %v", n, err)
		}
		return b
	}
	b1, b2, b3 := mkBook("1", 20), mkBook("2", 30), mkBook("3", 40)
	if err := store.Metadata().ReplaceBookGenres(ctx, b1.ID, []string{"Crime"}); err != nil {
		t.Fatalf("genres: %v", err)
	}
	if err := store.Metadata().ReplaceBookGenres(ctx, b2.ID, []string{"Crime", "Superhero"}); err != nil {
		t.Fatalf("genres: %v", err)
	}

	put := func(b domain.Book, status domain.ReadStatus, page int, at time.Time) {
		p := domain.Progress{
			UserID: domain.OwnerUserID, BookID: b.ID, Page: page, PageCount: b.PageCount,
			Status: status, StartedAt: at.UnixMilli(), UpdatedAt: at.UnixMilli(),
		}
		if status == domain.StatusRead {
			p.FinishedAt = at.UnixMilli()
		}
		if _, err := store.Progress().Upsert(ctx, p); err != nil {
			t.Fatalf("progress: %v", err)
		}
	}
	put(b1, domain.StatusRead, 19, now)                     // finished today
	put(b2, domain.StatusRead, 29, lastMonth)               // finished last month
	put(b3, domain.StatusInProgress, 5, now.AddDate(0, 0, -1)) // mid-read yesterday

	svc := New(store)
	svc.now = func() time.Time { return now }

	sum, err := svc.Summary(ctx, domain.OwnerUserID)
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	if sum.BooksRead != 2 || sum.PagesRead != 20+30+5 || sum.ThisYear != 2 {
		t.Fatalf("headline = %d books / %d pages / %d this year, want 2/55/2", sum.BooksRead, sum.PagesRead, sum.ThisYear)
	}
	if len(sum.Months) != 12 || sum.Months[11].Count != 1 || sum.Months[10].Count != 1 {
		t.Fatalf("months = %+v, want 1 in each of the last two buckets", sum.Months)
	}
	if sum.Months[11].Label != now.Format("Jan") {
		t.Fatalf("last bucket label = %q, want %q", sum.Months[11].Label, now.Format("Jan"))
	}
	// Day buckets: 30 entries ending today. b1 finished today lands in the last bucket;
	// b2 finished ~30 days ago (lastMonth) falls just outside the window.
	if len(sum.Days) != 30 || sum.Days[29].Count != 1 || sum.Days[29].Label != now.Format("Jan 2") {
		t.Fatalf("days last bucket = %+v, want 1 today (%q)", sum.Days[len(sum.Days)-1:], now.Format("Jan 2"))
	}
	dayTotal := 0
	for _, d := range sum.Days {
		dayTotal += d.Count
	}
	if dayTotal != 1 {
		t.Fatalf("day total = %d, want 1 (only today's finish is inside 30 days)", dayTotal)
	}
	// Reading days: today (b1) + yesterday (b3) → current streak 2.
	if sum.Streak != 2 {
		t.Fatalf("streak = %d, want 2", sum.Streak)
	}
	if len(sum.Genres) != 2 || sum.Genres[0].Name != "Crime" || sum.Genres[0].Count != 2 {
		t.Fatalf("genres = %+v, want Crime x2 first", sum.Genres)
	}
	if len(sum.Publishers) != 1 || sum.Publishers[0].Name != "DC Comics" || sum.Publishers[0].Count != 2 {
		t.Fatalf("publishers = %+v, want DC Comics x2", sum.Publishers)
	}
	if len(sum.Finished) != 2 || sum.Finished[0].BookID != b1.ID || sum.Finished[0].SeriesName != "Gotham Central" {
		t.Fatalf("finished = %+v, want b1 (newest) first", sum.Finished)
	}
}
