package http

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	"github.com/siposbnc/comic-hub/server/internal/config"
	"github.com/siposbnc/comic-hub/server/internal/version"
)

func handleHealthz() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":  "ok",
			"version": version.Version,
		})
	}
}

func handleReadyz(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			writeError(w, http.StatusServiceUnavailable, "not_ready", "database unavailable")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "ready"})
	}
}

func handleServerInfo(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"name":    "ComicHub",
			"version": version.Version,
			"commit":  version.Commit,
			"mode":    cfg.Mode,
			// Feature flags let clients adapt to the server build (see docs/03-api.md §12).
			"capabilities": map[string]bool{
				"avif":      false,
				"pdf":       false,
				"epub":      false,
				"multiuser": cfg.Mode == config.ModeServer,
			},
		})
	}
}

func handleServerStats(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats := map[string]int64{}
		for key, table := range map[string]string{
			"libraries": "library",
			"series":    "series",
			"books":     "book",
		} {
			var n int64
			// Table names are fixed constants here, never user input.
			if err := db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM "+table).Scan(&n); err != nil {
				writeError(w, http.StatusInternalServerError, "stats_failed", "could not read stats")
				return
			}
			stats[key] = n
		}
		writeJSON(w, http.StatusOK, stats)
	}
}

func handleAuthHandshake(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// In embedded mode there is a single implicit owner; reaching this endpoint with
		// a valid token is proof of identity.
		writeJSON(w, http.StatusOK, map[string]any{
			"mode": cfg.Mode,
			"user": map[string]any{
				"id":          "owner",
				"username":    "owner",
				"displayName": "Owner",
				"role":        "owner",
			},
		})
	}
}

func handleShutdown(logger *slog.Logger, shutdown context.CancelFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Info("shutdown requested via API")
		writeJSON(w, http.StatusAccepted, map[string]any{"status": "shutting_down"})
		if shutdown != nil {
			// Let the response flush before tearing down.
			go func() {
				time.Sleep(100 * time.Millisecond)
				shutdown()
			}()
		}
	}
}
