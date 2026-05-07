package rtunnels

import (
	"io"
	"sync"
)

var (
	Rtunnels  map[uint64]*RTunnel = make(map[uint64]*RTunnel)
	mutex     sync.RWMutex
	pending   map[string]map[string]int
	listeners map[string]map[uint32]string
	pendingMu sync.Mutex
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

func AddPending(sessionID string, connStr string) {
	pendingMu.Lock()
	defer pendingMu.Unlock()
	if pending == nil {
		pending = make(map[string]map[string]int)
	}
	addrMap := pending[sessionID]
	if addrMap == nil {
		addrMap = make(map[string]int)
		pending[sessionID] = addrMap
	}
	addrMap[connStr]++
}

func DeletePending(sessionID string, connStr string) {
	pendingMu.Lock()
	defer pendingMu.Unlock()
	if pending == nil {
		return
	}
	if addrMap, ok := pending[sessionID]; ok {
		if count, ok := addrMap[connStr]; ok {
			if count <= 1 {
				delete(addrMap, connStr)
			} else {
				addrMap[connStr] = count - 1
			}
		}
		if len(addrMap) == 0 {
			delete(pending, sessionID)
		}
	}
}

func Check(sessionID string, connStr string) bool {
	pendingMu.Lock()
	defer pendingMu.Unlock()
	if pending == nil {
		return false
	}
	if addrMap, ok := pending[sessionID]; ok {
		return addrMap[connStr] > 0
	}
	return false
}

func TrackListener(sessionID string, listenerID uint32, connStr string) {
	pendingMu.Lock()
	defer pendingMu.Unlock()
	if listeners == nil {
		listeners = make(map[string]map[uint32]string)
	}
	if pending == nil {
		pending = make(map[string]map[string]int)
	}
	lm := listeners[sessionID]
	if lm == nil {
		lm = make(map[uint32]string)
		listeners[sessionID] = lm
	}
	lm[listenerID] = connStr
	addrMap := pending[sessionID]
	if addrMap == nil {
		addrMap = make(map[string]int)
		pending[sessionID] = addrMap
	}
	addrMap[connStr]++
}

func UntrackListener(sessionID string, listenerID uint32) bool {
	pendingMu.Lock()
	defer pendingMu.Unlock()
	if listeners == nil || pending == nil {
		return false
	}
	lm, ok := listeners[sessionID]
	if !ok {
		return false
	}
	connStr, ok := lm[listenerID]
	if !ok {
		return false
	}
	delete(lm, listenerID)
	if len(lm) == 0 {
		delete(listeners, sessionID)
	}
	if addrMap, ok := pending[sessionID]; ok {
		if count, ok := addrMap[connStr]; ok {
			if count <= 1 {
				delete(addrMap, connStr)
			} else {
				addrMap[connStr] = count - 1
			}
		}
		if len(addrMap) == 0 {
			delete(pending, sessionID)
		}
	}
	return true
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
