package image

import (
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
)

// DiskCache is a content-addressed on-disk cache for derived images (covers,
// thumbnails, resized pages). Entries are keyed by an opaque string (built from the
// source content hash + transform params) and sharded by a hash of that key. Contents
// are derived data — safe to delete; regenerated on demand (docs/04-server.md §5).
//
// Eviction is not yet implemented: thumbnails are cheap and high-reuse, so they persist.
// A size-capped LRU sweep is a later enhancement.
type DiskCache struct {
	root string
}

// NewDiskCache creates (if needed) the cache directory and returns a cache rooted there.
func NewDiskCache(root string) (*DiskCache, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create derived cache dir: %w", err)
	}
	return &DiskCache{root: root}, nil
}

// Get returns the cached bytes for key, if present.
func (c *DiskCache) Get(key string) ([]byte, bool) {
	data, err := os.ReadFile(c.path(key))
	if err != nil {
		return nil, false
	}
	return data, true
}

// Put writes bytes for key, creating the shard directory. Write is atomic (temp +
// rename) so a concurrent reader never sees a partial file.
func (c *DiskCache) Put(key string, data []byte) error {
	p := c.path(key)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, p)
}

func (c *DiskCache) path(key string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(key))
	sum := fmt.Sprintf("%016x", h.Sum64())
	return filepath.Join(c.root, sum[:2], sum)
}
