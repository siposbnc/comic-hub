// Package connection handles the embedded-mode handshake: the server publishes the
// loopback port + token to a small file that the client reads to connect. See
// docs/01-architecture.md §3.
package connection

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Handshake is the connection descriptor written by an embedded server and read by
// the client that spawned it.
type Handshake struct {
	Port    int    `json:"port"`
	Token   string `json:"token"`
	PID     int    `json:"pid"`
	Version string `json:"version"`
	BaseURL string `json:"baseUrl"`
}

// Write atomically writes the handshake to path with owner-only permissions.
func Write(path string, hs Handshake) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(hs, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename handshake into place: %w", err)
	}
	return nil
}

// Read loads a handshake from path.
func Read(path string) (Handshake, error) {
	var hs Handshake
	data, err := os.ReadFile(path)
	if err != nil {
		return hs, err
	}
	if err := json.Unmarshal(data, &hs); err != nil {
		return hs, err
	}
	return hs, nil
}

// GenerateToken returns a 256-bit hex token suitable for loopback auth.
func GenerateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
