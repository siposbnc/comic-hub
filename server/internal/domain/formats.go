package domain

import "strings"

// IsSupportedFormat reports whether ext (with or without a leading dot, any case) is a
// supported comic format. The canonical lists (SupportedFormats, ArchiveFormats) live in
// formats_gen.go, generated from tools/codegen/formats.json.
func IsSupportedFormat(ext string) bool {
	e := strings.ToLower(strings.TrimPrefix(ext, "."))
	for _, f := range SupportedFormats {
		if f == e {
			return true
		}
	}
	return false
}
