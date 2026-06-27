// Package ulid generates ULIDs: 128-bit, lexicographically sortable, URL-safe
// identifiers (48-bit millisecond timestamp + 80 bits of randomness) encoded as a
// 26-character Crockford base32 string. They are the primary-key format for the
// catalog (see docs/02-data-model.md) — sortable by creation time, no coordination.
package ulid

import (
	"crypto/rand"
	"time"
)

// crockford is Crockford's base32 alphabet (no I, L, O, U).
const crockford = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// New returns a fresh ULID string. Safe for concurrent use.
func New() string {
	var id [16]byte

	ms := uint64(time.Now().UnixMilli())
	id[0] = byte(ms >> 40)
	id[1] = byte(ms >> 32)
	id[2] = byte(ms >> 24)
	id[3] = byte(ms >> 16)
	id[4] = byte(ms >> 8)
	id[5] = byte(ms)

	// crypto/rand is safe for concurrent use and never returns short reads.
	_, _ = rand.Read(id[6:])

	return encode(id)
}

// encode renders the 128-bit id as 26 Crockford base32 chars. The value is treated
// as a big-endian integer left-padded to 130 bits (26*5), so the leading char only
// carries the top 2 bits.
func encode(id [16]byte) string {
	out := make([]byte, 26)
	for i := range out {
		startBit := i*5 - 2 // first group has two virtual leading zero bits
		val := 0
		for b := 0; b < 5; b++ {
			bit := startBit + b
			if bit < 0 {
				continue // virtual leading zero
			}
			val = val<<1 | int((id[bit/8]>>(7-uint(bit%8)))&1)
		}
		out[i] = crockford[val]
	}
	return string(out)
}
