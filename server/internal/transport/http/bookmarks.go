package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/service/reading"
)

type bookmarkDTO struct {
	ID        string `json:"id"`
	BookID    string `json:"bookId"`
	Page      int    `json:"page"`
	Note      string `json:"note"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}

func toBookmarkDTO(b domain.Bookmark) bookmarkDTO {
	return bookmarkDTO{
		ID: b.ID, BookID: b.BookID, Page: b.Page, Note: b.Note,
		CreatedAt: b.CreatedAt, UpdatedAt: b.UpdatedAt,
	}
}

func toBookmarkDTOs(bs []domain.Bookmark) []bookmarkDTO {
	out := make([]bookmarkDTO, len(bs))
	for i, b := range bs {
		out[i] = toBookmarkDTO(b)
	}
	return out
}

// handleListBookmarks returns a book's bookmarks for the user, ordered by page.
func handleListBookmarks(rd *reading.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bs, err := rd.ListBookmarks(r.Context(), currentUserID(r), chi.URLParam(r, "id"))
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": toBookmarkDTOs(bs)})
	}
}

// handleAddBookmark bookmarks a page (idempotent: re-adding a page updates its note).
func handleAddBookmark(rd *reading.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Page int    `json:"page"`
			Note string `json:"note"`
		}
		if !decodeJSON(w, r, &req) {
			return
		}
		bm, err := rd.AddBookmark(r.Context(), currentUserID(r), chi.URLParam(r, "id"), req.Page, req.Note)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, toBookmarkDTO(bm))
	}
}

// handleUpdateBookmark replaces a bookmark's note.
func handleUpdateBookmark(rd *reading.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Note string `json:"note"`
		}
		if !decodeJSON(w, r, &req) {
			return
		}
		bm, err := rd.UpdateBookmarkNote(r.Context(), currentUserID(r), chi.URLParam(r, "bookmarkId"), req.Note)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toBookmarkDTO(bm))
	}
}

// handleDeleteBookmark removes a bookmark.
func handleDeleteBookmark(rd *reading.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := rd.DeleteBookmark(r.Context(), currentUserID(r), chi.URLParam(r, "bookmarkId")); err != nil {
			writeDomainError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
