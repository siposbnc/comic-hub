package http

import (
	"net/http"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/service/stats"
)

// The stats wire shape mirrors the design_handoff_stats data contract
// (docs/03-api.md §6): headline numbers, 12 month buckets oldest-first, ranked
// genres/publishers, and the recently-finished cover rail.

type statsMonthDTO struct {
	Label string `json:"m"`
	Count int    `json:"n"`
}

type statsNameCountDTO struct {
	Name  string `json:"name"`
	Count int    `json:"n"`
}

type statsFinishedDTO struct {
	BookID     string `json:"bookId"`
	Title      string `json:"title,omitempty"`
	Number     string `json:"number,omitempty"`
	SeriesID   string `json:"seriesId"`
	SeriesName string `json:"seriesName"`
	FinishedAt int64  `json:"finishedAt"`
}

type statsDTO struct {
	BooksRead  int                 `json:"booksRead"`
	PagesRead  int                 `json:"pagesRead"`
	ThisYear   int                 `json:"thisYear"`
	Streak     int                 `json:"streak"`
	BestStreak int                 `json:"bestStreak"`
	Months     []statsMonthDTO     `json:"months"`
	Genres     []statsNameCountDTO `json:"genres"`
	Publishers []statsNameCountDTO `json:"publishers"`
	Finished   []statsFinishedDTO  `json:"finished"`
}

func toStatsDTO(s stats.Summary) statsDTO {
	dto := statsDTO{
		BooksRead: s.BooksRead, PagesRead: s.PagesRead, ThisYear: s.ThisYear,
		Streak: s.Streak, BestStreak: s.BestStreak,
		Months:     make([]statsMonthDTO, 0, len(s.Months)),
		Genres:     nameCounts(s.Genres),
		Publishers: nameCounts(s.Publishers),
		Finished:   make([]statsFinishedDTO, 0, len(s.Finished)),
	}
	for _, m := range s.Months {
		dto.Months = append(dto.Months, statsMonthDTO{Label: m.Label, Count: m.Count})
	}
	for _, f := range s.Finished {
		dto.Finished = append(dto.Finished, statsFinishedDTO{
			BookID: f.BookID, Title: f.Title, Number: f.Number,
			SeriesID: f.SeriesID, SeriesName: f.SeriesName, FinishedAt: f.FinishedAt,
		})
	}
	return dto
}

func nameCounts(in []domain.NameCount) []statsNameCountDTO {
	out := make([]statsNameCountDTO, 0, len(in))
	for _, nc := range in {
		out = append(out, statsNameCountDTO{Name: nc.Name, Count: nc.Count})
	}
	return out
}

// handleMyStats serves the acting user's reading dashboard (Milestone G).
func handleMyStats(svc *stats.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		summary, err := svc.Summary(r.Context(), currentUserID(r))
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toStatsDTO(summary))
	}
}
