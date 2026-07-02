// Command comichub-server is the ComicHub media server: a single binary that owns the
// library catalog, database, files, caches, and background work. It runs embedded
// (spawned by the client as a sidecar) or standalone (a remote always-on server).
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/siposbnc/comic-hub/server/internal/archive"
	"github.com/siposbnc/comic-hub/server/internal/config"
	"github.com/siposbnc/comic-hub/server/internal/connection"
	"github.com/siposbnc/comic-hub/server/internal/discovery"
	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/image"
	"github.com/siposbnc/comic-hub/server/internal/jobs"
	"github.com/siposbnc/comic-hub/server/internal/logging"
	"github.com/siposbnc/comic-hub/server/internal/providers"
	"github.com/siposbnc/comic-hub/server/internal/providers/comicvine"
	"github.com/siposbnc/comic-hub/server/internal/providers/metron"
	"github.com/siposbnc/comic-hub/server/internal/scanner"
	"github.com/siposbnc/comic-hub/server/internal/service/auth"
	"github.com/siposbnc/comic-hub/server/internal/service/browse"
	"github.com/siposbnc/comic-hub/server/internal/service/health"
	"github.com/siposbnc/comic-hub/server/internal/service/library"
	"github.com/siposbnc/comic-hub/server/internal/service/metadata"
	"github.com/siposbnc/comic-hub/server/internal/service/organize"
	"github.com/siposbnc/comic-hub/server/internal/service/presence"
	"github.com/siposbnc/comic-hub/server/internal/service/reader"
	"github.com/siposbnc/comic-hub/server/internal/service/reading"
	"github.com/siposbnc/comic-hub/server/internal/service/sidecar"
	"github.com/siposbnc/comic-hub/server/internal/service/stats"
	"github.com/siposbnc/comic-hub/server/internal/store/sqlstore"
	httptransport "github.com/siposbnc/comic-hub/server/internal/transport/http"
	"github.com/siposbnc/comic-hub/server/internal/version"
	"github.com/siposbnc/comic-hub/server/internal/watch"
)

// hashLargeThreshold is the file size above which content hashing switches to sampled
// mode (config-driven later; see docs/04-server.md §9).
const hashLargeThreshold = 256 << 20 // 256 MiB

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		return err
	}

	logger := logging.New(cfg.LogLevel, cfg.LogFormat)
	slog.SetDefault(logger)
	// Everything of ours logs through slog; what arrives via the std `log` bridge is
	// third-party library chatter (e.g. zeroconf warns per interface on every mDNS
	// response on Windows) — keep it out of production logs, visible at --log-level debug.
	slog.SetLogLoggerLevel(slog.LevelDebug)
	logger.Info("starting comichub-server",
		"version", version.Version,
		"mode", cfg.Mode,
		"dataDir", cfg.DataDir,
	)

	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	db, err := sqlstore.Open(sqlstore.Driver(cfg.Database.Driver), cfg.Database.DSN())
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	if err := sqlstore.Migrate(context.Background(), db); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	// The Postgres DSN may carry credentials — log only the driver and sqlite path.
	logger.Info("database ready", "driver", cfg.Database.Driver, "path", cfg.Database.Path)

	// Catalog store + application services over the domain.Repository boundary.
	store := sqlstore.NewStore(db)
	libraries := library.New(store)
	browsing := browse.New(store)
	organizing := organize.New(store)
	healthSvc := health.New(store)
	statsSvc := stats.New(store)

	// WebSocket hub for push (jobs/progress/bookmarks/presence); services broadcast
	// through it. Presence ("now reading", Milestone E) is derived from progress writes:
	// an in-progress write touches the user's presence entry (enriched here for direct
	// display); finishing or marking clears it; the tracker expires idle readers.
	hub := httptransport.NewHub(logger)
	presenceTracker := presence.New(presence.DefaultTTL)
	presenceTracker.OnChange(hub.BroadcastPresence)
	observePresence := presenceTracker.ObserveProgress(store)
	readingSvc := reading.New(store, func(userID string, p domain.Progress) {
		hub.BroadcastProgress(p)
		observePresence(userID, p)
	})
	readingSvc.OnBookmarkChange(hub.BroadcastBookmarks)

	// Shared format registry for scanning and reading.
	registry := archive.DefaultRegistry()

	// Image pipeline: pure-Go processor + on-disk derived-image cache.
	derivedCache, err := image.NewDiskCache(filepath.Join(cfg.DataDir, "cache", "derived"))
	if err != nil {
		return fmt.Errorf("init image cache: %w", err)
	}
	readerSvc, err := reader.New(store, registry, image.New(), derivedCache)
	if err != nil {
		return fmt.Errorf("init reader service: %w", err)
	}

	// Background jobs: the scanner runs as a "scan" job on the worker pool.
	runner := jobs.NewRunner(store, logger, 4)
	runner.OnUpdate(hub.BroadcastJob)
	defer runner.Shutdown()
	sc := scanner.New(store, registry, logger, hashLargeThreshold)

	// Online metadata matching. Provider credentials come from persisted settings, falling
	// back to env vars — so the settings UI can configure them at runtime (and the packaged
	// app needs no env vars). Multiple providers can be configured; matching searches them
	// all and ranks the combined candidates. The metadata_automatch job (chained after a
	// scan) auto-applies 100% matches and flags the rest incomplete.
	buildProviders := func(ctx context.Context) []providers.Provider {
		saved, _ := store.Settings().GetAll(ctx)
		eff := func(key, fallback string) string {
			if v, ok := saved[key]; ok {
				return v
			}
			return fallback
		}
		cvKey := eff(domain.SettingComicVineAPIKey, cfg.ComicVineAPIKey)
		mUser := eff(domain.SettingMetronUsername, cfg.MetronUsername)
		mPass := eff(domain.SettingMetronPassword, cfg.MetronPassword)
		var provs []providers.Provider
		if cvKey != "" {
			provs = append(provs, comicvine.New(cvKey))
		}
		if mUser != "" && mPass != "" {
			provs = append(provs, metron.New(mUser, mPass))
		}
		return provs
	}

	metaSvc := metadata.New(store, buildProviders(context.Background())...)
	logger.Info("metadata providers configured", "providers", metaSvc.Names())
	reloadProviders := func(ctx context.Context) error {
		metaSvc.Configure(buildProviders(ctx)...)
		logger.Info("metadata providers reloaded", "providers", metaSvc.Names())
		return nil
	}

	// Opt-in: after applying metadata to a book, write it back into the archive as a
	// ComicInfo.xml when the user has enabled it in settings (checked live per book).
	sidecarWriter := sidecar.New(store, hashLargeThreshold)
	metaSvc.OnApply(func(ctx context.Context, bookID string) {
		if v, _ := store.Settings().Get(ctx, domain.SettingWriteSidecar); v != "true" {
			return
		}
		if err := sidecarWriter.Write(ctx, bookID); err != nil {
			logger.Warn("write sidecar failed", "book", bookID, "err", err)
		}
	})

	// Authentication (server mode). Resolve the access-token signing secret (env → persisted
	// setting → freshly generated + persisted), build the auth service, and bootstrap a
	// login-capable admin from env when configured. Embedded/dev runs leave auth disabled and
	// act as the implicit owner; the secret is still resolved so the service is always usable.
	jwtSecret, err := resolveJWTSecret(context.Background(), store, cfg)
	if err != nil {
		return fmt.Errorf("resolve auth secret: %w", err)
	}
	authSvc := auth.New(store, jwtSecret)
	if cfg.AuthEnabled {
		adminUser := cfg.AdminUsername
		if adminUser == "" && cfg.AdminPassword != "" {
			adminUser = domain.OwnerUserID // default: set the implicit owner's password
		}
		if err := authSvc.EnsureAdmin(context.Background(), adminUser, cfg.AdminDisplayName, cfg.AdminPassword); err != nil {
			return fmt.Errorf("bootstrap admin: %w", err)
		}
		logger.Info("authentication enabled", "adminBootstrapped", adminUser != "")
	}

	runner.Register(domain.JobScan, func(ctx context.Context, payload string, progress jobs.ProgressFunc) error {
		var p scanner.JobPayload
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			return err
		}
		if err := sc.Scan(ctx, p.LibraryID, p.Full, scanner.ProgressFunc(progress)); err != nil {
			return err
		}
		// After a successful scan, auto-match newly-added series (100% matches applied; the
		// rest flagged incomplete for manual matching) when a provider is configured.
		if metaSvc.HasProviders() {
			amPayload, _ := json.Marshal(httptransport.AutoMatchJobPayload{LibraryID: p.LibraryID})
			if _, err := runner.Submit(ctx, domain.JobMetadataAutoMatch, string(amPayload)); err != nil {
				logger.Warn("scan: enqueue auto-match failed", "library", p.LibraryID, "err", err)
			}
		}
		return nil
	})

	runner.Register(domain.JobMetadataMatch, func(ctx context.Context, payload string, progress jobs.ProgressFunc) error {
		var p httptransport.MatchJobPayload
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			return err
		}
		return metaSvc.MatchSeries(ctx, p.SeriesID, p.Provider, p.VolumeProviderID, p.Fields,
			func(done, total int) { progress(int64(done), int64(total)) })
	})

	runner.Register(domain.JobMetadataAutoMatch, func(ctx context.Context, payload string, progress jobs.ProgressFunc) error {
		var p httptransport.AutoMatchJobPayload
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			return err
		}
		return metaSvc.AutoMatchLibrary(ctx, p.LibraryID,
			func(done, total int) { progress(int64(done), int64(total)) })
	})

	ln, err := net.Listen("tcp", cfg.Bind)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", cfg.Bind, err)
	}
	addr := ln.Addr().(*net.TCPAddr)

	// appCtx is cancelled either by an OS signal or the /admin/shutdown endpoint.
	appCtx, appCancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer appCancel()

	// Expire idle readers out of the presence set.
	go presenceTracker.Run(appCtx)

	// File-watching: a debounced incremental rescan whenever a library's files change on
	// disk. A moved/renamed file is reconciled by the scanner via content hash.
	if watcher, werr := watch.New(logger, 2*time.Second, func(libraryID string) {
		// Coalesce: skip if a scan for this library is already queued/running, so a
		// watcher event can't spawn a scan that overlaps an in-flight one (which would
		// race on series creation and re-read the same files).
		if existing, ok := httptransport.ActiveScanJobID(appCtx, store, libraryID); ok {
			logger.Debug("watch: scan already active, skipping", "library", libraryID, "job", existing)
			return
		}
		payload, _ := json.Marshal(scanner.JobPayload{LibraryID: libraryID, Full: false})
		if _, err := runner.Submit(appCtx, domain.JobScan, string(payload)); err != nil {
			logger.Warn("watch: enqueue scan failed", "library", libraryID, "err", err)
		}
	}); werr != nil {
		logger.Warn("file-watching disabled", "err", werr)
	} else {
		defer watcher.Close()
		if libs, err := store.Libraries().List(appCtx); err == nil {
			for _, l := range libs {
				watcher.Add(l.ID, l.Roots)
			}
		}
		libraries.OnCreate(func(l domain.Library) { watcher.Add(l.ID, l.Roots) })
		libraries.OnDelete(func(id string) { watcher.Remove(id) })
		go watcher.Run(appCtx)
		logger.Info("file-watching enabled")
	}

	handler := httptransport.NewRouter(httptransport.Deps{
		Logger:   logger,
		DB:       db.Unwrap(),
		Config:   cfg,
		Shutdown: appCancel,
		Library:  libraries,
		Repo:     store,
		Runner:   runner,
		Reader:   readerSvc,
		Browse:   browsing,
		Reading:  readingSvc,
		Stats:    statsSvc,
		Metadata: metaSvc,
		Organize: organizing,
		Health:   healthSvc,
		Auth:     authSvc,
		Hub:      hub,
		Presence: presenceTracker,

		ReloadProviders: reloadProviders,
	})

	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// In embedded mode, publish the chosen port + loopback token so the client can connect.
	if cfg.Mode == config.ModeEmbedded {
		hs := connection.Handshake{
			Port:    addr.Port,
			Token:   cfg.Token,
			PID:     os.Getpid(),
			Version: version.Version,
			BaseURL: fmt.Sprintf("http://127.0.0.1:%d", addr.Port),
		}
		if err := connection.Write(cfg.HandshakeFile, hs); err != nil {
			return fmt.Errorf("write handshake: %w", err)
		}
		logger.Info("handshake published", "file", cfg.HandshakeFile, "port", addr.Port)
	}

	// In server mode, advertise over mDNS so clients on the LAN can discover this
	// server without typing a URL (Milestone D). Best-effort: a network stack that
	// can't do multicast just logs a warning and manual pairing still works.
	if cfg.Mode == config.ModeServer && cfg.MDNS {
		adv, err := discovery.Advertise(discovery.Info{
			Instance:     cfg.ServerName,
			Port:         addr.Port,
			Version:      version.Version,
			AuthRequired: cfg.AuthEnabled,
		})
		if err != nil {
			logger.Warn("mDNS advertising disabled", "err", err)
		} else {
			defer adv.Close()
			logger.Info("mDNS advertising enabled",
				"instance", cfg.ServerName, "service", discovery.ServiceType, "port", addr.Port)
		}
	}

	serveErr := make(chan error, 1)
	go func() {
		logger.Info("listening", "addr", addr.String())
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

	select {
	case err := <-serveErr:
		return fmt.Errorf("serve: %w", err)
	case <-appCtx.Done():
		logger.Info("shutdown requested")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	logger.Info("stopped cleanly")
	return nil
}

// resolveJWTSecret returns the access-token signing secret: the configured value, else the
// persisted one, else a freshly generated 32-byte secret that is persisted so tokens survive
// restarts. Generation/persistence only happens when auth is enabled (an unused secret in
// embedded/dev mode stays ephemeral and never touches the database).
func resolveJWTSecret(ctx context.Context, store *sqlstore.Store, cfg config.Config) ([]byte, error) {
	if cfg.JWTSecret != "" {
		return []byte(cfg.JWTSecret), nil
	}
	if saved, err := store.Settings().Get(ctx, domain.SettingJWTSecret); err == nil && saved != "" {
		return []byte(saved), nil
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	secret := hex.EncodeToString(buf)
	if cfg.AuthEnabled {
		if err := store.Settings().Set(ctx, domain.SettingJWTSecret, secret); err != nil {
			return nil, err
		}
	}
	return []byte(secret), nil
}
