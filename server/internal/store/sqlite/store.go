package sqlite

import (
	"database/sql"

	"github.com/siposbnc/comic-hub/server/internal/domain"
)

// Store is the SQLite-backed implementation of domain.Repository. It groups the
// per-entity repositories over a single *sql.DB. The rest of the server depends on
// the domain interfaces, not this type (see docs/01-architecture.md §11).
type Store struct {
	db        *sql.DB
	libraries *libraryRepo
	series    *seriesRepo
	books     *bookRepo
	progress  *progressRepo
	jobs      *jobRepo
	metadata  *metadataRepo
}

// NewStore wraps an open database in the catalog repositories.
func NewStore(db *sql.DB) *Store {
	return &Store{
		db:        db,
		libraries: &libraryRepo{db: db},
		series:    &seriesRepo{db: db},
		books:     &bookRepo{db: db},
		progress:  &progressRepo{db: db},
		jobs:      &jobRepo{db: db},
		metadata:  &metadataRepo{db: db},
	}
}

func (s *Store) Libraries() domain.LibraryRepository { return s.libraries }
func (s *Store) Series() domain.SeriesRepository     { return s.series }
func (s *Store) Books() domain.BookRepository        { return s.books }
func (s *Store) Progress() domain.ProgressRepository { return s.progress }
func (s *Store) Jobs() domain.JobRepository          { return s.jobs }
func (s *Store) Metadata() domain.MetadataRepository { return s.metadata }

// compile-time assertion that Store satisfies the domain boundary.
var _ domain.Repository = (*Store)(nil)
