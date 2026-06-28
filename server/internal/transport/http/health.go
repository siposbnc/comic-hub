package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/siposbnc/comic-hub/server/internal/service/health"
)

// handleLibraryHealth returns a library's maintenance report (corrupt / orphaned /
// unmatched / duplicate books). See docs/03-api.md §7.
func handleLibraryHealth(svc *health.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		report, err := svc.Report(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, report)
	}
}
