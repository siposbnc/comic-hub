package http

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/siposbnc/comic-hub/server/internal/config"
)

// tokenAuth enforces the loopback bearer token in embedded mode. When the configured
// token is empty (auth disabled, e.g. a dev run), all requests pass. Image endpoints
// will additionally accept a `?token=` query param so <img> tags can authenticate.
func tokenAuth(cfg config.Config) func(http.Handler) http.Handler {
	token := cfg.Token
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}
			provided := bearerToken(r)
			if provided == "" {
				provided = r.URL.Query().Get("token")
			}
			if subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
				writeError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if strings.HasPrefix(h, prefix) {
		return strings.TrimSpace(h[len(prefix):])
	}
	return ""
}
