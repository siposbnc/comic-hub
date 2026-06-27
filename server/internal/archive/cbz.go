package archive

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"path"
	"sort"

	"github.com/siposbnc/comic-hub/server/internal/pkg/natsort"
)

// CBZ reads ZIP-based comic archives (.cbz, and plain .zip). ZIP supports random
// access, so pages are served directly from the central directory with no full-archive
// scan.
type CBZ struct{}

// Extensions reports the file extensions this reader handles.
func (CBZ) Extensions() []string { return []string{"cbz", "zip"} }

// Open opens a CBZ file, enumerating and naturally sorting its image entries.
func (CBZ) Open(filePath string) (PageSource, error) {
	zr, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("open cbz: %w", err)
	}

	src := &zipSource{zr: zr}
	var total uint64
	for _, f := range zr.File {
		if f.FileInfo().IsDir() || isUnsafeEntry(f.Name) {
			continue
		}
		switch {
		case isImageEntry(f.Name):
			if f.UncompressedSize64 > maxEntryBytes {
				_ = zr.Close()
				return nil, fmt.Errorf("cbz: entry %q exceeds size limit", f.Name)
			}
			total += f.UncompressedSize64
			src.pages = append(src.pages, f)
		case src.sidecar == nil && isSidecarEntry(f.Name):
			src.sidecar = f
		}
	}

	if len(src.pages) > maxEntries {
		_ = zr.Close()
		return nil, fmt.Errorf("cbz: too many entries (%d)", len(src.pages))
	}
	if total > maxTotalBytes {
		_ = zr.Close()
		return nil, fmt.Errorf("cbz: total uncompressed size exceeds limit")
	}

	sort.SliceStable(src.pages, func(i, j int) bool {
		return natsort.Less(src.pages[i].Name, src.pages[j].Name)
	})
	return src, nil
}

// zipSource is an opened CBZ. The underlying *zip.ReadCloser is held until Close.
type zipSource struct {
	zr      *zip.ReadCloser
	pages   []*zip.File
	sidecar *zip.File
}

func (s *zipSource) PageCount() int { return len(s.pages) }

func (s *zipSource) Pages() []PageInfo {
	out := make([]PageInfo, len(s.pages))
	for i, f := range s.pages {
		out[i] = PageInfo{Index: i, FileName: path.Base(f.Name), Size: int64(f.UncompressedSize64)}
	}
	return out
}

func (s *zipSource) Page(i int) (io.ReadCloser, PageInfo, error) {
	if i < 0 || i >= len(s.pages) {
		return nil, PageInfo{}, fmt.Errorf("page %d out of range [0,%d)", i, len(s.pages))
	}
	f := s.pages[i]
	rc, err := f.Open()
	if err != nil {
		return nil, PageInfo{}, fmt.Errorf("open page %d: %w", i, err)
	}
	info := PageInfo{
		Index:    i,
		FileName: path.Base(f.Name),
		Size:     int64(f.UncompressedSize64),
	}
	return rc, info, nil
}

// Sidecar returns the ComicInfo.xml contents. It is read fully into memory (the file
// is tiny) so the caller gets a self-contained reader independent of the archive.
func (s *zipSource) Sidecar() (io.Reader, bool) {
	if s.sidecar == nil {
		return nil, false
	}
	rc, err := s.sidecar.Open()
	if err != nil {
		return nil, false
	}
	defer rc.Close()
	data, err := io.ReadAll(io.LimitReader(rc, maxEntryBytes))
	if err != nil {
		return nil, false
	}
	return bytes.NewReader(data), true
}

func (s *zipSource) Close() error { return s.zr.Close() }
