// Package password hashes and verifies user passwords with argon2id, the memory-hard
// algorithm recommended for password storage. Hashes are self-describing (they encode the
// parameters + salt) so the cost can be raised later without invalidating old hashes.
package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Tunable argon2id parameters. Defaults follow current OWASP guidance (19 MiB, 2 passes).
// Encoded into each hash so a future bump doesn't break verification of existing hashes.
const (
	argonTime    = 2
	argonMemory  = 19 * 1024 // KiB
	argonThreads = 1
	argonKeyLen  = 32
	saltLen      = 16
)

// ErrMismatch is returned when a password does not match the hash.
var ErrMismatch = errors.New("password: hash mismatch")

// Hash returns an argon2id PHC-style string ("$argon2id$v=19$m=...,t=...,p=...$salt$hash").
func Hash(plaintext string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("password: read salt: %w", err)
	}
	key := argon2.IDKey([]byte(plaintext), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// Verify reports whether plaintext matches the encoded argon2id hash. It returns ErrMismatch
// on a wrong password and a different error if the hash is malformed.
func Verify(plaintext, encoded string) error {
	params, salt, want, err := decode(encoded)
	if err != nil {
		return err
	}
	got := argon2.IDKey([]byte(plaintext), salt, params.time, params.memory, params.threads, uint32(len(want)))
	if subtle.ConstantTimeCompare(got, want) != 1 {
		return ErrMismatch
	}
	return nil
}

type params struct {
	memory  uint32
	time    uint32
	threads uint8
}

func decode(encoded string) (params, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	// ["", "argon2id", "v=19", "m=...,t=...,p=...", salt, hash]
	if len(parts) != 6 || parts[1] != "argon2id" {
		return params{}, nil, nil, errors.New("password: invalid hash format")
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return params{}, nil, nil, errors.New("password: unsupported argon2 version")
	}
	var p params
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.memory, &p.time, &p.threads); err != nil {
		return params{}, nil, nil, errors.New("password: invalid hash parameters")
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return params{}, nil, nil, errors.New("password: invalid salt encoding")
	}
	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return params{}, nil, nil, errors.New("password: invalid hash encoding")
	}
	return p, salt, hash, nil
}
