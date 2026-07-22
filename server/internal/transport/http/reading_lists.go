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

// readingListEntryDTO is one ordered entry of a reading list. `kind` is "book" for a
// normal entry and "collection" for a collection reference. Stale entries (deleted books,
// or placeholders added manually for issues not in the library) have no `book` and render
// from the snapshot fields; they hold their slot but can't be opened. A collection entry
// carries `collection` — the group's name and its books in order, resolved live.
type readingListEntryDTO struct {
	ID         string                    `json:"id"`
	Kind       string                    `json:"kind"`
	Stale      bool                      `json:"stale"`
	SeriesName string                    `json:"seriesName,omitempty"`
	Number     string                    `json:"number,omitempty"`
	Title      string                    `json:"title,omitempty"`
	AddedAt    int64                     `json:"addedAt"`
	Book       *browse.BookCard          `json:"book,omitempty"`
	Collection *readingListCollectionDTO `json:"collection,omitempty"`
}

// readingListCollectionDTO is a collection reference expanded for display: the group label
// plus its books in collection order (books the viewer can't see are omitted).
type readingListCollectionDTO struct {
	ID    string            `json:"id"`
	Name  string            `json:"name"`
	Books []browse.BookCard `json:"books"`
}

// manualEntryRequest describes a placeholder to add for an issue not in the library.
type manualEntryRequest struct {
	SeriesName string `json:"seriesName"`
	Number     string `json:"number"`
	Title      string `json:"title"`
}

// addReadingListItemsRequest adds real books (bookIds) and/or manual placeholders.
type addReadingListItemsRequest struct {
	BookIDs []string             `json:"bookIds"`
	Manual  []manualEntryRequest `json:"manual"`
}

// addReadingListCollectionsRequest references one or more whole collections into a list.
type addReadingListCollectionsRequest struct {
	CollectionIDs []string `json:"collectionIds"`
}

type relinkItemRequest struct {
	BookID string `json:"bookId"`
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

// handleGetReadingList returns the list header plus its entries in display order —
// linked entries carry a BookCard, stale ones only their snapshot. `books` (linked
// cards only) is kept alongside for older clients.
func handleGetReadingList(svc *organize.Service, b *browse.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid := currentUserID(r)
		id := chi.URLParam(r, "id")
		l, err := svc.GetReadingList(r.Context(), uid, id)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		entries, err := svc.ReadingListEntries(r.Context(), uid, id)
		if err != nil {
			writeDomainError(w, err)
			return
		}

		// Gather every book id we need cards for in one pass: individual linked entries,
		// plus the members of each referenced collection (resolved live, in order). Also
		// remember each collection's name and ordered member ids for the group DTO.
		ids := make([]string, 0, len(entries))
		type colRef struct {
			name    string
			bookIDs []string
		}
		colByEntry := make(map[string]colRef)
		for _, it := range entries {
			switch {
			case it.IsCollection():
				c, err := svc.GetCollection(r.Context(), it.CollectionID)
				if err != nil {
					// A collection deleted out from under a cached list: skip its group.
					continue
				}
				memberIDs, err := svc.CollectionItems(r.Context(), it.CollectionID)
				if err != nil {
					writeDomainError(w, err)
					return
				}
				ids = append(ids, memberIDs...)
				colByEntry[it.ID] = colRef{name: c.Name, bookIDs: memberIDs}
			case !it.Stale():
				ids = append(ids, it.BookID)
			}
		}
		books, err := b.BooksByIDs(r.Context(), ids, uid)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		cardByID := make(map[string]*browse.BookCard, len(books))
		for i := range books {
			cardByID[books[i].ID] = &books[i]
		}

		items := make([]readingListEntryDTO, 0, len(entries))
		for _, it := range entries {
			dto := readingListEntryDTO{
				ID:         it.ID,
				Kind:       "book",
				Stale:      it.Stale(),
				SeriesName: it.SeriesName,
				Number:     it.Number,
				Title:      it.Title,
				AddedAt:    it.AddedAt,
			}
			if it.IsCollection() {
				ref, ok := colByEntry[it.ID]
				if !ok {
					// Collection vanished; drop the entry from the response.
					continue
				}
				group := &readingListCollectionDTO{ID: it.CollectionID, Name: ref.name, Books: []browse.BookCard{}}
				for _, bid := range ref.bookIDs {
					if card, ok := cardByID[bid]; ok {
						group.Books = append(group.Books, *card)
					}
				}
				dto.Kind = "collection"
				dto.Stale = false
				dto.Collection = group
				items = append(items, dto)
				continue
			}
			if card, ok := cardByID[it.BookID]; ok {
				dto.Book = card
			} else if !it.Stale() {
				// Linked but filtered out (e.g. above a restricted user's ceiling): hide
				// the target, show the slot.
				dto.Stale = true
			}
			items = append(items, dto)
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"readingList": toReadingListDTO(l),
			"items":       items,
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
		var req addReadingListItemsRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		uid, id := currentUserID(r), chi.URLParam(r, "id")
		if len(req.BookIDs) > 0 {
			if err := svc.AddReadingListItems(r.Context(), uid, id, req.BookIDs); err != nil {
				writeDomainError(w, err)
				return
			}
		}
		if len(req.Manual) > 0 {
			entries := make([]domain.ManualListItem, len(req.Manual))
			for i, m := range req.Manual {
				entries[i] = domain.ManualListItem{SeriesName: m.SeriesName, Number: m.Number, Title: m.Title}
			}
			if err := svc.AddReadingListManualItems(r.Context(), uid, id, entries); err != nil {
				writeDomainError(w, err)
				return
			}
		}
		if len(req.BookIDs) == 0 && len(req.Manual) == 0 {
			writeError(w, http.StatusBadRequest, "validation", "bookIds or manual entries required")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleAddReadingListCollections references one or more whole collections into a list,
// each as a single ordered group. Adding a collection already in the list is a no-op.
func handleAddReadingListCollections(svc *organize.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req addReadingListCollectionsRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		if len(req.CollectionIDs) == 0 {
			writeError(w, http.StatusBadRequest, "validation", "collectionIds is required")
			return
		}
		uid, id := currentUserID(r), chi.URLParam(r, "id")
		for _, cid := range req.CollectionIDs {
			if err := svc.AddReadingListCollection(r.Context(), uid, id, cid); err != nil {
				writeDomainError(w, err)
				return
			}
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleRelinkReadingListItem points an entry (usually stale) at a real book.
func handleRelinkReadingListItem(svc *organize.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req relinkItemRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		err := svc.RelinkReadingListItem(r.Context(), currentUserID(r),
			chi.URLParam(r, "id"), chi.URLParam(r, "itemId"), req.BookID)
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
