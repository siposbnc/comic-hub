package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/service/browse"
	"github.com/siposbnc/comic-hub/server/internal/service/organize"
)

// readingListDTO is the wire shape for a per-user reading list (docs/03-api.md §7).
type readingListDTO struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Active    bool   `json:"active"`
	BookCount int    `json:"bookCount"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}

func toReadingListDTO(l domain.ReadingList) readingListDTO {
	return readingListDTO{
		ID:        l.ID,
		Name:      l.Name,
		Active:    l.Active,
		BookCount: l.BookCount,
		CreatedAt: l.CreatedAt,
		UpdatedAt: l.UpdatedAt,
	}
}

type createReadingListRequest struct {
	Name string `json:"name"`
}

type renameReadingListRequest struct {
	Name string `json:"name"`
}

func handleListReadingLists(svc *organize.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lists, err := svc.ListReadingLists(r.Context(), currentUserID(r))
		if err != nil {
			writeDomainError(w, err)
			return
		}
		items := make([]readingListDTO, 0, len(lists))
		for _, l := range lists {
			items = append(items, toReadingListDTO(l))
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

func handleCreateReadingList(svc *organize.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createReadingListRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		l, err := svc.CreateReadingList(r.Context(), currentUserID(r), req.Name)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, toReadingListDTO(l))
	}
}

// handleGetReadingList returns the list header plus its books in display order.
func handleGetReadingList(svc *organize.Service, b *browse.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid := currentUserID(r)
		id := chi.URLParam(r, "id")
		l, err := svc.GetReadingList(r.Context(), uid, id)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		ids, err := svc.ReadingListItems(r.Context(), uid, id)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		books, err := b.BooksByIDs(r.Context(), ids, uid)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"readingList": toReadingListDTO(l),
			"books":       books,
		})
	}
}

func handleUpdateReadingList(svc *organize.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req renameReadingListRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		l, err := svc.RenameReadingList(r.Context(), currentUserID(r), chi.URLParam(r, "id"), req.Name)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toReadingListDTO(l))
	}
}

func handleSetActiveReadingList(svc *organize.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := svc.SetActiveReadingList(r.Context(), currentUserID(r), chi.URLParam(r, "id"))
		if err != nil {
			writeDomainError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleDeleteReadingList(svc *organize.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := svc.DeleteReadingList(r.Context(), currentUserID(r), chi.URLParam(r, "id"))
		if err != nil {
			writeDomainError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleAddReadingListItems(svc *organize.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req addItemsRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		err := svc.AddReadingListItems(r.Context(), currentUserID(r), chi.URLParam(r, "id"), req.BookIDs)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleReorderReadingListItem(svc *organize.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req reorderRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		err := svc.ReorderReadingList(r.Context(), currentUserID(r), chi.URLParam(r, "id"), req.BookID, req.BeforeID)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleRemoveReadingListItem(svc *organize.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := svc.RemoveReadingListItem(r.Context(), currentUserID(r), chi.URLParam(r, "id"), chi.URLParam(r, "bookId"))
		if err != nil {
			writeDomainError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
