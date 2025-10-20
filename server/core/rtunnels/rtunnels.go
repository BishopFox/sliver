package rtunnels

import (
	"io"
	"sync"
)

var (
	Rtunnels map[uint64]*RTunnel = make(map[uint64]*RTunnel)
	mutex    sync.RWMutex
)

// RTunnel - Duplex byte read/write
type RTunnel struct {
	ID        uint64
	SessionID string
	// Reader       io.ReadCloser
	Readers      []io.ReadCloser
	readSequence uint64

	Writer        io.WriteCloser
	writeSequence uint64

	mutex *sync.RWMutex
}

func NewRTunnel(id uint64, sID string, writer io.WriteCloser, readers ...io.ReadCloser) *RTunnel {
	return &RTunnel{
		ID:        id,
		SessionID: sID,
		Readers:   readers,
		Writer:    writer,
		mutex:     &sync.RWMutex{},
	}
}

func (c *RTunnel) ReadSequence() uint64 {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.readSequence
}

func (c *RTunnel) WriteSequence() uint64 {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.writeSequence
}

func (c *RTunnel) IncReadSequence() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.readSequence += 1
}

func (c *RTunnel) IncWriteSequence() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.writeSequence += 1
}

// Close - close RTunnel reader and writer
func (c *RTunnel) Close() {
	for _, rc := range c.Readers {
		if rc != nil {
			rc.Close()
		}
	}
	c.Writer.Close()
}

// Tunnel - Add tunnel to mapping
func GetRTunnel(ID uint64) *RTunnel {
	mutex.RLock()
	defer mutex.RUnlock()
	return Rtunnels[ID]
}

// AddTunnel - Add tunnel to mapping
func AddRTunnel(tun *RTunnel) {
	mutex.Lock()
	defer mutex.Unlock()

	Rtunnels[tun.ID] = tun
}

// RemoveTunnel - Add tunnel to mapping
func RemoveRTunnel(ID uint64) {
	mutex.Lock()
	defer mutex.Unlock()

	delete(Rtunnels, ID)
}

// func removeAndCloseAllRTunnels() {
// 	mutex.Lock()
// 	defer mutex.Unlock()

// 	for id, tunnel := range Rtunnels {
// 		tunnel.Close()

// 		delete(Rtunnels, id)
// 	}
// }

// func (c *Connection) RequestResendR(data []byte) {
// 	c.Send <- &pb.Envelope{
// 		Type: pb.MsgTunnelData,
// 		Data: data,
// 	}
// }
