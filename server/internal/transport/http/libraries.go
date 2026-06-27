package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/service/library"
)

// libraryDTO is the wire shape for a library (see docs/03-api.md §3). Summary counts
// (series/books) arrive once the scanner populates the catalog.
type libraryDTO struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Kind      string   `json:"kind"`
	Roots     []string `json:"roots"`
	CreatedAt int64    `json:"createdAt"`
	UpdatedAt int64    `json:"updatedAt"`
}

func toLibraryDTO(l domain.Library) libraryDTO {
	roots := l.Roots
	if roots == nil {
		roots = []string{}
	}
	return libraryDTO{
		ID:        l.ID,
		Name:      l.Name,
		Kind:      l.Kind,
		Roots:     roots,
		CreatedAt: l.CreatedAt,
		UpdatedAt: l.UpdatedAt,
	}
}

type createLibraryRequest struct {
	Name  string   `json:"name"`
	Kind  string   `json:"kind"`
	Roots []string `json:"roots"`
}

func handleListLibraries(svc *library.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		libs, err := svc.List(r.Context())
		if err != nil {
			writeDomainError(w, err)
			return
		}
		items := make([]libraryDTO, 0, len(libs))
		for _, l := range libs {
			items = append(items, toLibraryDTO(l))
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

func handleCreateLibrary(svc *library.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createLibraryRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		lib, err := svc.Create(r.Context(), library.CreateInput{
			Name:  req.Name,
			Kind:  req.Kind,
			Roots: req.Roots,
		})
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, toLibraryDTO(lib))
	}
}

func handleGetLibrary(svc *library.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		lib, err := svc.Get(r.Context(), id)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toLibraryDTO(lib))
	}
}

func handleDeleteLibrary(svc *library.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := svc.Delete(r.Context(), id); err != nil {
			writeDomainError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
