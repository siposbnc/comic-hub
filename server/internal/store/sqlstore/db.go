// Package sqlstore provides the SQL-backed catalog store and migration runner —
// SQLite by default (embedded, zero-admin), Postgres opt-in for managed deployments
// (ADR-005). One shared implementation: repositories write portable SQL with `?`
// placeholders, and the DB wrapper rebinds them to `$n` for Postgres. The rest of
// the server depends on the domain.Repository interface, not this package.
package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib" // Postgres driver, registered as "pgx"
	_ "modernc.org/sqlite"             // pure-Go SQLite driver, registered as "sqlite"
)

// Driver selects the SQL dialect.
type Driver string

const (
	DriverSQLite   Driver = "sqlite"
	DriverPostgres Driver = "postgres"
)

// DB wraps *sql.DB with the dialect, rebinding `?` placeholders to `$n` for
// Postgres so repository SQL is written once.
type DB struct {
	sql *sql.DB
	drv Driver
}

// Open connects with the given driver and verifies connectivity.
// SQLite DSNs carry their pragmas (see config.DSN); Postgres DSNs are standard
// `postgres://user:pass@host/db` URLs.
func Open(driver Driver, dsn string) (*DB, error) {
	name := "sqlite"
	switch driver {
	case DriverPostgres:
		name = "pgx"
	case DriverSQLite, "":
		driver = DriverSQLite
	default:
		return nil, fmt.Errorf("unsupported db driver %q (want sqlite|postgres)", driver)
	}
	db, err := sql.Open(name, dsn)
	if err != nil {
		return nil, err
	}
	if driver == DriverSQLite {
		// SQLite serializes writes; a small pool with WAL gives concurrent readers.
		db.SetMaxOpenConns(8)
		db.SetMaxIdleConns(8)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &DB{sql: db, drv: driver}, nil
}

// OpenSQLite opens a SQLite database — the embedded default and the test helper.
func OpenSQLite(dsn string) (*DB, error) { return Open(DriverSQLite, dsn) }

// Driver reports the dialect this store runs on.
func (d *DB) Driver() Driver { return d.drv }

// Unwrap exposes the raw *sql.DB (readiness probes, server stats).
func (d *DB) Unwrap() *sql.DB { return d.sql }

func (d *DB) Close() error { return d.sql.Close() }

func (d *DB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return d.sql.ExecContext(ctx, d.rebind(query), args...)
}

func (d *DB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return d.sql.QueryContext(ctx, d.rebind(query), args...)
}

func (d *DB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return d.sql.QueryRowContext(ctx, d.rebind(query), args...)
}

// Tx mirrors the wrapper over a transaction.
type Tx struct {
	tx  *sql.Tx
	drv Driver
}

func (d *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	tx, err := d.sql.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &Tx{tx: tx, drv: d.drv}, nil
}

func (t *Tx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return t.tx.ExecContext(ctx, rebind(t.drv, query), args...)
}

func (t *Tx) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return t.tx.QueryContext(ctx, rebind(t.drv, query), args...)
}

func (t *Tx) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return t.tx.QueryRowContext(ctx, rebind(t.drv, query), args...)
}

func (t *Tx) Commit() error   { return t.tx.Commit() }
func (t *Tx) Rollback() error { return t.tx.Rollback() }

func (d *DB) rebind(query string) string { return rebind(d.drv, query) }

// rebind converts `?` placeholders to `$1..$n` for Postgres, skipping quoted
// literals and identifiers. Repository SQL never uses `?` any other way (no JSON
// operators), so this stays a simple scan.
func rebind(drv Driver, query string) string {
	if drv != DriverPostgres || !strings.ContainsRune(query, '?') {
		return query
	}
	var b strings.Builder
	b.Grow(len(query) + 8)
	n := 0
	var quote byte
	for i := 0; i < len(query); i++ {
		c := query[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
			b.WriteByte(c)
		case c == '\'' || c == '"':
			quote = c
			b.WriteByte(c)
		case c == '?':
			n++
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(n))
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}
