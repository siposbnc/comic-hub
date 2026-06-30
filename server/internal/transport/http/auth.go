package http

import (
	"context"
	"crypto/subtle"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/siposbnc/comic-hub/server/internal/access"
	"github.com/siposbnc/comic-hub/server/internal/config"
	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/service/auth"
)

type ctxKey int

const userCtxKey ctxKey = iota

// withUser stores the acting user on the request context.
func withUser(ctx context.Context, u domain.User) context.Context {
	return context.WithValue(ctx, userCtxKey, u)
}

// withActor stores the acting user and their content ceiling, so services can apply
// restrictions (browse filtering, reader access) straight from the context.
func withActor(ctx context.Context, u domain.User) context.Context {
	return access.WithCeiling(withUser(ctx, u), u.AgeRatingMax)
}

// userFromContext returns the acting user set by the auth middleware.
func userFromContext(ctx context.Context) (domain.User, bool) {
	u, ok := ctx.Value(userCtxKey).(domain.User)
	return u, ok
}

// publicAuthPaths bypass access-token authentication: login/refresh precede having a token,
// and logout authenticates via the refresh token in its body (so an expired access token
// doesn't block signing out).
var publicAuthPaths = map[string]bool{
	"/api/v1/auth/login":   true,
	"/api/v1/auth/refresh": true,
	"/api/v1/auth/logout":  true,
}

// authMiddleware resolves the acting user for each request.
//
//   - Auth disabled (embedded mode, or server mode pre-bootstrap/dev): preserves the loopback
//     bearer-token check and acts as the implicit owner — unchanged behavior.
//   - Auth enabled (server mode): requires a valid access token (Authorization: Bearer, or a
//     ?token= query param for <img> tags) and acts as that user; login/refresh are public.
func authMiddleware(cfg config.Config, authSvc *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !cfg.AuthEnabled {
				if cfg.Token != "" && !validLoopbackToken(cfg.Token, r) {
					writeError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
					return
				}
				next.ServeHTTP(w, r.WithContext(withActor(r.Context(), implicitOwner())))
				return
			}

			if publicAuthPaths[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}
			tok := bearerToken(r)
			if tok == "" {
				tok = r.URL.Query().Get("token")
			}
			u, err := authSvc.Authenticate(r.Context(), tok)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
				return
			}
			next.ServeHTTP(w, r.WithContext(withActor(r.Context(), u)))
		})
	}
}

// requireRole gates a route to users at or above min. The acting user is set by
// authMiddleware; with auth disabled it is the implicit owner, so embedded/dev runs pass.
func requireRole(min domain.UserRole) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, ok := userFromContext(r.Context())
			if !ok || !u.Role.AtLeast(min) {
				writeError(w, http.StatusForbidden, "forbidden", "insufficient permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// requireBookAccess refuses (403) a restricted user access to a book rated above their
// ceiling — the security boundary for content restrictions, applied to the reader's content
// routes (manifest / cover / pages / prefetch). A no-op for unrestricted users.
func requireBookAccess(repo domain.Repository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ceiling := access.CeilingFrom(r.Context()); ceiling != "" {
				if b, err := repo.Books().Get(r.Context(), chi.URLParam(r, "id")); err == nil &&
					!access.Allowed(ceiling, b.AgeRating) {
					writeError(w, http.StatusForbidden, "forbidden", "this content is restricted")
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func validLoopbackToken(token string, r *http.Request) bool {
	provided := bearerToken(r)
	if provided == "" {
		provided = r.URL.Query().Get("token")
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(token)) == 1
}

// implicitOwner is the single owner identity used when auth is disabled.
func implicitOwner() domain.User {
	return domain.User{ID: domain.OwnerUserID, Username: "owner", DisplayName: "Owner", Role: domain.RoleOwner}
}

type userDTO struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	DisplayName  string `json:"displayName"`
	Role         string `json:"role"`
	AgeRatingMax string `json:"ageRatingMax,omitempty"`
}

func toUserDTO(u domain.User) userDTO {
	return userDTO{
		ID:           u.ID,
		Username:     u.Username,
		DisplayName:  u.DisplayName,
		Role:         string(u.Role),
		AgeRatingMax: u.AgeRatingMax,
	}
}

type tokensDTO struct {
	Access       string  `json:"access"`
	Refresh      string  `json:"refresh"`
	AccessExpiry int64   `json:"accessExpiry"`
	User         userDTO `json:"user"`
}

func toTokensDTO(t auth.Tokens, u domain.User) tokensDTO {
	return tokensDTO{Access: t.Access, Refresh: t.Refresh, AccessExpiry: t.AccessExpiry, User: toUserDTO(u)}
}

// handleLogin authenticates a username/password and returns a token pair.
func handleLogin(authSvc *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if !decodeJSON(w, r, &req) {
			return
		}
		toks, u, err := authSvc.Login(r.Context(), req.Username, req.Password)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toTokensDTO(toks, u))
	}
}

// handleRefresh rotates a refresh token for a fresh pair.
func handleRefresh(authSvc *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Refresh string `json:"refresh"`
		}
		if !decodeJSON(w, r, &req) {
			return
		}
		toks, u, err := authSvc.Refresh(r.Context(), req.Refresh)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toTokensDTO(toks, u))
	}
}

// handleLogout revokes a refresh token's session.
func handleLogout(authSvc *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Refresh string `json:"refresh"`
		}
		if r.ContentLength > 0 && !decodeJSON(w, r, &req) {
			return
		}
		if err := authSvc.Logout(r.Context(), req.Refresh); err != nil {
			writeDomainError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
