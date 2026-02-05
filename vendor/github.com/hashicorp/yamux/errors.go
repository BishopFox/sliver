package yamux

import "errors"

var (
	// ErrSessionShutdown is returned when the session is closed.
	ErrSessionShutdown = errors.New("yamux: session shutdown")
)
