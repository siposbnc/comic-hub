package http

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/service/browse"
	"github.com/siposbnc/comic-hub/server/internal/service/organize"
)

// smartRuleDTO mirrors domain.SmartRule on the wire.
type smartRuleDTO struct {
	Field string `json:"field"`
	Op    string `json:"op"`
	Value string `json:"value"`
}

type smartRulesDTO struct {
	Match string         `json:"match"`
	Rules []smartRuleDTO `json:"rules"`
}

// smartListDTO is the wire shape for a smart list.
type smartListDTO struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Rules     smartRulesDTO `json:"rules"`
	BookCount int           `json:"bookCount"`
	CreatedAt int64         `json:"createdAt"`
	UpdatedAt int64         `json:"updatedAt"`
}

func toSmartRulesDTO(r domain.SmartRules) smartRulesDTO {
	out := smartRulesDTO{Match: r.Match, Rules: make([]smartRuleDTO, 0, len(r.Rules))}
	for _, rule := range r.Rules {
		out.Rules = append(out.Rules, smartRuleDTO{Field: rule.Field, Op: rule.Op, Value: rule.Value})
	}
	return out
}

func fromSmartRulesDTO(d smartRulesDTO) domain.SmartRules {
	out := domain.SmartRules{Match: d.Match, Rules: make([]domain.SmartRule, 0, len(d.Rules))}
	for _, rule := range d.Rules {
		out.Rules = append(out.Rules, domain.SmartRule{Field: rule.Field, Op: rule.Op, Value: rule.Value})
	}
	return out
}

func toSmartListDTO(l domain.SmartList) smartListDTO {
	return smartListDTO{
		ID:        l.ID,
		Name:      l.Name,
		Rules:     toSmartRulesDTO(l.Rules),
		BookCount: l.BookCount,
		CreatedAt: l.CreatedAt,
		UpdatedAt: l.UpdatedAt,
	}
}

type createSmartListRequest struct {
	Name  string        `json:"name"`
	Rules smartRulesDTO `json:"rules"`
}

type updateSmartListRequest struct {
	Name  *string        `json:"name"`
	Rules *smartRulesDTO `json:"rules"`
}

func handleListSmartLists(svc *organize.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lists, err := svc.ListSmartLists(r.Context(), currentUserID(r))
		if err != nil {
			writeDomainError(w, err)
			return
		}
		items := make([]smartListDTO, 0, len(lists))
		for _, l := range lists {
			items = append(items, toSmartListDTO(l))
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

func handleCreateSmartList(svc *organize.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createSmartListRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		l, err := svc.CreateSmartList(r.Context(), currentUserID(r), req.Name, fromSmartRulesDTO(req.Rules))
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, toSmartListDTO(l))
	}
}

func handleUpdateSmartList(svc *organize.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req updateSmartListRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		patch := organize.SmartListPatch{Name: req.Name}
		if req.Rules != nil {
			rules := fromSmartRulesDTO(*req.Rules)
			patch.Rules = &rules
		}
		l, err := svc.UpdateSmartList(r.Context(), chi.URLParam(r, "id"), patch)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toSmartListDTO(l))
	}
}

func handleDeleteSmartList(svc *organize.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := svc.DeleteSmartList(r.Context(), chi.URLParam(r, "id")); err != nil {
			writeDomainError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleSmartListResults evaluates a smart list and returns its books as browse cards.
func handleSmartListResults(svc *organize.Service, b *browse.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid := currentUserID(r)
		limit := 0
		if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil {
			limit = v
		}
		list, ids, err := svc.SmartListResults(r.Context(), chi.URLParam(r, "id"), uid, limit)
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
			"smartList": toSmartListDTO(list),
			"books":     books,
		})
	}
}
