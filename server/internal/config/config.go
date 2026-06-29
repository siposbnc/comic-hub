// Package config loads server configuration from flags and environment variables.
// Precedence: flags > environment (COMICHUB_*) > built-in defaults. A TOML config
// file is planned (see docs/04-server.md §9) and will slot in below flags.
package config

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Mode is the server deployment mode.
type Mode string

const (
	// ModeEmbedded is the default: spawned by the client, binds loopback, publishes a
	// handshake file with its port + token. Single implicit owner.
	ModeEmbedded Mode = "embedded"
	// ModeServer is a standalone always-on server (LAN/remote) with real auth.
	ModeServer Mode = "server"
)

// Config is the fully-resolved server configuration.
type Config struct {
	Mode          Mode
	Bind          string // host:port; port 0 = ephemeral (embedded default)
	DataDir       string
	Token         string // loopback bearer token; empty disables auth
	HandshakeFile string
	LogLevel      string // debug|info|warn|error
	LogFormat     string // json|text
	Database      DatabaseConfig

	// ComicVineAPIKey is the Comic Vine metadata provider key, read from the
	// COMICVINE_API_KEY environment variable (server-side only; never sent to clients).
	// Empty disables online matching against Comic Vine.
	ComicVineAPIKey string

	// MetronUsername / MetronPassword authenticate the Metron provider (HTTP Basic Auth),
	// read from METRON_USERNAME / METRON_PASSWORD (server-side only). Both must be set to
	// enable matching against Metron.
	MetronUsername string
	MetronPassword string
}

// DatabaseConfig describes the catalog store.
type DatabaseConfig struct {
	Driver string // sqlite (postgres planned)
	Path   string // sqlite file path
}

// DSN returns the database connection string.
func (d DatabaseConfig) DSN() string {
	// modernc.org/sqlite pragma params: WAL for concurrent reads, FK enforcement,
	// a busy timeout so transient locks retry instead of erroring.
	return fmt.Sprintf(
		"file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)",
		d.Path,
	)
}

// Load resolves configuration from the given args and the environment.
func Load(args []string) (Config, error) {
	fs := flag.NewFlagSet("comichub-server", flag.ContinueOnError)

	defaultDataDir := defaultDataDir()

	mode := fs.String("mode", env("MODE", string(ModeEmbedded)), "deployment mode: embedded|server")
	dataDir := fs.String("data-dir", env("DATA_DIR", defaultDataDir), "data directory")
	bind := fs.String("bind", env("BIND", ""), "bind address host:port (default depends on mode)")
	token := fs.String("token", env("TOKEN", ""), "loopback bearer token (auto-generated in embedded mode if empty)")
	handshake := fs.String("handshake-file", env("HANDSHAKE_FILE", ""), "path to write the connection handshake (embedded mode)")
	logLevel := fs.String("log-level", env("LOG_LEVEL", "info"), "log level: debug|info|warn|error")
	logFormat := fs.String("log-format", env("LOG_FORMAT", "json"), "log format: json|text")
	dbPath := fs.String("db", env("DB_PATH", ""), "sqlite database path (default <data-dir>/comichub.db)")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	cfg := Config{
		Mode:          Mode(*mode),
		DataDir:       *dataDir,
		Bind:          *bind,
		Token:         *token,
		HandshakeFile: *handshake,
		LogLevel:      *logLevel,
		LogFormat:     *logFormat,
		Database:      DatabaseConfig{Driver: "sqlite", Path: *dbPath},
		// Provider keys are env-only (never flags, so they don't leak into shell history).
		ComicVineAPIKey: strings.TrimSpace(os.Getenv("COMICVINE_API_KEY")),
		MetronUsername:  strings.TrimSpace(os.Getenv("METRON_USERNAME")),
		MetronPassword:  strings.TrimSpace(os.Getenv("METRON_PASSWORD")),
	}

	if cfg.Mode != ModeEmbedded && cfg.Mode != ModeServer {
		return Config{}, fmt.Errorf("invalid mode %q (want embedded|server)", cfg.Mode)
	}

	// Mode-dependent defaults.
	if cfg.Bind == "" {
		if cfg.Mode == ModeEmbedded {
			cfg.Bind = "127.0.0.1:0" // ephemeral loopback
		} else {
			cfg.Bind = "0.0.0.0:8080"
		}
	}
	if cfg.Database.Path == "" {
		cfg.Database.Path = filepath.Join(cfg.DataDir, "comichub.db")
	}
	if cfg.HandshakeFile == "" {
		cfg.HandshakeFile = filepath.Join(cfg.DataDir, "connection.json")
	}

	// In embedded mode, generate a loopback token if none was supplied.
	if cfg.Mode == ModeEmbedded && cfg.Token == "" {
		tok, err := generateToken()
		if err != nil {
			return Config{}, fmt.Errorf("generate token: %w", err)
		}
		cfg.Token = tok
	}

	return cfg, nil
}

func env(key, fallback string) string {
	if v, ok := os.LookupEnv("COMICHUB_" + key); ok && strings.TrimSpace(v) != "" {
		return v
	}
	return fallback
}

func defaultDataDir() string {
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "ComicHub")
	}
	return filepath.Join(".", ".comichub")
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
