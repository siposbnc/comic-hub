package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/service/browse"
	"github.com/siposbnc/comic-hub/server/internal/service/organize"
)

// collectionDTO is the wire shape for a collection (docs/03-api.md §7).
type collectionDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	CoverBookID string `json:"coverBookId,omitempty"`
	BookCount   int    `json:"bookCount"`
	CreatedAt   int64  `json:"createdAt"`
	UpdatedAt   int64  `json:"updatedAt"`
}

func toCollectionDTO(c domain.Collection) collectionDTO {
	return collectionDTO{
		ID:          c.ID,
		Name:        c.Name,
		Description: c.Description,
		CoverBookID: c.CoverBookID,
		BookCount:   c.BookCount,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}

type createCollectionRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type updateCollectionRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	CoverBookID *string `json:"coverBookId"`
}

type addItemsRequest struct {
	BookIDs []string `json:"bookIds"`
}

type reorderRequest struct {
	BookID   string `json:"bookId"`
	BeforeID string `json:"beforeId"`
}

func handleListCollections(svc *organize.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cols, err := svc.ListCollections(r.Context())
		if err != nil {
			writeDomainError(w, err)
			return
		}
		items := make([]collectionDTO, 0, len(cols))
		for _, c := range cols {
			items = append(items, toCollectionDTO(c))
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

func handleCreateCollection(svc *organize.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createCollectionRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		c, err := svc.CreateCollection(r.Context(), currentUserID(r), organize.CollectionInput{
			Name:        req.Name,
			Description: req.Description,
		})
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, toCollectionDTO(c))
	}
}

// handleGetCollection returns the collection header plus its books in display order.
func handleGetCollection(svc *organize.Service, b *browse.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		c, err := svc.GetCollection(r.Context(), id)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		ids, err := svc.CollectionItems(r.Context(), id)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		books, err := b.BooksByIDs(r.Context(), ids, currentUserID(r))
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"collection": toCollectionDTO(c),
			"books":      books,
		})
	}
}

func handleUpdateCollection(svc *organize.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req updateCollectionRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		c, err := svc.UpdateCollection(r.Context(), chi.URLParam(r, "id"), organize.CollectionPatch{
			Name:        req.Name,
			Description: req.Description,
			CoverBookID: req.CoverBookID,
		})
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toCollectionDTO(c))
	}
}

func handleDeleteCollection(svc *organize.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := svc.DeleteCollection(r.Context(), chi.URLParam(r, "id")); err != nil {
			writeDomainError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleAddCollectionItems(svc *organize.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req addItemsRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		if err := svc.AddItems(r.Context(), chi.URLParam(r, "id"), req.BookIDs); err != nil {
			writeDomainError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleReorderCollectionItem(svc *organize.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req reorderRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		if err := svc.Reorder(r.Context(), chi.URLParam(r, "id"), req.BookID, req.BeforeID); err != nil {
			writeDomainError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleRemoveCollectionItem(svc *organize.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := svc.RemoveItem(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "bookId"))
		if err != nil {
			writeDomainError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
