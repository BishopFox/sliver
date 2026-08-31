package transports

import (
	"io"
	"sync"
)

// Tunnel - Duplex byte read/write
type Tunnel struct {
	ID uint64

	// Reader       io.ReadCloser
	Readers      []io.ReadCloser
	readSequence uint64

	Writer        io.WriteCloser
	writeSequence uint64

	mutex     *sync.RWMutex
	closeOnce sync.Once
}

func NewTunnel(id uint64, writer io.WriteCloser, readers ...io.ReadCloser) *Tunnel {
	return &Tunnel{
		ID:      id,
		Readers: readers,
		Writer:  writer,
		mutex:   &sync.RWMutex{},
	}
}

func (c *Tunnel) ReadSequence() uint64 {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.readSequence
}

func (c *Tunnel) WriteSequence() uint64 {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.writeSequence
}

// NextWriteSequence atomically reserves the next outbound sequence number.
// A shell can have independent stdout and stderr readers writing concurrently,
// so reading and incrementing the counter must be a single operation.
func (c *Tunnel) NextWriteSequence() uint64 {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	sequence := c.writeSequence
	c.writeSequence++
	return sequence
}

// IncReadSequence advances the expected inbound sequence number.
func (c *Tunnel) IncReadSequence() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.readSequence++
}

// Close - close tunnel reader and writer
func (c *Tunnel) Close() {
	c.closeOnce.Do(func() {
		for _, rc := range c.Readers {
			if rc != nil {
				_ = rc.Close()
			}
		}
		if c.Writer != nil {
			_ = c.Writer.Close()
		}
	})
}
