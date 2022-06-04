package transports

import (
	"io"
	"sync"
)

// Tunnel - Duplex byte read/write
type Tunnel struct {
	ID uint64

	Reader       io.ReadCloser
	readSequence uint64

	Writer        io.WriteCloser
	writeSequence uint64

	mutex *sync.RWMutex
}

func NewTunnel(id uint64, reader io.ReadCloser, writer io.WriteCloser) *Tunnel {
	return &Tunnel{
		ID:     id,
		Reader: reader,
		Writer: writer,
		mutex:  &sync.RWMutex{},
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

func (c *Tunnel) IncReadSequence() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.readSequence += 1
}

func (c *Tunnel) IncWriteSequence() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.writeSequence += 1
}

// Close - close tunnel reader and writer
func (c *Tunnel) Close() {
	c.Reader.Close()
	c.Writer.Close()
}
