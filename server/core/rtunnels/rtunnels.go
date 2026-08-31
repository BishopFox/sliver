package rtunnels

import (
	"errors"
	"fmt"
	"io"
	"sync"
)

var (
	rtunnels map[uint64]*RTunnel = make(map[uint64]*RTunnel)
	mutex    sync.RWMutex
)

var ErrDuplicateTunnelID = errors.New("reverse tunnel ID is already registered")

const (
	maxReverseTunnelFrameBytes    = 64 * 1024
	maxReverseTunnelPendingFrames = 64
	maxReverseTunnelPendingBytes  = maxReverseTunnelFrameBytes * maxReverseTunnelPendingFrames
)

var (
	ErrReverseTunnelFrameTooLarge = errors.New("reverse tunnel frame exceeds the size limit")
	ErrReverseTunnelWindow        = errors.New("reverse tunnel sequence exceeds the pending window")
	ErrReverseTunnelPendingBytes  = errors.New("reverse tunnel pending data exceeds the byte limit")
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

	mutex           *sync.RWMutex
	inboundMutex    sync.Mutex
	pendingInbound  map[uint64][]byte
	pendingBytes    int
	closeOnce       sync.Once
	done            chan struct{}
	authorizationID AuthorizationID
}

func NewRTunnel(id uint64, sID string, writer io.WriteCloser, readers ...io.ReadCloser) *RTunnel {
	return &RTunnel{
		ID:             id,
		SessionID:      sID,
		Readers:        readers,
		Writer:         writer,
		mutex:          &sync.RWMutex{},
		pendingInbound: map[uint64][]byte{},
		done:           make(chan struct{}),
	}
}

// NewAuthorizedRTunnel associates a reverse tunnel with the authorization that
// created its outbound connection. NewRTunnel remains available for callers
// that do not use reverse port forward authorization.
func NewAuthorizedRTunnel(id uint64, sessionID string, authorizationID AuthorizationID, writer io.WriteCloser, readers ...io.ReadCloser) *RTunnel {
	tunnel := NewRTunnel(id, sessionID, writer, readers...)
	tunnel.authorizationID = authorizationID
	return tunnel
}

// AuthorizationID returns the server-owned authorization associated with the
// tunnel, or an empty ID for non-reverse tunnels.
func (c *RTunnel) AuthorizationID() AuthorizationID {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.authorizationID
}

// Done is closed when the relay is revoked or otherwise closed. Writers use it
// to abandon a blocked implant send without leaking a goroutine.
func (c *RTunnel) Done() <-chan struct{} {
	return c.done
}

// ProcessInbound serializes one relay's inbound stream, bounds out-of-order
// buffering, and drains contiguous frames through write. Pending data is owned
// by this tunnel instance, so stale cleanup for a reused numeric ID cannot
// delete another generation's cache.
func (c *RTunnel) ProcessInbound(sequence uint64, data []byte, write func([]byte) error) (int, error) {
	if write == nil {
		return 0, errors.New("reverse tunnel write function is nil")
	}
	c.inboundMutex.Lock()
	defer c.inboundMutex.Unlock()

	expected := c.ReadSequence()
	if sequence < expected {
		return len(c.pendingInbound), nil
	}
	if len(data) > maxReverseTunnelFrameBytes {
		return len(c.pendingInbound), fmt.Errorf("%w: got %d bytes, limit %d", ErrReverseTunnelFrameTooLarge, len(data), maxReverseTunnelFrameBytes)
	}
	if sequence-expected >= maxReverseTunnelPendingFrames {
		return len(c.pendingInbound), fmt.Errorf("%w: got %d, expected %d", ErrReverseTunnelWindow, sequence, expected)
	}
	if _, duplicate := c.pendingInbound[sequence]; !duplicate {
		if c.pendingBytes+len(data) > maxReverseTunnelPendingBytes {
			return len(c.pendingInbound), ErrReverseTunnelPendingBytes
		}
		payload := append([]byte(nil), data...)
		c.pendingInbound[sequence] = payload
		c.pendingBytes += len(payload)
	}

	for {
		payload, ok := c.pendingInbound[expected]
		if !ok {
			break
		}
		if err := write(payload); err != nil {
			return len(c.pendingInbound), err
		}
		delete(c.pendingInbound, expected)
		c.pendingBytes -= len(payload)
		c.IncReadSequence()
		expected++
	}
	return len(c.pendingInbound), nil
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
	c.closeOnce.Do(func() {
		close(c.done)
		for _, rc := range c.Readers {
			if rc != nil {
				_ = rc.Close()
			}
		}
		if c.Writer != nil {
			_ = c.Writer.Close()
		}
		c.inboundMutex.Lock()
		c.pendingInbound = nil
		c.pendingBytes = 0
		c.inboundMutex.Unlock()
	})
}

// Tunnel - Add tunnel to mapping
func GetRTunnel(ID uint64) *RTunnel {
	mutex.RLock()
	defer mutex.RUnlock()
	return rtunnels[ID]
}

// TryAddRTunnel atomically rejects duplicate tunnel IDs instead of replacing
// and orphaning an existing tunnel.
func TryAddRTunnel(tun *RTunnel) bool {
	if tun == nil {
		return false
	}
	mutex.Lock()
	defer mutex.Unlock()

	if rtunnels[tun.ID] != nil {
		return false
	}
	rtunnels[tun.ID] = tun
	return true
}

// RemoveRTunnelIf removes ID only when it still points to expected. This keeps
// stale cleanup from deleting a different tunnel that reused the same ID.
func RemoveRTunnelIf(id uint64, expected *RTunnel) bool {
	if expected == nil {
		return false
	}
	mutex.Lock()
	defer mutex.Unlock()
	if rtunnels[id] != expected {
		return false
	}
	delete(rtunnels, id)
	return true
}

// CloseSession atomically detaches and closes only tunnels owned by sessionID.
// It is safe to call repeatedly during disconnect cleanup.
func CloseSession(sessionID string) int {
	return closeMatchingTunnels(func(tunnel *RTunnel) bool {
		return tunnel.SessionID == sessionID
	})
}

// CloseAuthorization atomically detaches and closes active relays created by a
// specific authorization. It never closes another session's tunnel.
func CloseAuthorization(sessionID string, authorizationID AuthorizationID) int {
	if authorizationID == "" {
		return 0
	}
	return closeMatchingTunnels(func(tunnel *RTunnel) bool {
		return tunnel.SessionID == sessionID && tunnel.AuthorizationID() == authorizationID
	})
}

func closeMatchingTunnels(matches func(*RTunnel) bool) int {
	mutex.Lock()
	tunnels := make([]*RTunnel, 0)
	for id, tunnel := range rtunnels {
		if tunnel != nil && matches(tunnel) {
			delete(rtunnels, id)
			tunnels = append(tunnels, tunnel)
		}
	}
	mutex.Unlock()

	for _, tunnel := range tunnels {
		tunnel.Close()
	}
	return len(tunnels)
}
