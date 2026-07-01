package http

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/service/browse"
	"github.com/siposbnc/comic-hub/server/internal/service/reading"
)

type progressDTO struct {
	BookID    string  `json:"bookId"`
	Page      int     `json:"page"`
	PageCount int     `json:"pageCount"`
	Status    string  `json:"status"`
	Percent   float64 `json:"percent"`
	UpdatedAt int64   `json:"updatedAt"`
}

func toProgressDTO(p domain.Progress) progressDTO {
	percent := 0.0
	if p.Status == domain.StatusRead {
		percent = 100 // a finished book is 100%, not (pageCount-1)/pageCount
	} else if p.PageCount > 0 {
		percent = math.Round(float64(p.Page)/float64(p.PageCount)*1000) / 10
	}
	return progressDTO{
		BookID: p.BookID, Page: p.Page, PageCount: p.PageCount,
		Status: string(p.Status), Percent: percent, UpdatedAt: p.UpdatedAt,
	}
}

// handleContinueReading serves the Continue Reading rail.
func handleContinueReading(b *browse.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cards, err := b.ContinueReading(r.Context(), currentUserID(r))
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": cards})
	}
}

func handleGetProgress(rd *reading.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, err := rd.Get(r.Context(), currentUserID(r), chi.URLParam(r, "bookId"))
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toProgressDTO(p))
	}
}

// progressWriteReq is a single progress write: PUT body, or one batch item (which also
// names its book). updatedAt is optional — readers replaying offline progress stamp
// when the reading happened; last-writer-wins by updatedAt (ADR-008). A batch item may
// carry contentHash instead of bookId: standalone-mode progress is keyed by the file's
// content hash (the reader has no book id), and the server resolves it here.
type progressWriteReq struct {
	BookID      string `json:"bookId,omitempty"`
	ContentHash string `json:"contentHash,omitempty"`
	Page        int    `json:"page"`
	Status      string `json:"status"`
	Device      string `json:"device"`
	UpdatedAt   int64  `json:"updatedAt"`
}

func handlePutProgress(rd *reading.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req progressWriteReq
		if !decodeJSON(w, r, &req) {
			return
		}
		p, err := rd.Upsert(r.Context(), currentUserID(r), chi.URLParam(r, "bookId"), reading.UpsertInput{
			Page: req.Page, Status: req.Status, Device: req.Device, UpdatedAt: req.UpdatedAt,
		})
		// A stale write (older than the stored row) is not an error to a debounced
		// client: respond 200 with the authoritative row so the device can adopt it
		// (or offer "resume here / there"). Detectable via the returned updatedAt.
		if err != nil && !errors.Is(err, reading.ErrStaleWrite) {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toProgressDTO(p))
	}
}

// handleBatchProgress bulk-upserts progress — the reader flushes offline/standalone
// progress here (docs/03-api.md §6). Items are applied independently; each result
// reports whether the write won (applied) and the authoritative row. A contentHash
// item applies to every catalog book carrying that hash (duplicate copies share your
// place); its result reports the first.
func handleBatchProgress(rd *reading.Service, repo domain.Repository) http.HandlerFunc {
	const maxBatch = 500
	type itemResult struct {
		BookID   string       `json:"bookId,omitempty"`
		Applied  bool         `json:"applied"`
		Progress *progressDTO `json:"progress,omitempty"`
		Error    string       `json:"error,omitempty"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Items []progressWriteReq `json:"items"`
		}
		if !decodeJSON(w, r, &req) {
			return
		}
		if len(req.Items) == 0 {
			writeError(w, http.StatusBadRequest, "empty_batch", "items is empty")
			return
		}
		if len(req.Items) > maxBatch {
			writeError(w, http.StatusBadRequest, "batch_too_large", "at most 500 items per batch")
			return
		}
		userID := currentUserID(r)
		apply := func(res *itemResult, bookID string, it progressWriteReq) {
			p, err := rd.Upsert(r.Context(), userID, bookID, reading.UpsertInput{
				Page: it.Page, Status: it.Status, Device: it.Device, UpdatedAt: it.UpdatedAt,
			})
			switch {
			case err == nil:
				res.Applied = true
				dto := toProgressDTO(p)
				res.Progress = &dto
			case errors.Is(err, reading.ErrStaleWrite):
				dto := toProgressDTO(p) // the authoritative (newer) row
				res.Progress = &dto
			default:
				res.Error = err.Error()
			}
		}
		results := make([]itemResult, 0, len(req.Items))
		for _, it := range req.Items {
			res := itemResult{BookID: it.BookID}
			switch {
			case it.BookID != "":
				apply(&res, it.BookID, it)
			case it.ContentHash != "":
				books := booksByHash(r.Context(), repo, it.ContentHash)
				if len(books) == 0 {
					res.Error = "no book with that content hash"
					break
				}
				res.BookID = books[0].ID
				apply(&res, books[0].ID, it)
				for _, b := range books[1:] {
					dup := itemResult{} // duplicate copies share the place; result unused
					apply(&dup, b.ID, it)
				}
			default:
				res.Error = "bookId or contentHash is required"
			}
			results = append(results, res)
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": results})
	}
}

// booksByHash resolves a content hash to catalog books across every library (the repo
// keys hash lookups per library — they power the scanner's dedup path).
func booksByHash(ctx context.Context, repo domain.Repository, hash string) []domain.Book {
	libs, err := repo.Libraries().List(ctx)
	if err != nil {
		return nil
	}
	var out []domain.Book
	for _, l := range libs {
		if books, err := repo.Books().ByContentHash(ctx, l.ID, hash); err == nil {
			out = append(out, books...)
		}
	}
	return out
}

func handleMarkBook(rd *reading.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Status string `json:"status"`
		}
		if !decodeJSON(w, r, &req) {
			return
		}
		p, err := rd.Mark(r.Context(), currentUserID(r), chi.URLParam(r, "id"), req.Status)
		if err != nil {
			if err.Error() == "status must be \"read\" or \"unread\"" {
				writeError(w, http.StatusBadRequest, "invalid_status", err.Error())
				return
			}
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toProgressDTO(p))
	}
}

// handleGetReaderPrefs returns the user's per-book reader overrides (opaque JSON).
func handleGetReaderPrefs(rd *reading.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		settings, err := rd.GetReaderPrefs(r.Context(), currentUserID(r), chi.URLParam(r, "id"))
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"settings": settings})
	}
}

// handlePutReaderPrefs stores the user's per-book reader overrides.
func handlePutReaderPrefs(rd *reading.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Settings json.RawMessage `json:"settings"`
		}
		if !decodeJSON(w, r, &req) {
			return
		}
		if err := rd.SetReaderPrefs(r.Context(), currentUserID(r), chi.URLParam(r, "id"), req.Settings); err != nil {
			writeDomainError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
