// Command comichub-server is the ComicHub media server: a single binary that owns the
// library catalog, database, files, caches, and background work. It runs embedded
// (spawned by the client as a sidecar) or standalone (a remote always-on server).
package main

import (
	"context"
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
	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/image"
	"github.com/siposbnc/comic-hub/server/internal/jobs"
	"github.com/siposbnc/comic-hub/server/internal/logging"
	"github.com/siposbnc/comic-hub/server/internal/providers"
	"github.com/siposbnc/comic-hub/server/internal/providers/comicvine"
	"github.com/siposbnc/comic-hub/server/internal/scanner"
	"github.com/siposbnc/comic-hub/server/internal/service/browse"
	"github.com/siposbnc/comic-hub/server/internal/service/library"
	"github.com/siposbnc/comic-hub/server/internal/service/metadata"
	"github.com/siposbnc/comic-hub/server/internal/service/organize"
	"github.com/siposbnc/comic-hub/server/internal/service/reader"
	"github.com/siposbnc/comic-hub/server/internal/service/reading"
	"github.com/siposbnc/comic-hub/server/internal/store/sqlite"
	httptransport "github.com/siposbnc/comic-hub/server/internal/transport/http"
	"github.com/siposbnc/comic-hub/server/internal/version"
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
	logger.Info("starting comichub-server",
		"version", version.Version,
		"mode", cfg.Mode,
		"dataDir", cfg.DataDir,
	)

	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	db, err := sqlite.Open(cfg.Database.DSN())
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	if err := sqlite.Migrate(context.Background(), db); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	logger.Info("database ready", "path", cfg.Database.Path)

	// Catalog store + application services over the domain.Repository boundary.
	store := sqlite.NewStore(db)
	libraries := library.New(store)
	browsing := browse.New(store)
	organizing := organize.New(store)

	// WebSocket hub for push (jobs/progress); services broadcast through it.
	hub := httptransport.NewHub(logger)
	readingSvc := reading.New(store, func(_ string, p domain.Progress) { hub.BroadcastProgress(p) })

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
	runner.Register(domain.JobScan, func(ctx context.Context, payload string, progress jobs.ProgressFunc) error {
		var p scanner.JobPayload
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			return err
		}
		return sc.Scan(ctx, p.LibraryID, p.Full, scanner.ProgressFunc(progress))
	})

	// Online metadata matching (Comic Vine via COMICVINE_API_KEY). The metadata_match
	// job batch-applies a chosen provider volume to a series' books.
	var metaProviders []providers.Provider
	if cfg.ComicVineAPIKey != "" {
		metaProviders = append(metaProviders, comicvine.New(cfg.ComicVineAPIKey))
		logger.Info("metadata provider configured", "provider", "comicvine")
	}
	metaSvc := metadata.New(store, metaProviders...)
	runner.Register(domain.JobMetadataMatch, func(ctx context.Context, payload string, progress jobs.ProgressFunc) error {
		var p httptransport.MatchJobPayload
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			return err
		}
		return metaSvc.MatchSeries(ctx, p.SeriesID, p.Provider, p.VolumeProviderID, p.Fields,
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

	handler := httptransport.NewRouter(httptransport.Deps{
		Logger:   logger,
		DB:       db,
		Config:   cfg,
		Shutdown: appCancel,
		Library:  libraries,
		Repo:     store,
		Runner:   runner,
		Reader:   readerSvc,
		Browse:   browsing,
		Reading:  readingSvc,
		Metadata: metaSvc,
		Organize: organizing,
		Hub:      hub,
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
