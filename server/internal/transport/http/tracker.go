package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/service/organize"
)

// trackerIssueDTO is one cell of the tracker matrix. A library issue carries `bookId` (its
// read state is toggled via /me/books/{id}/mark); an overlay issue carries `id` (toggled via
// /me/tracker/issues/{id}/mark) and no book.
type trackerIssueDTO struct {
	ID     string  `json:"id,omitempty"`
	Number string  `json:"number"`
	Sort   float64 `json:"sort"`
	BookID string  `json:"bookId,omitempty"`
	State  string  `json:"state"`
	Page   int     `json:"page,omitempty"`
	Pages  int     `json:"pages,omitempty"`
	Source string  `json:"source"`
}

// trackerTrackDTO is one row: a library series or a standalone track.
type trackerTrackDTO struct {
	ID        string            `json:"id"`
	SeriesID  string            `json:"seriesId,omitempty"`
	LibraryID string            `json:"libraryId,omitempty"`
	Name      string            `json:"name"`
	Link      string            `json:"link"`
	Issues    []trackerIssueDTO `json:"issues"`
}

// trackDTO is the wire shape of a standalone track (create/rename responses).
type trackDTO struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}

func toTrackDTO(t domain.Track) trackDTO {
	return trackDTO{ID: t.ID, Name: t.Name, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt}
}

type createTrackRequest struct {
	Name string `json:"name"`
}

type addTrackIssuesRequest struct {
	TrackID  string   `json:"trackId"`
	SeriesID string   `json:"seriesId"`
	Numbers  []string `json:"numbers"`
}

type markTrackIssueRequest struct {
	Read bool `json:"read"`
}

func handleGetTracker(svc *organize.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tracks, err := svc.Tracker(r.Context(), currentUserID(r))
		if err != nil {
			writeDomainError(w, err)
			return
		}
		out := make([]trackerTrackDTO, 0, len(tracks))
		for _, t := range tracks {
			issues := make([]trackerIssueDTO, 0, len(t.Issues))
			for _, it := range t.Issues {
				issues = append(issues, trackerIssueDTO{
					ID:     it.ID,
					Number: it.Number,
					Sort:   it.Sort,
					BookID: it.BookID,
					State:  it.State,
					Page:   it.Page,
					Pages:  it.Pages,
					Source: it.Source,
				})
			}
			out = append(out, trackerTrackDTO{
				ID:        t.ID,
				SeriesID:  t.SeriesID,
				LibraryID: t.LibraryID,
				Name:      t.Name,
				Link:      t.Link,
				Issues:    issues,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"tracks": out})
	}
}

func handleCreateTrack(svc *organize.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createTrackRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		t, err := svc.CreateTrack(r.Context(), currentUserID(r), req.Name)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, toTrackDTO(t))
	}
}

func handleRenameTrack(svc *organize.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createTrackRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		t, err := svc.RenameTrack(r.Context(), currentUserID(r), chi.URLParam(r, "id"), req.Name)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toTrackDTO(t))
	}
}

func handleDeleteTrack(svc *organize.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := svc.DeleteTrack(r.Context(), currentUserID(r), chi.URLParam(r, "id")); err != nil {
			writeDomainError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleAddTrackIssues(svc *organize.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req addTrackIssuesRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		err := svc.AddTrackIssues(r.Context(), currentUserID(r), req.TrackID, req.SeriesID, req.Numbers)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleMarkTrackIssue(svc *organize.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req markTrackIssueRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		err := svc.MarkTrackIssue(r.Context(), currentUserID(r), chi.URLParam(r, "id"), req.Read)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleRemoveTrackIssue(svc *organize.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := svc.RemoveTrackIssue(r.Context(), currentUserID(r), chi.URLParam(r, "id")); err != nil {
			writeDomainError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
