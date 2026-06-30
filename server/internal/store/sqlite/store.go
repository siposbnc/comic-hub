package sqlite

import (
	"database/sql"

	"github.com/siposbnc/comic-hub/server/internal/domain"
)

// Store is the SQLite-backed implementation of domain.Repository. It groups the
// per-entity repositories over a single *sql.DB. The rest of the server depends on
// the domain interfaces, not this type (see docs/01-architecture.md §11).
type Store struct {
	db           *sql.DB
	libraries    *libraryRepo
	series       *seriesRepo
	books        *bookRepo
	progress     *progressRepo
	bookmarks    *bookmarkRepo
	jobs         *jobRepo
	metadata     *metadataRepo
	search       *searchRepo
	collections  *collectionRepo
	readingLists *readingListRepo
	tags         *tagRepo
	smartLists   *smartListRepo
	readerPrefs  *readerPrefRepo
	settings     *settingsRepo
	users        *userRepo
	sessions     *sessionRepo
}

// NewStore wraps an open database in the catalog repositories.
func NewStore(db *sql.DB) *Store {
	return &Store{
		db:           db,
		libraries:    &libraryRepo{db: db},
		series:       &seriesRepo{db: db},
		books:        &bookRepo{db: db},
		progress:     &progressRepo{db: db},
		bookmarks:    &bookmarkRepo{db: db},
		jobs:         &jobRepo{db: db},
		metadata:     &metadataRepo{db: db},
		search:       &searchRepo{db: db},
		collections:  &collectionRepo{db: db},
		readingLists: &readingListRepo{db: db},
		tags:         &tagRepo{db: db},
		smartLists:   &smartListRepo{db: db},
		readerPrefs:  &readerPrefRepo{db: db},
		settings:     &settingsRepo{db: db},
		users:        &userRepo{db: db},
		sessions:     &sessionRepo{db: db},
	}
}

func (s *Store) Libraries() domain.LibraryRepository        { return s.libraries }
func (s *Store) Series() domain.SeriesRepository            { return s.series }
func (s *Store) Books() domain.BookRepository               { return s.books }
func (s *Store) Progress() domain.ProgressRepository        { return s.progress }
func (s *Store) Bookmarks() domain.BookmarkRepository       { return s.bookmarks }
func (s *Store) Jobs() domain.JobRepository                 { return s.jobs }
func (s *Store) Metadata() domain.MetadataRepository        { return s.metadata }
func (s *Store) Search() domain.SearchRepository            { return s.search }
func (s *Store) Collections() domain.CollectionRepository   { return s.collections }
func (s *Store) ReadingLists() domain.ReadingListRepository { return s.readingLists }
func (s *Store) Tags() domain.TagRepository                 { return s.tags }
func (s *Store) SmartLists() domain.SmartListRepository     { return s.smartLists }
func (s *Store) ReaderPrefs() domain.ReaderPrefRepository   { return s.readerPrefs }
func (s *Store) Settings() domain.SettingsRepository        { return s.settings }
func (s *Store) Users() domain.UserRepository               { return s.users }
func (s *Store) Sessions() domain.SessionRepository         { return s.sessions }

// compile-time assertion that Store satisfies the domain boundary.
var _ domain.Repository = (*Store)(nil)
