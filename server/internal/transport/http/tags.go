package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/service/browse"
	"github.com/siposbnc/comic-hub/server/internal/service/organize"
)

// tagDTO is the wire shape for a tag (with its book count).
type tagDTO struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Color     string `json:"color,omitempty"`
	BookCount int    `json:"bookCount"`
}

func toTagDTO(t domain.Tag) tagDTO {
	return tagDTO{ID: t.ID, Name: t.Name, Color: t.Color, BookCount: t.BookCount}
}

type createTagRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type updateTagRequest struct {
	Name  *string `json:"name"`
	Color *string `json:"color"`
}

type assignTagsRequest struct {
	TagIDs []string `json:"tagIds"`
}

func handleListTags(svc *organize.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tags, err := svc.ListTags(r.Context())
		if err != nil {
			writeDomainError(w, err)
			return
		}
		items := make([]tagDTO, 0, len(tags))
		for _, t := range tags {
			items = append(items, toTagDTO(t))
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

func handleCreateTag(svc *organize.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createTagRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		t, err := svc.CreateTag(r.Context(), req.Name, req.Color)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, toTagDTO(t))
	}
}

func handleUpdateTag(svc *organize.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req updateTagRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		t, err := svc.UpdateTag(r.Context(), chi.URLParam(r, "id"), organize.TagPatch{
			Name:  req.Name,
			Color: req.Color,
		})
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toTagDTO(t))
	}
}

func handleDeleteTag(svc *organize.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := svc.DeleteTag(r.Context(), chi.URLParam(r, "id")); err != nil {
			writeDomainError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleTagBooks lists the books carrying a tag (newest-added first), as browse cards.
func handleTagBooks(svc *organize.Service, b *browse.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ids, err := svc.TaggedBookIDs(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			writeDomainError(w, err)
			return
		}
		books, err := b.BooksByIDs(r.Context(), ids, currentUserID(r))
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": books})
	}
}

func handleAssignTags(svc *organize.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req assignTagsRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		if err := svc.AssignTags(r.Context(), chi.URLParam(r, "id"), req.TagIDs); err != nil {
			writeDomainError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleUnassignTag(svc *organize.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := svc.UnassignTag(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "tagId"))
		if err != nil {
			writeDomainError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
