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
	"github.com/siposbnc/comic-hub/server/internal/service/auth"
	"github.com/siposbnc/comic-hub/server/internal/service/browse"
	"github.com/siposbnc/comic-hub/server/internal/service/health"
	"github.com/siposbnc/comic-hub/server/internal/service/library"
	"github.com/siposbnc/comic-hub/server/internal/service/metadata"
	"github.com/siposbnc/comic-hub/server/internal/service/organize"
	"github.com/siposbnc/comic-hub/server/internal/service/presence"
	"github.com/siposbnc/comic-hub/server/internal/service/reader"
	"github.com/siposbnc/comic-hub/server/internal/service/reading"
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
	Reader   *reader.Service
	Browse   *browse.Service
	Reading  *reading.Service
	Metadata *metadata.Service
	Organize *organize.Service
	Health   *health.Service
	Auth     *auth.Service
	Hub      *Hub
	// Presence is the in-memory "now reading" tracker (Milestone E); nil disables the
	// snapshot endpoint (tests that don't wire it).
	Presence *presence.Tracker
	// ReloadProviders rebuilds the metadata service's providers from persisted settings +
	// env, after credentials change. Supplied by main.
	ReloadProviders func(context.Context) error
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
		r.Use(authMiddleware(d.Config, d.Auth))

		r.Get("/server/info", handleServerInfo(d.Config))
		r.Get("/server/stats", handleServerStats(d.DB))
		r.Get("/providers", handleProviders(d.Metadata))

		// Authentication (server mode). login/refresh/logout authenticate via the request
		// body (credentials / refresh token), so they bypass the access-token middleware.
		r.Post("/auth/login", handleLogin(d.Auth))
		r.Post("/auth/refresh", handleRefresh(d.Auth))
		r.Post("/auth/logout", handleLogout(d.Auth))

		// Provider credentials (metadata sources), editable from the settings screen.
		providerEnvCfg := providerEnv{
			ComicVineAPIKey: d.Config.ComicVineAPIKey,
			MetronUsername:  d.Config.MetronUsername,
			MetronPassword:  d.Config.MetronPassword,
		}
		r.Get("/settings/providers", handleGetProviderSettings(d.Repo, d.Metadata, providerEnvCfg))
		r.Put("/settings/providers", handlePutProviderSettings(d.Repo, d.Metadata, providerEnvCfg, d.ReloadProviders))
		r.Get("/auth/handshake", handleAuthHandshake(d.Config))
		// Admin-only: shutting the server down is privileged (a member must not be able to).
		r.With(requireRole(domain.RoleAdmin)).Post("/admin/shutdown", handleShutdown(d.Logger, d.Shutdown))

		// Account management (admin only).
		r.Route("/users", func(r chi.Router) {
			r.Use(requireRole(domain.RoleAdmin))
			r.Get("/", handleListUsers(d.Auth))
			r.Post("/", handleCreateUser(d.Auth))
			r.Patch("/{id}", handlePatchUser(d.Auth))
			r.Delete("/{id}", handleDeleteUser(d.Auth))
		})

		r.Route("/libraries", func(r chi.Router) {
			r.Get("/", handleListLibraries(d.Library))
			r.Post("/", handleCreateLibrary(d.Library))
			r.Get("/{id}", handleGetLibrary(d.Library))
			r.Delete("/{id}", handleDeleteLibrary(d.Library))
			r.Post("/{id}/scan", handleScanLibrary(d.Library, d.Runner, d.Repo))
			r.Post("/{id}/scan/cancel", handleCancelScan(d.Runner, d.Repo))
			r.Get("/{id}/health", handleLibraryHealth(d.Health))
		})

		r.Get("/jobs/{id}", handleGetJob(d.Repo))

		// Browse: series & books.
		r.Route("/series", func(r chi.Router) {
			r.Get("/", handleListSeries(d.Browse))
			r.Get("/{id}", handleSeriesDetail(d.Browse))
			r.Get("/{id}/story-arcs/{arcId}", handleStoryArcDetail(d.Browse))
			r.Get("/{id}/volumes/{volume}", handleVolumeDetail(d.Browse))
			r.Get("/{id}/match/candidates", handleSeriesCandidates(d.Metadata))
			r.Post("/{id}/match/apply", handleSeriesMatch(d.Metadata, d.Runner))
		})

		r.Route("/books", func(r chi.Router) {
			r.Get("/", handleListBooks(d.Browse))
			r.Get("/{id}", handleBookDetail(d.Browse))
			// Content routes enforce the restricted-user age ceiling (403 if exceeded).
			r.Group(func(r chi.Router) {
				r.Use(requireBookAccess(d.Repo))
				r.Get("/{id}/manifest", handleManifest(d.Reader))
				r.Get("/{id}/cover", handleCover(d.Reader))
				r.Get("/{id}/pages/{idx}", handlePage(d.Reader))
				r.Get("/{id}/pages/{idx}/thumb", handlePageThumb(d.Reader))
				r.Post("/{id}/prefetch", handlePrefetch(d.Reader))
			})
			r.Post("/{id}/match/apply", handleBookApply(d.Metadata))
			r.Post("/{id}/tags", handleAssignTags(d.Organize))
			r.Delete("/{id}/tags/{tagId}", handleUnassignTag(d.Organize))
		})

		// Tags: free-form labels applied across books.
		r.Route("/tags", func(r chi.Router) {
			r.Get("/", handleListTags(d.Organize))
			r.Post("/", handleCreateTag(d.Organize))
			r.Patch("/{id}", handleUpdateTag(d.Organize))
			r.Delete("/{id}", handleDeleteTag(d.Organize))
			r.Get("/{id}/books", handleTagBooks(d.Organize, d.Browse))
		})

		// Smart lists: rule-based, evaluated on demand.
		r.Route("/smart-lists", func(r chi.Router) {
			r.Get("/", handleListSmartLists(d.Organize))
			r.Post("/", handleCreateSmartList(d.Organize))
			r.Patch("/{id}", handleUpdateSmartList(d.Organize))
			r.Delete("/{id}", handleDeleteSmartList(d.Organize))
			r.Get("/{id}/results", handleSmartListResults(d.Organize, d.Browse))
		})

		r.Get("/discover", handleDiscover(d.Browse))
		r.Get("/search", handleSearch(d.Browse))

		// Collections: curated, ordered, shared shelves.
		r.Route("/collections", func(r chi.Router) {
			r.Get("/", handleListCollections(d.Organize))
			r.Post("/", handleCreateCollection(d.Organize))
			r.Get("/{id}", handleGetCollection(d.Organize, d.Browse))
			r.Patch("/{id}", handleUpdateCollection(d.Organize))
			r.Delete("/{id}", handleDeleteCollection(d.Organize))
			r.Post("/{id}/items", handleAddCollectionItems(d.Organize))
			r.Patch("/{id}/items/reorder", handleReorderCollectionItem(d.Organize))
			r.Delete("/{id}/items/{bookId}", handleRemoveCollectionItem(d.Organize))
		})

		// Progress & reading state (acting user = implicit owner in embedded mode).
		r.Route("/me", func(r chi.Router) {
			r.Get("/continue", handleContinueReading(d.Browse))
			r.Get("/progress/{bookId}", handleGetProgress(d.Reading))
			r.Put("/progress/{bookId}", handlePutProgress(d.Reading))
			r.Post("/progress/batch", handleBatchProgress(d.Reading))
			r.Post("/books/{id}/mark", handleMarkBook(d.Reading))
			r.Get("/books/{id}/next", handleNextBook(d.Browse))
			r.Get("/books/{id}/reader-prefs", handleGetReaderPrefs(d.Reading))
			r.Put("/books/{id}/reader-prefs", handlePutReaderPrefs(d.Reading))

			// Per-book bookmarks (page + optional note).
			r.Get("/books/{id}/bookmarks", handleListBookmarks(d.Reading))
			r.Post("/books/{id}/bookmarks", handleAddBookmark(d.Reading))
			r.Patch("/books/{id}/bookmarks/{bookmarkId}", handleUpdateBookmark(d.Reading))
			r.Delete("/books/{id}/bookmarks/{bookmarkId}", handleDeleteBookmark(d.Reading))

			// Personal reading lists (per-user, ordered).
			r.Route("/reading-lists", func(r chi.Router) {
				r.Get("/", handleListReadingLists(d.Organize))
				r.Post("/", handleCreateReadingList(d.Organize))
				r.Get("/{id}", handleGetReadingList(d.Organize, d.Browse))
				r.Patch("/{id}", handleUpdateReadingList(d.Organize))
				r.Delete("/{id}", handleDeleteReadingList(d.Organize))
				r.Post("/{id}/active", handleSetActiveReadingList(d.Organize))
				r.Post("/{id}/items", handleAddReadingListItems(d.Organize))
				r.Patch("/{id}/items/reorder", handleReorderReadingListItem(d.Organize))
				r.Delete("/{id}/items/{bookId}", handleRemoveReadingListItem(d.Organize))
			})
		})

		// Who's reading right now (Milestone E) — snapshot for initial render; live
		// updates arrive on the WS presence topic.
		if d.Presence != nil {
			r.Get("/presence", handlePresence(d.Presence))
		}

		// Multiplexed push socket (jobs/progress/bookmarks/presence topics).
		if d.Hub != nil {
			r.Get("/ws", d.Hub.handle())
		}
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
