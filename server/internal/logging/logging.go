// Package logging configures the application's structured logger.
package logging

import (
	"log/slog"
	"os"
	"strings"
)

// New returns a slog.Logger writing to stderr. format is "json" (default) or "text";
// level is one of debug|info|warn|error.
func New(level, format string) *slog.Logger {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(strings.ToLower(level))); err != nil {
		lvl = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: lvl}

	var h slog.Handler
	if strings.EqualFold(format, "text") {
		h = slog.NewTextHandler(os.Stderr, opts)
	} else {
		h = slog.NewJSONHandler(os.Stderr, opts)
	}
	return slog.New(h)
}
