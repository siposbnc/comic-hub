package archive

import (
	"path"
	"strings"
)

// Extraction limits guard against malicious or accidental "zip bombs" — an archive
// that is small on disk but enormous when expanded. They are generous enough never to
// reject a legitimate comic. See docs/01-architecture.md §8 and docs/04-server.md §3.3.
const (
	// maxEntries caps the number of files in an archive.
	maxEntries = 20000
	// maxEntryBytes caps a single page's uncompressed size (512 MiB).
	maxEntryBytes = 512 << 20
	// maxTotalBytes caps the total uncompressed size of all pages (8 GiB).
	maxTotalBytes = 8 << 30
)

// imageExts is the set of page image formats found inside comic archives
// (see docs/00-overview.md §6).
var imageExts = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".webp": true,
	".avif": true,
	".gif":  true,
	".bmp":  true,
}

// isImageEntry reports whether an archive entry name is a page image.
func isImageEntry(name string) bool {
	return imageExts[strings.ToLower(path.Ext(name))]
}

// isSidecarEntry reports whether an entry is the ComicInfo.xml metadata sidecar.
func isSidecarEntry(name string) bool {
	return strings.EqualFold(path.Base(name), "ComicInfo.xml")
}

// isUnsafeEntry rejects entries that try to escape the archive root via traversal or
// absolute paths. Comic readers only ever use the base name, but we refuse such
// entries defensively.
func isUnsafeEntry(name string) bool {
	clean := path.Clean("/" + strings.ReplaceAll(name, "\\", "/"))
	return clean == "/" || strings.Contains(name, "..")
}
