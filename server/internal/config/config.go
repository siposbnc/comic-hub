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

	// AuthEnabled turns on multi-user authentication (server mode): access tokens required,
	// login/refresh public. Off by default — embedded mode and dev runs use the implicit
	// owner. (Phase 3 — Milestone A.)
	AuthEnabled bool

	// MDNS advertises the server over mDNS/DNS-SD on the LAN so clients can discover it
	// (Phase 3 — Milestone D). On by default, but only effective in server mode — an
	// embedded sidecar never advertises. `--mdns=false` / COMICHUB_MDNS=false opts out.
	MDNS bool
	// ServerName is the human-readable instance name shown in clients' discovery lists
	// (--server-name / COMICHUB_SERVER_NAME; defaults to the machine hostname).
	ServerName string
	// JWTSecret signs access tokens (COMICHUB_JWT_SECRET). When auth is enabled and this is
	// empty, the server generates and persists one.
	JWTSecret string
	// Admin* bootstrap a login-capable admin on boot (COMICHUB_ADMIN_USERNAME / _PASSWORD /
	// _DISPLAY_NAME), so a packaged/Docker deployment can seed the first account from env.
	AdminUsername    string
	AdminPassword    string
	AdminDisplayName string
}

// DatabaseConfig describes the catalog store. SQLite (a file in the data dir) is the
// default; Postgres is opt-in for managed deployments (--db-driver postgres +
// --db-dsn / COMICHUB_DB_DSN; see docs/10-deployment.md §4).
type DatabaseConfig struct {
	Driver string // sqlite | postgres
	Path   string // sqlite file path
	// ConnString is the explicit connection string (required for postgres; overrides
	// Path-derived DSN for sqlite when set).
	ConnString string
}

// DSN returns the database connection string.
func (d DatabaseConfig) DSN() string {
	if d.ConnString != "" {
		return d.ConnString
	}
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
	dbDriver := fs.String("db-driver", env("DB_DRIVER", "sqlite"), "database driver: sqlite|postgres")
	dbDSN := fs.String("db-dsn", env("DB_DSN", ""), "database connection string (required for postgres)")
	authEnabled := fs.Bool("auth", env("AUTH_ENABLED", "") == "true", "enable multi-user authentication (server mode)")
	mdns := fs.Bool("mdns", env("MDNS", "true") != "false", "advertise the server over mDNS on the LAN (server mode)")
	serverName := fs.String("server-name", env("SERVER_NAME", ""), "server name shown in clients' discovery lists (default: hostname)")

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
		Database:      DatabaseConfig{Driver: *dbDriver, Path: *dbPath, ConnString: *dbDSN},
		// Provider keys are env-only (never flags, so they don't leak into shell history).
		ComicVineAPIKey: strings.TrimSpace(os.Getenv("COMICVINE_API_KEY")),
		MetronUsername:  strings.TrimSpace(os.Getenv("METRON_USERNAME")),
		MetronPassword:  strings.TrimSpace(os.Getenv("METRON_PASSWORD")),

		MDNS:       *mdns,
		ServerName: *serverName,

		// Auth: secret + admin bootstrap are env-only (sensitive); the toggle is a flag.
		AuthEnabled:      *authEnabled,
		JWTSecret:        strings.TrimSpace(os.Getenv("COMICHUB_JWT_SECRET")),
		AdminUsername:    strings.TrimSpace(os.Getenv("COMICHUB_ADMIN_USERNAME")),
		AdminPassword:    os.Getenv("COMICHUB_ADMIN_PASSWORD"),
		AdminDisplayName: strings.TrimSpace(os.Getenv("COMICHUB_ADMIN_DISPLAY_NAME")),
	}

	if cfg.Mode != ModeEmbedded && cfg.Mode != ModeServer {
		return Config{}, fmt.Errorf("invalid mode %q (want embedded|server)", cfg.Mode)
	}
	if cfg.Database.Driver != "sqlite" && cfg.Database.Driver != "postgres" {
		return Config{}, fmt.Errorf("invalid db driver %q (want sqlite|postgres)", cfg.Database.Driver)
	}
	if cfg.Database.Driver == "postgres" && cfg.Database.ConnString == "" {
		return Config{}, fmt.Errorf("--db-dsn (COMICHUB_DB_DSN) is required with the postgres driver")
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

	if cfg.ServerName == "" {
		if host, err := os.Hostname(); err == nil {
			cfg.ServerName = host
		} else {
			cfg.ServerName = "ComicHub"
		}
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
