package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/jobs"
	"github.com/siposbnc/comic-hub/server/internal/scanner"
	"github.com/siposbnc/comic-hub/server/internal/service/library"
)

// ActiveScanJobID returns the id of a queued or running scan job for the library, if one
// exists. Used to coalesce scans: the file-watcher's debounced scan and an explicit scan
// (or two rapid explicit scans) must not run concurrently on the same library — overlapping
// scans race on series creation and waste work re-reading the same files.
func ActiveScanJobID(ctx context.Context, repo domain.Repository, libraryID string) (string, bool) {
	for _, state := range []domain.JobState{domain.JobRunning, domain.JobQueued} {
		list, err := repo.Jobs().ListByState(ctx, state, 200)
		if err != nil {
			continue
		}
		for _, j := range list {
			if j.Type != domain.JobScan {
				continue
			}
			var p scanner.JobPayload
			if json.Unmarshal([]byte(j.Payload), &p) == nil && p.LibraryID == libraryID {
				return j.ID, true
			}
		}
	}
	return "", false
}

// handleScanLibrary starts a scan job for a library and returns its job id. Body is
// optional: {"mode":"full"|"incremental"} (default incremental). If a scan is already
// queued or running for this library, its id is returned instead of starting another.
func handleScanLibrary(lib *library.Service, runner *jobs.Runner, repo domain.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if _, err := lib.Get(r.Context(), id); err != nil {
			writeDomainError(w, err)
			return
		}

		mode := "incremental"
		if r.ContentLength > 0 {
			var req struct {
				Mode string `json:"mode"`
			}
			if !decodeJSON(w, r, &req) {
				return
			}
			if req.Mode != "" {
				mode = req.Mode
			}
		}

		if existing, ok := ActiveScanJobID(r.Context(), repo, id); ok {
			writeJSON(w, http.StatusAccepted, map[string]any{"jobId": existing, "coalesced": true})
			return
		}

		payload, _ := json.Marshal(scanner.JobPayload{LibraryID: id, Full: mode == "full"})
		jobID, err := runner.Submit(r.Context(), domain.JobScan, string(payload))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "scan_failed", "could not start scan")
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]any{"jobId": jobID})
	}
}

// handleCancelScan cancels any running or queued scan jobs for a library.
func handleCancelScan(runner *jobs.Runner, repo domain.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		canceled := 0
		for _, state := range []domain.JobState{domain.JobRunning, domain.JobQueued} {
			list, err := repo.Jobs().ListByState(r.Context(), state, 200)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "cancel_failed", "could not list jobs")
				return
			}
			for _, j := range list {
				if j.Type != domain.JobScan {
					continue
				}
				var p scanner.JobPayload
				if json.Unmarshal([]byte(j.Payload), &p) == nil && p.LibraryID == id {
					runner.Cancel(j.ID)
					canceled++
				}
			}
		}
		writeJSON(w, http.StatusAccepted, map[string]any{"canceled": canceled})
	}
}

type jobDTO struct {
	ID         string  `json:"id"`
	Type       string  `json:"type"`
	State      string  `json:"state"`
	Progress   float64 `json:"progress"`
	Total      int64   `json:"total"`
	Done       int64   `json:"done"`
	Error      string  `json:"error,omitempty"`
	CreatedAt  int64   `json:"createdAt"`
	StartedAt  int64   `json:"startedAt,omitempty"`
	FinishedAt int64   `json:"finishedAt,omitempty"`
}

func toJobDTO(j domain.Job) jobDTO {
	return jobDTO{
		ID: j.ID, Type: j.Type, State: string(j.State), Progress: j.Progress,
		Total: j.Total, Done: j.Done, Error: j.Error,
		CreatedAt: j.CreatedAt, StartedAt: j.StartedAt, FinishedAt: j.FinishedAt,
	}
}

// handleGetJob returns a job's current state (also broadcast live on the WS jobs topic).
func handleGetJob(repo domain.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		j, err := repo.Jobs().Get(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toJobDTO(j))
	}
}
