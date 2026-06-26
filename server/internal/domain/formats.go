package domain

import "strings"

// SupportedFormats is the canonical set of comic formats ComicHub reads — the single
// source of truth on the server. Keep in sync with the spec (docs/00-overview.md §6),
// the reader's Rust formats.rs, and the frontend's @comichub/reader-core
// SUPPORTED_FORMATS. Don't hardcode format lists elsewhere.
var SupportedFormats = []string{"cbz", "cbr", "cb7", "cbt", "pdf"}

// ArchiveFormats are the container-based formats (everything except PDF). PDF is read
// via rasterization rather than archive extraction.
var ArchiveFormats = []string{"cbz", "cbr", "cb7", "cbt"}

// IsSupportedFormat reports whether ext (with or without a leading dot, any case) is a
// supported comic format.
func IsSupportedFormat(ext string) bool {
	e := strings.ToLower(strings.TrimPrefix(ext, "."))
	for _, f := range SupportedFormats {
		if f == e {
			return true
		}
	}
	return false
}
