package rtunnels

import (
	"io"
	"sync"
)

var (
	Rtunnels  map[uint64]*RTunnel = make(map[uint64]*RTunnel)
	mutex     sync.RWMutex
	pending   map[string]map[string]*pendingInfo
	listeners map[string]map[uint32]string
	pendingMu sync.Mutex
)

type pendingInfo struct {
	count     int
	keepAlive int32
}

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
		pending = make(map[string]map[string]*pendingInfo)
	}
	addrMap := pending[sessionID]
	if addrMap == nil {
		addrMap = make(map[string]*pendingInfo)
		pending[sessionID] = addrMap
	}
	if info, ok := addrMap[connStr]; ok {
		info.count++
	} else {
		addrMap[connStr] = &pendingInfo{count: 1}
	}
}

func DeletePending(sessionID string, connStr string) {
	pendingMu.Lock()
	defer pendingMu.Unlock()
	if pending == nil {
		return
	}
	if addrMap, ok := pending[sessionID]; ok {
		if info, ok := addrMap[connStr]; ok {
			if info.count <= 1 {
				delete(addrMap, connStr)
			} else {
				info.count--
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
		if info, ok := addrMap[connStr]; ok {
			return info.count > 0
		}
	}
	return false
}

func GetKeepAlive(sessionID string, connStr string) int32 {
	pendingMu.Lock()
	defer pendingMu.Unlock()
	if pending == nil {
		return 0
	}
	if addrMap, ok := pending[sessionID]; ok {
		if info, ok := addrMap[connStr]; ok {
			return info.keepAlive
		}
	}
	return 0
}

func TrackListener(sessionID string, listenerID uint32, connStr string, keepAlive int32) {
	pendingMu.Lock()
	defer pendingMu.Unlock()
	if listeners == nil {
		listeners = make(map[string]map[uint32]string)
	}
	if pending == nil {
		pending = make(map[string]map[string]*pendingInfo)
	}
	lm := listeners[sessionID]
	if lm == nil {
		lm = make(map[uint32]string)
		listeners[sessionID] = lm
	}
	lm[listenerID] = connStr
	addrMap := pending[sessionID]
	if addrMap == nil {
		addrMap = make(map[string]*pendingInfo)
		pending[sessionID] = addrMap
	}
	if info, ok := addrMap[connStr]; ok {
		info.count++
		// If multiple listeners point to the same connStr, the last one's KeepAlive wins
		// but they should be the same anyway.
		info.keepAlive = keepAlive
	} else {
		addrMap[connStr] = &pendingInfo{
			count:     1,
			keepAlive: keepAlive,
		}
	}
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
		if info, ok := addrMap[connStr]; ok {
			if info.count <= 1 {
				delete(addrMap, connStr)
			} else {
				info.count--
			}
		}
		if len(addrMap) == 0 {
			delete(pending, sessionID)
		}
	}
	return true
}
