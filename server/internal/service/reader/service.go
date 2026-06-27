// Package reader serves the book-reading surface: the page manifest and page/cover/
// thumbnail images. It opens archives via the format registry, resizes/transcodes via
// the image processor, and caches derived images on disk plus recent original page
// bytes in memory. See docs/03-api.md §5 and docs/04-server.md §5.
package reader

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/siposbnc/comic-hub/server/internal/archive"
	"github.com/siposbnc/comic-hub/server/internal/domain"
	"github.com/siposbnc/comic-hub/server/internal/image"
)

const (
	defaultCoverWidth = 300
	thumbWidth        = 160
	rawPageCacheSize  = 256
)

// Image is a rendered image ready to serve.
type Image struct {
	Data        []byte
	ContentType string
	ETag        string
}

// PageEntry is one page in a manifest.
type PageEntry struct {
	Idx    int    `json:"idx"`
	W      int    `json:"w"`
	H      int    `json:"h"`
	Type   string `json:"type,omitempty"`
	Double bool   `json:"double,omitempty"`
}

// Manifest is the reader's source of truth for a book (docs/03-api.md §5).
type Manifest struct {
	BookID     string      `json:"bookId"`
	PageCount  int         `json:"pageCount"`
	ReadingDir string      `json:"readingDir"`
	Pages      []PageEntry `json:"pages"`
}

// PageOptions controls a page render. The zero value means "original bytes, untouched".
type PageOptions struct {
	Width   int
	Format  image.Format
	Quality int
}

type rawPage struct {
	data        []byte
	contentType string
}

// Service renders manifests and images for books.
type Service struct {
	repo     domain.Repository
	registry *archive.Registry
	proc     image.Processor
	derived  *image.DiskCache
	raw      *lru.Cache[string, rawPage]
}

// New constructs the reader service.
func New(repo domain.Repository, registry *archive.Registry, proc image.Processor, derived *image.DiskCache) (*Service, error) {
	raw, err := lru.New[string, rawPage](rawPageCacheSize)
	if err != nil {
		return nil, err
	}
	return &Service{repo: repo, registry: registry, proc: proc, derived: derived, raw: raw}, nil
}

// Manifest returns the ordered page list + reading direction for a book.
func (s *Service) Manifest(ctx context.Context, bookID string) (Manifest, error) {
	b, err := s.repo.Books().Get(ctx, bookID)
	if err != nil {
		return Manifest{}, err
	}
	pages, err := s.repo.Books().ListPages(ctx, bookID)
	if err != nil {
		return Manifest{}, err
	}

	readingDir := string(domain.LTR)
	if ser, err := s.repo.Series().Get(ctx, b.SeriesID); err == nil && ser.ReadingDir != "" {
		readingDir = string(ser.ReadingDir)
	}

	m := Manifest{BookID: b.ID, PageCount: b.PageCount, ReadingDir: readingDir, Pages: make([]PageEntry, 0, len(pages))}
	for _, p := range pages {
		m.Pages = append(m.Pages, PageEntry{Idx: p.Index, W: p.Width, H: p.Height, Type: p.PageType, Double: p.IsDouble})
	}
	return m, nil
}

// Page renders page idx of a book. With the zero PageOptions it streams the original
// bytes; otherwise it resizes/transcodes (cached on disk, content-addressed).
func (s *Service) Page(ctx context.Context, bookID string, idx int, opts PageOptions) (Image, error) {
	b, err := s.repo.Books().Get(ctx, bookID)
	if err != nil {
		return Image{}, err
	}
	if b.IsCorrupt {
		return Image{}, domain.ErrNotFound
	}
	return s.render(b, idx, opts)
}

// Cover renders a book's cover page resized to width (default 300) as JPEG.
func (s *Service) Cover(ctx context.Context, bookID string, width int) (Image, error) {
	b, err := s.repo.Books().Get(ctx, bookID)
	if err != nil {
		return Image{}, err
	}
	if b.IsCorrupt {
		return Image{}, domain.ErrNotFound
	}
	if width <= 0 {
		width = defaultCoverWidth
	}
	return s.render(b, b.CoverPage, PageOptions{Width: width, Format: image.FormatJPEG})
}

// Thumb renders a small page thumbnail (scrubber strip).
func (s *Service) Thumb(ctx context.Context, bookID string, idx int) (Image, error) {
	b, err := s.repo.Books().Get(ctx, bookID)
	if err != nil {
		return Image{}, err
	}
	if b.IsCorrupt {
		return Image{}, domain.ErrNotFound
	}
	return s.render(b, idx, PageOptions{Width: thumbWidth, Format: image.FormatJPEG})
}

// Prefetch best-effort warms the original-bytes cache for pages [from, from+count).
func (s *Service) Prefetch(ctx context.Context, bookID string, from, count int) {
	b, err := s.repo.Books().Get(ctx, bookID)
	if err != nil || b.IsCorrupt {
		return
	}
	for i := from; i < from+count && i < b.PageCount; i++ {
		if i < 0 {
			continue
		}
		_, _ = s.rawPage(b, i)
	}
}

func (s *Service) render(b domain.Book, idx int, opts PageOptions) (Image, error) {
	// Original passthrough — no decode, just stream the archived bytes.
	if opts.Width == 0 && opts.Format == "" {
		rp, err := s.rawPage(b, idx)
		if err != nil {
			return Image{}, err
		}
		return Image{Data: rp.data, ContentType: rp.contentType, ETag: quote(s.base(b) + "|p" + strconv.Itoa(idx) + "|orig")}, nil
	}

	key := s.derivedKey(b, idx, opts)
	if data, ok := s.derived.Get(key); ok {
		return Image{Data: data, ContentType: contentTypeForFormat(opts.Format), ETag: quote(key)}, nil
	}

	rp, err := s.rawPage(b, idx)
	if err != nil {
		return Image{}, err
	}
	res, err := s.proc.Resize(rp.data, image.Options{Width: opts.Width, Format: opts.Format, Quality: opts.Quality})
	if err != nil {
		return Image{}, err
	}
	_ = s.derived.Put(key, res.Data)
	return Image{Data: res.Data, ContentType: res.ContentType, ETag: quote(key)}, nil
}

// rawPage returns the original bytes of a page, caching recent reads in memory (which
// also masks the per-page re-scan cost of sequential formats like CBR).
func (s *Service) rawPage(b domain.Book, idx int) (rawPage, error) {
	cacheKey := b.ID + "|" + strconv.Itoa(idx)
	if v, ok := s.raw.Get(cacheKey); ok {
		return v, nil
	}
	src, err := s.registry.Open(b.FilePath)
	if err != nil {
		return rawPage{}, err
	}
	defer src.Close()

	rc, info, err := src.Page(idx)
	if err != nil {
		return rawPage{}, err
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return rawPage{}, err
	}
	rp := rawPage{data: data, contentType: contentTypeFor(info.FileName)}
	s.raw.Add(cacheKey, rp)
	return rp, nil
}

func (s *Service) base(b domain.Book) string {
	if b.ContentHash != "" {
		return b.ContentHash
	}
	return b.ID
}

func (s *Service) derivedKey(b domain.Book, idx int, opts PageOptions) string {
	f := opts.Format
	if f == "" {
		f = image.FormatJPEG
	}
	q := opts.Quality
	if q == 0 {
		q = image.DefaultQuality
	}
	return fmt.Sprintf("%s|p%d|w%d|f%s|q%d", s.base(b), idx, opts.Width, f, q)
}

func quote(s string) string { return "\"" + s + "\"" }

func contentTypeForFormat(f image.Format) string {
	if f == image.FormatPNG {
		return "image/png"
	}
	return "image/jpeg"
}

func contentTypeFor(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".avif":
		return "image/avif"
	case ".bmp":
		return "image/bmp"
	case ".tif", ".tiff":
		return "image/tiff"
	default:
		return "application/octet-stream"
	}
}
