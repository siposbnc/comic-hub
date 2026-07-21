package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/jobs"
	"github.com/siposbnc/comic-hub/server/internal/service/metadata"
)

// MatchJobPayload is the JSON body of a metadata_match job: link `volumeProviderId` from
// `provider` to the series and apply each issue to the matching book. Shared by the
// enqueue handler and the job runner in main.
type MatchJobPayload struct {
	SeriesID         string   `json:"seriesId"`
	Provider         string   `json:"provider"`
	VolumeProviderID string   `json:"volumeProviderId"`
	Fields           []string `json:"fields,omitempty"`
}

// AutoMatchJobPayload is the JSON body of a metadata_automatch job: auto-match every
// not-yet-matched series in a library after a scan. Shared with the job runner in main.
type AutoMatchJobPayload struct {
	LibraryID string `json:"libraryId"`
}

// applyRequest is the body of the manual book/series apply endpoints.
type applyRequest struct {
	Provider   string   `json:"provider"`
	ProviderID string   `json:"providerId"`
	Fields     []string `json:"fields,omitempty"`
}

// handleSeriesCandidates returns ranked provider candidates for a series (docs/03-api.md §9).
func handleSeriesCandidates(svc *metadata.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cands, err := svc.Candidates(r.Context(), chi.URLParam(r, "id"),
			r.URL.Query().Get("provider"), r.URL.Query().Get("query"))
		if err != nil {
			writeMatchError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"candidates": cands})
	}
}

// handleSeriesMatch enqueues a metadata_match job that batch-applies a chosen provider
// volume to the series' books, returning the job id to track over the WS jobs topic.
func handleSeriesMatch(svc *metadata.Service, runner *jobs.Runner) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body applyRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
			return
		}
		if body.ProviderID == "" {
			writeError(w, http.StatusBadRequest, "bad_request", "providerId is required")
			return
		}
		payload, _ := json.Marshal(MatchJobPayload{
			SeriesID:         chi.URLParam(r, "id"),
			Provider:         body.Provider,
			VolumeProviderID: body.ProviderID,
			Fields:           body.Fields,
		})
		jobID, err := runner.Submit(r.Context(), domain.JobMetadataMatch, string(payload))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "job_failed", "could not start match")
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]any{"jobId": jobID})
	}
}

// handleBookApply applies one provider issue's metadata to a single book, synchronously.
func handleBookApply(svc *metadata.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body applyRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
			return
		}
		if body.ProviderID == "" {
			writeError(w, http.StatusBadRequest, "bad_request", "providerId is required")
			return
		}
		if err := svc.ApplyBook(r.Context(), chi.URLParam(r, "id"), body.Provider, body.ProviderID, body.Fields); err != nil {
			writeMatchError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// patchBookRequest is the body of PATCH /books/{id} — manual per-book corrections. Each
// field is optional (pointer); a present field is applied. number/title/kind lock the field
// so rescans and provider matches keep the fix; ignored hides/restores the file.
type patchBookRequest struct {
	Number  *string `json:"number"`
	Title   *string `json:"title"`
	Kind    *string `json:"kind"`
	Ignored *bool   `json:"ignored"`
}

// handlePatchBook applies a manual correction to a book (number/title/kind/ignore).
func handlePatchBook(svc *metadata.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body patchBookRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
			return
		}
		if body.Number == nil && body.Title == nil && body.Kind == nil && body.Ignored == nil {
			writeError(w, http.StatusBadRequest, "bad_request", "no fields to update")
			return
		}
		edit := metadata.BookEdit{Number: body.Number, Title: body.Title, Ignored: body.Ignored}
		if body.Kind != nil {
			k := domain.BookKind(*body.Kind)
			edit.Kind = &k
		}
		if err := svc.EditBook(r.Context(), chi.URLParam(r, "id"), edit); err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				writeError(w, http.StatusNotFound, "not_found", "not found")
				return
			}
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// writeMatchError maps service errors to HTTP statuses: missing entities 404, an
// unconfigured provider 503, and upstream/provider failures 502.
func writeMatchError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "not found")
	case strings.Contains(err.Error(), "not configured"):
		writeError(w, http.StatusServiceUnavailable, "provider_unconfigured", err.Error())
	default:
		writeError(w, http.StatusBadGateway, "provider_error", err.Error())
	}
}
