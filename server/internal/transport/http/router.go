// Package http wires the server's HTTP + (future) WebSocket surface. It exposes a
// single API under /api/v1 plus unauthenticated liveness endpoints. See docs/03-api.md.
package http

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/siposbnc/comic-hub/server/internal/config"
	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/jobs"
	"github.com/siposbnc/comic-hub/server/internal/service/library"
)

// Deps are the dependencies the HTTP layer needs.
type Deps struct {
	Logger   *slog.Logger
	DB       *sql.DB
	Config   config.Config
	Shutdown context.CancelFunc
	Library  *library.Service
	Repo     domain.Repository
	Runner   *jobs.Runner
}

// NewRouter builds the HTTP handler tree.
func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(requestLogger(d.Logger))
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware())

	// Liveness/readiness are unauthenticated so the client can health-check the sidecar
	// before it has the token loaded.
	r.Get("/healthz", handleHealthz())
	r.Get("/readyz", handleReadyz(d.DB))

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(tokenAuth(d.Config))

		r.Get("/server/info", handleServerInfo(d.Config))
		r.Get("/server/stats", handleServerStats(d.DB))
		r.Get("/auth/handshake", handleAuthHandshake(d.Config))
		r.Post("/admin/shutdown", handleShutdown(d.Logger, d.Shutdown))

		r.Route("/libraries", func(r chi.Router) {
			r.Get("/", handleListLibraries(d.Library))
			r.Post("/", handleCreateLibrary(d.Library))
			r.Get("/{id}", handleGetLibrary(d.Library))
			r.Delete("/{id}", handleDeleteLibrary(d.Library))
			r.Post("/{id}/scan", handleScanLibrary(d.Library, d.Runner))
			r.Post("/{id}/scan/cancel", handleCancelScan(d.Runner, d.Repo))
		})

		r.Get("/jobs/{id}", handleGetJob(d.Repo))
	})

	return r
}

func corsMiddleware() func(http.Handler) http.Handler {
	return cors.Handler(cors.Options{
		// Tauri webviews and the Vite dev server. Loopback only by design.
		AllowedOrigins: []string{
			"http://localhost:*",
			"http://127.0.0.1:*",
			"tauri://localhost",
			"http://tauri.localhost",
			"https://tauri.localhost",
		},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		ExposedHeaders:   []string{"ETag"},
		AllowCredentials: false,
		MaxAge:           300,
	})
}

func requestLogger(l *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			l.LogAttrs(r.Context(), slog.LevelInfo, "http_request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", ww.Status()),
				slog.Int("bytes", ww.BytesWritten()),
				slog.Duration("dur", time.Since(start)),
				slog.String("reqid", middleware.GetReqID(r.Context())),
			)
		})
	}
}
