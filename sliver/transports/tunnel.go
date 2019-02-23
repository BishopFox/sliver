package transports

import (
	"io"
)

// Tunnel - Duplex byte read/write
type Tunnel struct {
	ID     uint64
	Reader io.ReadCloser
	Writer io.WriteCloser
}
