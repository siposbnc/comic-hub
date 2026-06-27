// Package contenthash computes stable content hashes of comic files for dedup and
// reader<->server progress reconciliation (docs/02-data-model.md §5). It uses xxhash64
// (fast, non-cryptographic) and a sampled mode for very large files.
package contenthash

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/cespare/xxhash/v2"
)

// sampleChunk is how many bytes are read from each sampled region of a large file.
const sampleChunk = 1 << 20 // 1 MiB

// OfFile returns the hex content hash of the file at path. Files at or under
// largeThreshold bytes are hashed in full; larger files use a sampled hash (the file
// size plus head/middle/tail chunks), which is far cheaper and still effectively unique
// for change detection. A largeThreshold <= 0 always hashes in full.
func OfFile(path string, largeThreshold int64) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return "", err
	}
	size := fi.Size()

	h := xxhash.New()
	if largeThreshold <= 0 || size <= largeThreshold {
		if _, err := io.Copy(h, f); err != nil {
			return "", err
		}
		return format(h.Sum64()), nil
	}

	// Sampled: mix in the size so files differing only in length still differ, then
	// fold in three regions.
	var sizeBuf [8]byte
	binary.LittleEndian.PutUint64(sizeBuf[:], uint64(size))
	_, _ = h.Write(sizeBuf[:])

	for _, off := range []int64{0, size/2 - sampleChunk/2, size - sampleChunk} {
		if off < 0 {
			off = 0
		}
		if _, err := f.Seek(off, io.SeekStart); err != nil {
			return "", err
		}
		if _, err := io.CopyN(h, f, sampleChunk); err != nil && err != io.EOF {
			return "", err
		}
	}
	return format(h.Sum64()), nil
}

func format(sum uint64) string { return fmt.Sprintf("%016x", sum) }
