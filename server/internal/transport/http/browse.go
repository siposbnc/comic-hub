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

// handleStoryArcDetail returns a story arc's header + its issues in reading order.
func handleStoryArcDetail(b *browse.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		detail, err := b.StoryArcDetail(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "arcId"), currentUserID(r))
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, detail)
	}
}

// handleVolumeDetail returns a derived volume's header + its issues.
func handleVolumeDetail(b *browse.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		volume, err := strconv.Atoi(chi.URLParam(r, "volume"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "volume must be an integer")
			return
		}
		detail, err := b.VolumeDetail(r.Context(), chi.URLParam(r, "id"), volume, currentUserID(r))
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

// handleNextBook returns the issue to read after {id}: by series order, or — with
// ?context=readingList — the next item in the user's active reading list. `book` is null
// when there is no next issue.
func handleNextBook(b *browse.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		card, err := b.NextAfter(r.Context(), currentUserID(r), chi.URLParam(r, "id"),
			r.URL.Query().Get("context"))
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"book": card})
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
