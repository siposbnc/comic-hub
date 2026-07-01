// Package discovery advertises the server on the local network over mDNS/DNS-SD
// (Bonjour), so clients can find it without typing a URL (Phase 3 — Milestone D).
// Server mode only: an embedded sidecar is loopback-bound and private to its client,
// so it never advertises.
package discovery

import (
	"fmt"

	"github.com/libp2p/zeroconf/v2"
)

// ServiceType is the DNS-SD service type clients browse for.
const ServiceType = "_comichub._tcp"

// domain is the standard link-local mDNS domain.
const domain = "local."

// Info is what the advertisement carries. Everything here is public to the LAN by
// design — keep it to what the connect screen needs to render a server row.
type Info struct {
	// Instance is the human-readable server name shown in the client's discovery
	// list (DNS-SD instance name, e.g. the machine's hostname).
	Instance string
	// Port is the TCP port the HTTP API listens on.
	Port int
	// Version is the server version (TXT `version`).
	Version string
	// AuthRequired reports whether the server requires login (TXT `auth`), so the
	// client can route straight to the right screen after picking a server.
	AuthRequired bool
}

// TXT renders the advertisement's TXT key/value records.
func (i Info) TXT() []string {
	return []string{
		fmt.Sprintf("version=%s", i.Version),
		fmt.Sprintf("auth=%t", i.AuthRequired),
	}
}

// Advertiser keeps the mDNS registration alive until closed.
type Advertiser struct {
	server *zeroconf.Server
}

// Advertise registers the service on all multicast-capable interfaces and answers
// queries until Close. The instance name must be non-empty.
func Advertise(info Info) (*Advertiser, error) {
	if info.Instance == "" {
		return nil, fmt.Errorf("mdns: instance name is empty")
	}
	srv, err := zeroconf.Register(info.Instance, ServiceType, domain, info.Port, info.TXT(), nil)
	if err != nil {
		return nil, fmt.Errorf("mdns register: %w", err)
	}
	return &Advertiser{server: srv}, nil
}

// Close withdraws the advertisement (sends a goodbye packet) and stops responding.
func (a *Advertiser) Close() {
	a.server.Shutdown()
}
