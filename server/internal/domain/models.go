// Package domain holds the core entities and behaviour-defining interfaces for the
// catalog. It depends on nothing else in the server; adapters (store, archive,
// providers) depend on it. Models here mirror docs/02-data-model.md.
package domain

// ReadingDirection is the page-turn direction for a series/book.
type ReadingDirection string

const (
	LTR ReadingDirection = "ltr"
	RTL ReadingDirection = "rtl"
)

// MetadataState tracks where a book's metadata came from, governing scraper precedence.
type MetadataState string

const (
	MetaNone    MetadataState = "none"    // filename heuristics only
	MetaSidecar MetadataState = "sidecar" // from ComicInfo.xml
	MetaMatched MetadataState = "matched" // from an online provider
	MetaLocked  MetadataState = "locked"  // user-edited; never overwritten
)

// ReadStatus is a user's reading state for a book.
type ReadStatus string

const (
	StatusUnread     ReadStatus = "unread"
	StatusInProgress ReadStatus = "in_progress"
	StatusRead       ReadStatus = "read"
)

// Library is a named set of root folders ComicHub scans.
type Library struct {
	ID        string
	Name      string
	Kind      string // comic | manga
	Roots     []string
	CreatedAt int64
	UpdatedAt int64
}

// Series groups issues that belong together.
type Series struct {
	ID          string
	LibraryID   string
	FolderPath  string
	Name        string
	SortName    string
	Year        int
	Publisher   string
	Description  string
	ReadingDir  ReadingDirection
	CoverBookID string
	CreatedAt   int64
	UpdatedAt   int64
}

// Book is a single comic file — the atomic readable unit.
type Book struct {
	ID            string
	SeriesID      string
	LibraryID     string
	FilePath      string
	FileFormat    string // one of domain.SupportedFormats
	FileSize      int64
	FileMTime     int64
	ContentHash   string
	PageCount     int
	Title         string
	Number        string
	SortNumber    float64
	Volume        int
	ReleaseDate   int64
	AgeRating     string
	Language      string
	Summary       string
	CoverPage     int
	MetadataState MetadataState
	IsCorrupt     bool
	AddedAt       int64
	UpdatedAt     int64
}

// Page is one image within a book.
type Page struct {
	BookID   string
	Index    int
	FileName string
	Width    int
	Height   int
	Size     int64
	PageType string
	IsDouble bool
}

// Progress is a user's reading state for a single book.
type Progress struct {
	UserID     string
	BookID     string
	Page       int
	PageCount  int
	Status     ReadStatus
	StartedAt  int64
	FinishedAt int64
	UpdatedAt  int64
	Device     string
}
