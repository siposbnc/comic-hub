package archive

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"

	"github.com/nwaples/rardecode/v2"

	"github.com/siposbnc/comic-hub/server/internal/pkg/natsort"
)

// CBR reads RAR-based comic archives (.cbr, and plain .rar). RAR is read-only and
// sequential: there is no central directory for random access, so Open enumerates the
// entries once and Page re-scans to the requested entry. Repeated access is masked by
// the server's page cache (see docs/04-server.md §5); we never write RAR.
type CBR struct{}

// Extensions reports the file extensions this reader handles.
func (CBR) Extensions() []string { return []string{"cbr", "rar"} }

type cbrEntry struct {
	name string
	size int64
}

// rarSource is an opened CBR. It holds only the file path and a sorted entry index;
// page bytes are streamed on demand.
type rarSource struct {
	path    string
	pages   []cbrEntry
	sidecar []byte
}

// Open enumerates a CBR's image entries (one sequential pass), capturing the
// ComicInfo.xml sidecar along the way, then sorts the pages into reading order.
func (CBR) Open(filePath string) (PageSource, error) {
	rr, err := rardecode.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("open cbr: %w", err)
	}
	defer rr.Close()

	src := &rarSource{path: filePath}
	var total int64
	for {
		hdr, err := rr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("cbr: read header: %w", err)
		}
		if hdr.IsDir || isUnsafeEntry(hdr.Name) {
			continue
		}
		switch {
		case isImageEntry(hdr.Name):
			if hdr.UnPackedSize > maxEntryBytes {
				return nil, fmt.Errorf("cbr: entry %q exceeds size limit", hdr.Name)
			}
			total += hdr.UnPackedSize
			src.pages = append(src.pages, cbrEntry{name: hdr.Name, size: hdr.UnPackedSize})
		case src.sidecar == nil && isSidecarEntry(hdr.Name):
			data, err := io.ReadAll(io.LimitReader(rr, maxEntryBytes))
			if err != nil {
				return nil, fmt.Errorf("cbr: read sidecar: %w", err)
			}
			src.sidecar = data
		}
	}

	if len(src.pages) > maxEntries {
		return nil, fmt.Errorf("cbr: too many entries (%d)", len(src.pages))
	}
	if total > maxTotalBytes {
		return nil, fmt.Errorf("cbr: total uncompressed size exceeds limit")
	}

	sort.SliceStable(src.pages, func(i, j int) bool {
		return natsort.Less(src.pages[i].name, src.pages[j].name)
	})
	return src, nil
}

func (s *rarSource) PageCount() int { return len(s.pages) }

func (s *rarSource) Page(i int) (io.ReadCloser, PageInfo, error) {
	if i < 0 || i >= len(s.pages) {
		return nil, PageInfo{}, fmt.Errorf("page %d out of range [0,%d)", i, len(s.pages))
	}
	want := s.pages[i].name

	rr, err := rardecode.OpenReader(s.path)
	if err != nil {
		return nil, PageInfo{}, fmt.Errorf("reopen cbr: %w", err)
	}
	defer rr.Close()

	for {
		hdr, err := rr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, PageInfo{}, fmt.Errorf("cbr: scan to page %d: %w", i, err)
		}
		if hdr.Name != want {
			continue
		}
		// Read fully into memory so the returned reader is independent of rr, which we
		// close on return.
		data, err := io.ReadAll(io.LimitReader(rr, maxEntryBytes))
		if err != nil {
			return nil, PageInfo{}, fmt.Errorf("cbr: read page %d: %w", i, err)
		}
		info := PageInfo{Index: i, FileName: path.Base(want), Size: int64(len(data))}
		return io.NopCloser(bytes.NewReader(data)), info, nil
	}
	return nil, PageInfo{}, fmt.Errorf("cbr: page %d (%q) not found", i, want)
}

func (s *rarSource) Sidecar() (io.Reader, bool) {
	if s.sidecar == nil {
		return nil, false
	}
	return bytes.NewReader(s.sidecar), true
}

// Close is a no-op: rarSource holds no open handle between page reads.
func (s *rarSource) Close() error { return nil }
