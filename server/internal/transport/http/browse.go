package http

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/service/browse"
)

// currentUserID returns the acting user. In embedded mode that is always the implicit
// owner; auth mode will resolve it from the validated token/session.
func currentUserID(_ *http.Request) string { return domain.OwnerUserID }

func handleListSeries(b *browse.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lib := r.URL.Query().Get("library")
		if lib == "" {
			writeError(w, http.StatusBadRequest, "missing_library", "library query param is required")
			return
		}
		cards, err := b.ListSeries(r.Context(), lib, currentUserID(r))
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": cards})
	}
}

func handleSeriesDetail(b *browse.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		detail, err := b.SeriesDetail(r.Context(), chi.URLParam(r, "id"), currentUserID(r))
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, detail)
	}
}

func handleListBooks(b *browse.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lib := r.URL.Query().Get("library") // optional; empty spans all libraries
		limit := 0
		if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil {
			limit = v
		}
		cards, err := b.RecentBooks(r.Context(), lib, currentUserID(r), limit)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": cards})
	}
}

func handleBookDetail(b *browse.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		detail, err := b.BookDetail(r.Context(), chi.URLParam(r, "id"), currentUserID(r))
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, detail)
	}
}

func handleDiscover(b *browse.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		feed, err := b.Discover(r.Context(), r.URL.Query().Get("library"), currentUserID(r))
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, feed)
	}
}

// handleSearch runs a full-text catalog search (docs/03-api.md §8). `q` is the raw query,
// `type` filters to series/book (default all), `library` optionally scopes it.
func handleSearch(b *browse.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		limit := 0
		if v, err := strconv.Atoi(q.Get("limit")); err == nil {
			limit = v
		}
		results, err := b.Search(r.Context(), q.Get("library"), q.Get("q"), q.Get("type"), limit)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, results)
	}
}
