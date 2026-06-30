package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/service/auth"
)

// User management (admin only — gated in the router). Accounts are created with a password;
// roles/restrictions are edited here. The implicit owner can't be deleted.

func handleListUsers(authSvc *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, err := authSvc.ListUsers(r.Context())
		if err != nil {
			writeDomainError(w, err)
			return
		}
		out := make([]userDTO, 0, len(users))
		for _, u := range users {
			out = append(out, toUserDTO(u))
		}
		writeJSON(w, http.StatusOK, map[string]any{"users": out})
	}
}

func handleCreateUser(authSvc *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Username     string `json:"username"`
			DisplayName  string `json:"displayName"`
			Role         string `json:"role"`
			Password     string `json:"password"`
			AgeRatingMax string `json:"ageRatingMax"`
		}
		if !decodeJSON(w, r, &req) {
			return
		}
		u, err := authSvc.CreateUser(r.Context(), auth.CreateUserInput{
			Username:     req.Username,
			DisplayName:  req.DisplayName,
			Role:         domain.UserRole(req.Role),
			Password:     req.Password,
			AgeRatingMax: req.AgeRatingMax,
		})
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, toUserDTO(u))
	}
}

func handlePatchUser(authSvc *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		var req struct {
			DisplayName  *string `json:"displayName"`
			Role         *string `json:"role"`
			AgeRatingMax *string `json:"ageRatingMax"`
			Password     *string `json:"password"`
		}
		if !decodeJSON(w, r, &req) {
			return
		}
		// Password changes go through the dedicated path (it revokes sessions).
		if req.Password != nil {
			if err := authSvc.SetUserPassword(r.Context(), id, *req.Password); err != nil {
				writeDomainError(w, err)
				return
			}
		}
		// Profile/role/restriction update, when any of those fields is present.
		if req.DisplayName != nil || req.Role != nil || req.AgeRatingMax != nil {
			cur, err := authSvc.GetUser(r.Context(), id)
			if err != nil {
				writeDomainError(w, err)
				return
			}
			display := cur.DisplayName
			if req.DisplayName != nil {
				display = *req.DisplayName
			}
			role := cur.Role
			if req.Role != nil {
				role = domain.UserRole(*req.Role)
			}
			ageMax := cur.AgeRatingMax
			if req.AgeRatingMax != nil {
				ageMax = *req.AgeRatingMax
			}
			if _, err := authSvc.UpdateUser(r.Context(), id, display, role, ageMax); err != nil {
				writeDomainError(w, err)
				return
			}
		}
		u, err := authSvc.GetUser(r.Context(), id)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toUserDTO(u))
	}
}

func handleDeleteUser(authSvc *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := authSvc.DeleteUser(r.Context(), chi.URLParam(r, "id")); err != nil {
			writeDomainError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
