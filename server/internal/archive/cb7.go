package archive

import (
	"bytes"
	"fmt"
	"io"
	"path"
	"sort"

	"github.com/bodgit/sevenzip"

	"github.com/siposbnc/comic-hub/server/internal/pkg/natsort"
)

// CB7 reads 7-Zip-based comic archives (.cb7, and plain .7z). Like ZIP, 7z carries a
// directory of its entries, so pages are served on demand without a full-archive re-scan.
// We never write 7z.
type CB7 struct{}

// Extensions reports the file extensions this reader handles.
func (CB7) Extensions() []string { return []string{"cb7", "7z"} }

// Open opens a CB7 file, enumerating and naturally sorting its image entries.
func (CB7) Open(filePath string) (PageSource, error) {
	zr, err := sevenzip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("open cb7: %w", err)
	}

	src := &sevenZipSource{zr: zr}
	var total int64
	for _, f := range zr.File {
		if f.FileInfo().IsDir() || isUnsafeEntry(f.Name) {
			continue
		}
		size := f.FileInfo().Size()
		switch {
		case isImageEntry(f.Name):
			if size > maxEntryBytes {
				_ = zr.Close()
				return nil, fmt.Errorf("cb7: entry %q exceeds size limit", f.Name)
			}
			total += size
			src.pages = append(src.pages, f)
		case src.sidecar == nil && isSidecarEntry(f.Name):
			src.sidecar = f
		}
	}

	if len(src.pages) > maxEntries {
		_ = zr.Close()
		return nil, fmt.Errorf("cb7: too many entries (%d)", len(src.pages))
	}
	if total > maxTotalBytes {
		_ = zr.Close()
		return nil, fmt.Errorf("cb7: total uncompressed size exceeds limit")
	}

	sort.SliceStable(src.pages, func(i, j int) bool {
		return natsort.Less(src.pages[i].Name, src.pages[j].Name)
	})
	return src, nil
}

// sevenZipSource is an opened CB7. The underlying *sevenzip.ReadCloser is held until Close.
type sevenZipSource struct {
	zr      *sevenzip.ReadCloser
	pages   []*sevenzip.File
	sidecar *sevenzip.File
}

func (s *sevenZipSource) PageCount() int { return len(s.pages) }

func (s *sevenZipSource) Pages() []PageInfo {
	out := make([]PageInfo, len(s.pages))
	for i, f := range s.pages {
		out[i] = PageInfo{Index: i, FileName: path.Base(f.Name), Size: f.FileInfo().Size()}
	}
	return out
}

func (s *sevenZipSource) Page(i int) (io.ReadCloser, PageInfo, error) {
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
		Size:     f.FileInfo().Size(),
	}
	return rc, info, nil
}

// Sidecar returns the ComicInfo.xml contents, read fully into memory (the file is tiny) so
// the caller gets a self-contained reader independent of the archive.
func (s *sevenZipSource) Sidecar() (io.Reader, bool) {
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

func (s *sevenZipSource) Close() error { return s.zr.Close() }
