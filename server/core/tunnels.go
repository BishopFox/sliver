package core

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"sync"
	"time"

	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

var (
	// Tunnels - Interacting with duplex tunnels
	Tunnels = tunnels{
		tunnels: map[uint64]*Tunnel{},
		mutex:   &sync.Mutex{},
	}

	// ErrInvalidTunnelID - Invalid tunnel ID value
	ErrInvalidTunnelID = errors.New("invalid tunnel ID")

	// ErrInvalidSessionID - Invalid session ID value
	ErrInvalidSessionID = errors.New("invalid session ID")
)

const (
	// delayBeforeClose - delay before closing the tunnel.
	// I assume 10 seconds may be an overkill for a good connection, but it looks good enough for less stable one.
	delayBeforeClose = 10 * time.Second
)

// Tunnel  - Essentially just a mapping between a specific client and sliver
// with an identifier, these tunnels are full duplex. The server doesn't really
// care what data gets passed back and forth it just facilitates the connection
type Tunnel struct {
	ID        uint64
	SessionID string

	ToImplant         chan []byte
	ToImplantSequence uint64

	FromImplant         chan *sliverpb.TunnelData
	FromImplantSequence uint64

	Client rpcpb.SliverRPC_TunnelDataServer

	mutex               *sync.RWMutex
	clientMutex         sync.RWMutex
	clientBound         chan struct{}
	clientBoundOnce     sync.Once
	closeOnce           sync.Once
	done                chan struct{}
	lastDataMessageTime time.Time
}

func NewTunnel(id uint64, sessionID string) *Tunnel {
	return &Tunnel{
		ID:          id,
		SessionID:   sessionID,
		ToImplant:   make(chan []byte),
		FromImplant: make(chan *sliverpb.TunnelData),

		mutex:               &sync.RWMutex{},
		clientBound:         make(chan struct{}),
		done:                make(chan struct{}),
		lastDataMessageTime: time.Now(), // need to be initialized
	}
}

// BindClient reserves this tunnel for the first client stream.
func (t *Tunnel) BindClient(client rpcpb.SliverRPC_TunnelDataServer) bool {
	t.clientMutex.Lock()
	defer t.clientMutex.Unlock()
	if t.Client != nil {
		return false
	}

	t.Client = client
	return true
}

// MarkClientBound signals that the reserved stream accepted its bind frame.
func (t *Tunnel) MarkClientBound(client rpcpb.SliverRPC_TunnelDataServer) bool {
	t.clientMutex.Lock()
	defer t.clientMutex.Unlock()
	if t.Client != client {
		return false
	}

	t.clientBoundOnce.Do(func() { close(t.clientBound) })
	return true
}

// IsClient reports whether client owns this tunnel's stream binding.
func (t *Tunnel) IsClient(client rpcpb.SliverRPC_TunnelDataServer) bool {
	t.clientMutex.RLock()
	defer t.clientMutex.RUnlock()
	return t.Client == client
}

// ClientBound is closed after a client stream binds to the tunnel.
func (t *Tunnel) ClientBound() <-chan struct{} {
	return t.clientBound
}

func (t *Tunnel) setLastMessageTime() {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.lastDataMessageTime = time.Now()
}

// Touch records tunnel activity before scheduling an asynchronous close.
func (t *Tunnel) Touch() {
	t.setLastMessageTime()
}

func (t *Tunnel) GetLastMessageTime() time.Time {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	return t.lastDataMessageTime
}

func (t *Tunnel) SendDataFromImplant(tunnelData *sliverpb.TunnelData) bool {
	// Setting the date right before and right after message, since channel can be blocked for some amount of time
	t.setLastMessageTime()
	defer t.setLastMessageTime()

	select {
	case t.FromImplant <- tunnelData:
		return true
	case <-t.done:
		return false
	}
}

// SendDataToImplant queues tunnel data unless the tunnel has already closed.
func (t *Tunnel) SendDataToImplant(data []byte) bool {
	t.setLastMessageTime()
	defer t.setLastMessageTime()

	select {
	case t.ToImplant <- data:
		return true
	case <-t.done:
		return false
	}
}

// Done is closed when the tunnel is removed from the server registry.
func (t *Tunnel) Done() <-chan struct{} {
	return t.done
}

// Close marks the tunnel closed without waiting for a client or implant reader.
func (t *Tunnel) Close() {
	t.closeOnce.Do(func() {
		close(t.done)
	})
}

type tunnels struct {
	tunnels map[uint64]*Tunnel
	mutex   *sync.Mutex
}

func (t *tunnels) Create(sessionID string) (*Tunnel, error) {
	tunnelID := NewTunnelID()
	session := Sessions.Get(sessionID)
	if session == nil {
		return nil, ErrInvalidSessionID
	}

	tunnel := NewTunnel(
		tunnelID,
		session.ID,
	)

	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.tunnels[tunnel.ID] = tunnel

	return tunnel, nil
}

// ScheduleClose - schedules a close for tunnel, must be called as routine.
// will close it once there is no data for at least delayBeforeClose delay since last message
// This is _necessary_ since we processing messages asynchronously
// and if tunnelCloseHandler routine will fire before tunnelDataHandler routine we will lose some data
// (this is what happens for socks and portfwd)
// The quiet period protects asynchronously delivered tunnel data from being
// discarded when a close message overtakes it.
func (t *tunnels) ScheduleClose(tunnelID uint64) {
	tunnel := t.Get(tunnelID)
	if tunnel == nil {
		return
	}

	timeDelta := time.Since(tunnel.GetLastMessageTime())

	coreLog.Printf("Scheduled close for channel %d (delta: %v)", tunnelID, timeDelta)

	if timeDelta >= delayBeforeClose {
		coreLog.Printf("Closing channel %d", tunnelID)
		t.Close(tunnelID)
	} else {
		// Reschedule
		coreLog.Printf("Rescheduling closing channel %d", tunnelID)
		time.Sleep(delayBeforeClose - timeDelta + time.Second)
		go t.ScheduleClose(tunnelID)
	}
}

// Close - closing tunnel
// It's preferred to use ScheduleClose function if you don't 100% sure there is no more data to receive
func (t *tunnels) Close(tunnelID uint64) error {
	t.mutex.Lock()
	tunnel := t.tunnels[tunnelID]
	if tunnel == nil {
		t.mutex.Unlock()
		return ErrInvalidTunnelID
	}
	delete(t.tunnels, tunnelID)
	t.mutex.Unlock()

	// Never wait on tunnel consumers while holding the global registry lock.
	// A tunnel can be created and closed before the bidirectional stream binds,
	// in which case neither channel has a receiver.
	tunnel.Close()
	return nil
}

// CloseForClient closes every tunnel owned by a terminated client stream.
// Removal happens under the registry lock; tunnel shutdown happens afterward
// so no slow consumer can block unrelated tunnel operations.
func (t *tunnels) CloseForClient(client rpcpb.SliverRPC_TunnelDataServer) {
	if client == nil {
		return
	}

	closed := []*Tunnel{}
	t.mutex.Lock()
	for tunnelID, tunnel := range t.tunnels {
		if tunnel.IsClient(client) {
			delete(t.tunnels, tunnelID)
			closed = append(closed, tunnel)
		}
	}
	t.mutex.Unlock()

	for _, tunnel := range closed {
		tunnel.Close()
	}
}

// Get - Get a tunnel
func (t *tunnels) Get(tunnelID uint64) *Tunnel {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	return t.tunnels[tunnelID]
}

// NewTunnelID - New 64-bit identifier
func NewTunnelID() uint64 {
	randBuf := make([]byte, 8)
	rand.Read(randBuf)
	return binary.LittleEndian.Uint64(randBuf)
}
