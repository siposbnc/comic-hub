package http

import (
	"net/http"
	"strings"
)

// bearerToken extracts the token from an "Authorization: Bearer <token>" header (empty if
// absent). Validation lives in authMiddleware (see auth.go).
func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if strings.HasPrefix(h, prefix) {
		return strings.TrimSpace(h[len(prefix):])
	}
	return ""
}
