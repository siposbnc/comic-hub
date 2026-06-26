// Package version holds build metadata, injected at build time via -ldflags.
package version

var (
	// Version is the semantic version of the build.
	Version = "0.0.0-dev"
	// Commit is the git commit the binary was built from.
	Commit = "unknown"
	// Date is the build timestamp (RFC3339).
	Date = "unknown"
)
