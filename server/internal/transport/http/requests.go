package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/siposbnc/comic-hub/server/internal/domain"
)

// decodeJSON reads and decodes a JSON request body into dst. On failure it writes a
// 400 error response and returns false so the handler can return early.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return false
	}
	return true
}

// writeDomainError maps a domain/service error to the appropriate HTTP response.
func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "resource not found")
	case errors.Is(err, domain.ErrValidation):
		writeError(w, http.StatusBadRequest, "validation_failed", err.Error())
	case errors.Is(err, domain.ErrConflict):
		writeError(w, http.StatusConflict, "conflict", err.Error())
	case errors.Is(err, domain.ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "something went wrong")
	}
}
