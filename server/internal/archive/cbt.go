package archive

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sort"

	"github.com/siposbnc/comic-hub/server/internal/pkg/natsort"
)

// CBT reads TAR-based comic archives (.cbt, and plain .tar). TAR is read-only and
// sequential: there is no directory for random access, so Open enumerates the entries once
// and Page re-scans to the requested entry. Repeated access is masked by the server's page
// cache (see docs/04-server.md §5); we never write TAR.
type CBT struct{}

// Extensions reports the file extensions this reader handles.
func (CBT) Extensions() []string { return []string{"cbt", "tar"} }

type cbtEntry struct {
	name string
	size int64
}

// tarSource is an opened CBT. It holds only the file path and a sorted entry index; page
// bytes are streamed on demand.
type tarSource struct {
	path    string
	pages   []cbtEntry
	sidecar []byte
}

// Open enumerates a CBT's image entries (one sequential pass), capturing the
// ComicInfo.xml sidecar along the way, then sorts the pages into reading order.
func (CBT) Open(filePath string) (PageSource, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open cbt: %w", err)
	}
	defer f.Close()

	tr := tar.NewReader(f)
	src := &tarSource{path: filePath}
	var total int64
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("cbt: read header: %w", err)
		}
		if hdr.FileInfo().IsDir() || isUnsafeEntry(hdr.Name) {
			continue
		}
		switch {
		case isImageEntry(hdr.Name):
			if hdr.Size > maxEntryBytes {
				return nil, fmt.Errorf("cbt: entry %q exceeds size limit", hdr.Name)
			}
			total += hdr.Size
			src.pages = append(src.pages, cbtEntry{name: hdr.Name, size: hdr.Size})
		case src.sidecar == nil && isSidecarEntry(hdr.Name):
			data, err := io.ReadAll(io.LimitReader(tr, maxEntryBytes))
			if err != nil {
				return nil, fmt.Errorf("cbt: read sidecar: %w", err)
			}
			src.sidecar = data
		}
	}

	if len(src.pages) > maxEntries {
		return nil, fmt.Errorf("cbt: too many entries (%d)", len(src.pages))
	}
	if total > maxTotalBytes {
		return nil, fmt.Errorf("cbt: total uncompressed size exceeds limit")
	}

	sort.SliceStable(src.pages, func(i, j int) bool {
		return natsort.Less(src.pages[i].name, src.pages[j].name)
	})
	return src, nil
}

func (s *tarSource) PageCount() int { return len(s.pages) }

func (s *tarSource) Pages() []PageInfo {
	out := make([]PageInfo, len(s.pages))
	for i, e := range s.pages {
		out[i] = PageInfo{Index: i, FileName: path.Base(e.name), Size: e.size}
	}
	return out
}

func (s *tarSource) Page(i int) (io.ReadCloser, PageInfo, error) {
	if i < 0 || i >= len(s.pages) {
		return nil, PageInfo{}, fmt.Errorf("page %d out of range [0,%d)", i, len(s.pages))
	}
	want := s.pages[i].name

	f, err := os.Open(s.path)
	if err != nil {
		return nil, PageInfo{}, fmt.Errorf("reopen cbt: %w", err)
	}
	defer f.Close()

	tr := tar.NewReader(f)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, PageInfo{}, fmt.Errorf("cbt: scan to page %d: %w", i, err)
		}
		if hdr.Name != want {
			continue
		}
		// Read fully into memory so the returned reader is independent of f, which we
		// close on return.
		data, err := io.ReadAll(io.LimitReader(tr, maxEntryBytes))
		if err != nil {
			return nil, PageInfo{}, fmt.Errorf("cbt: read page %d: %w", i, err)
		}
		info := PageInfo{Index: i, FileName: path.Base(want), Size: int64(len(data))}
		return io.NopCloser(bytes.NewReader(data)), info, nil
	}
	return nil, PageInfo{}, fmt.Errorf("cbt: page %d (%q) not found", i, want)
}

func (s *tarSource) Sidecar() (io.Reader, bool) {
	if s.sidecar == nil {
		return nil, false
	}
	return bytes.NewReader(s.sidecar), true
}

// Close is a no-op: tarSource holds no open handle between page reads.
func (s *tarSource) Close() error { return nil }
