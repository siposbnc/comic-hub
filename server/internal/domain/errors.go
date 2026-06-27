package domain

import "errors"

// ErrNotFound is returned by repositories when a requested row does not exist. The
// transport layer maps it to HTTP 404.
var ErrNotFound = errors.New("not found")

// ErrValidation signals invalid input from a caller. The transport layer maps it to
// HTTP 400. Wrap it with a specific message via fmt.Errorf("%w: …", ErrValidation).
var ErrValidation = errors.New("validation failed")
