// Package archive defines the format-agnostic boundary for reading comic files. One
// implementation exists per container (CBZ/CBR/CB7/CBT/PDF); the rest of the server is
// format-agnostic. See docs/04-server.md §4. Implementations land in Phase 1.
package archive

import "io"

// PageInfo describes a single page extracted from an archive.
type PageInfo struct {
	Index    int
	FileName string
	Width    int
	Height   int
	Size     int64
	PageType string // ComicInfo page type, when known
	IsDouble bool
}

// PageSource is an opened comic file from which pages can be read on demand.
type PageSource interface {
	// PageCount returns the number of image pages.
	PageCount() int
	// Page returns the raw image bytes for page i along with its metadata.
	Page(i int) (io.ReadCloser, PageInfo, error)
	// Sidecar returns the embedded ComicInfo.xml if present.
	Sidecar() (io.Reader, bool)
	// Close releases the underlying file handle.
	Close() error
}

// Reader opens a comic file of a particular format into a PageSource.
type Reader interface {
	// Extensions returns the file extensions this reader handles (e.g. "cbz").
	Extensions() []string
	// Open opens the file at path.
	Open(path string) (PageSource, error)
}
