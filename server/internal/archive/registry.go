package archive

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Registry dispatches a file path to the Reader registered for its extension. It is
// the single entry point the scanner and reader use to open any supported format.
type Registry struct {
	byExt map[string]Reader
}

// NewRegistry builds a registry from the given readers, indexing each by the
// extensions it declares. Later readers override earlier ones for the same extension.
func NewRegistry(readers ...Reader) *Registry {
	r := &Registry{byExt: make(map[string]Reader)}
	for _, rd := range readers {
		for _, ext := range rd.Extensions() {
			r.byExt[strings.ToLower(ext)] = rd
		}
	}
	return r
}

// DefaultRegistry returns a registry with all built-in format readers.
func DefaultRegistry() *Registry {
	return NewRegistry(CBZ{}, CBR{}, CB7{}, CBT{})
}

// Supports reports whether a reader is registered for the file's extension.
func (r *Registry) Supports(filePath string) bool {
	_, ok := r.byExt[ext(filePath)]
	return ok
}

// Open opens a comic file using the reader registered for its extension.
func (r *Registry) Open(filePath string) (PageSource, error) {
	rd, ok := r.byExt[ext(filePath)]
	if !ok {
		return nil, fmt.Errorf("unsupported comic format %q", filepath.Ext(filePath))
	}
	return rd.Open(filePath)
}

func ext(filePath string) string {
	return strings.ToLower(strings.TrimPrefix(filepath.Ext(filePath), "."))
}
