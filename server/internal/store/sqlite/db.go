// Package sqlite provides the SQLite-backed catalog store and migration runner.
// The rest of the server depends on the domain.Repository interface, not this package
// directly, so a Postgres backend can be added later without touching the domain.
package sqlite

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // pure-Go driver, registered as "sqlite"
)

// Open opens (creating if needed) the SQLite database at the given DSN and verifies
// connectivity. WAL + foreign keys are enabled via DSN pragmas (see config.DSN).
func Open(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// SQLite serializes writes; a small pool with WAL gives concurrent readers.
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(8)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return db, nil
}
