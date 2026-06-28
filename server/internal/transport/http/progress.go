package http

import (
	"encoding/json"
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
	if p.PageCount > 0 {
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

func handlePutProgress(rd *reading.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Page   int    `json:"page"`
			Status string `json:"status"`
			Device string `json:"device"`
		}
		if !decodeJSON(w, r, &req) {
			return
		}
		p, err := rd.Upsert(r.Context(), currentUserID(r), chi.URLParam(r, "bookId"), reading.UpsertInput{
			Page: req.Page, Status: req.Status, Device: req.Device,
		})
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toProgressDTO(p))
	}
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
