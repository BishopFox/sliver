package transports

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

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
	"net/url"
	"sync"
	"time"

	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

const (
	tunnelSendTimeout = 10 * time.Second

	// maxClaimedTunnelIDsPerConnection bounds the permanent replay-defense
	// history for one C2 connection. Tunnel IDs cannot be reused safely because
	// the wire protocol has no generation number. Exhaustion therefore closes
	// the connection and creates a fresh ID domain on reconnect.
	maxClaimedTunnelIDsPerConnection = 4096
)

// TunnelAddResult distinguishes a replayed ID from exhaustion of the bounded
// per-connection ID domain. Capacity exhaustion has already failed the C2
// connection closed before it is returned.
type TunnelAddResult uint8

// TunnelAdded and the related results describe a tunnel publication attempt.
const (
	TunnelAdded TunnelAddResult = iota
	TunnelAddDuplicate
	TunnelAddCapacityExhausted
	TunnelAddConnectionClosed
	TunnelAddInvalid
)

type Connection struct {
	Send           chan *pb.Envelope
	Recv           chan *pb.Envelope
	IsOpen         bool
	ctrl           chan struct{}
	cleanup        func()
	cleanupOnce    sync.Once
	lifecycleOnce  sync.Once
	stateOnce      sync.Once
	done           chan struct{}
	tunnels        map[uint64]*Tunnel
	claimedTunnels map[uint64]struct{}
	mutex          *sync.RWMutex

	uri      *url.URL
	proxyURL *url.URL

	Start Start
	Stop  Stop
}

// TunnelEnvelopeBuilder builds a sequenced tunnel envelope for transport.
type TunnelEnvelopeBuilder func(sequence uint64, ack uint64) (*pb.Envelope, error)

func (c *Connection) initializeLifecycle() {
	c.lifecycleOnce.Do(func() {
		c.done = make(chan struct{})
	})
}

func (c *Connection) initializeTunnelState() {
	c.stateOnce.Do(func() {
		if c.mutex == nil {
			c.mutex = &sync.RWMutex{}
		}
		if c.tunnels == nil {
			c.tunnels = map[uint64]*Tunnel{}
		}
		if c.claimedTunnels == nil {
			c.claimedTunnels = map[uint64]struct{}{}
		}
	})
}

// Done is closed before transport cleanup starts. Tunnel producers use it to
// abandon an outbound envelope when the C2 connection is no longer usable.
func (c *Connection) Done() <-chan struct{} {
	c.initializeLifecycle()
	return c.done
}

// SendEnvelope queues an envelope while the connection remains live.
func (c *Connection) SendEnvelope(envelope *pb.Envelope) bool {
	if c == nil || envelope == nil || c.Send == nil {
		return false
	}
	timer := time.NewTimer(tunnelSendTimeout)
	defer timer.Stop()
	select {
	case c.Send <- envelope:
		return true
	case <-c.Done():
		return false
	case <-timer.C:
		return false
	}
}

// SendTunnelEnvelope bounds a producer on both the C2 and exact tunnel
// lifecycles. Closing one tunnel therefore unblocks its writers without waiting
// for an otherwise healthy connection to reconnect.
func (c *Connection) SendTunnelEnvelope(tunnel *Tunnel, envelope *pb.Envelope) bool {
	if c == nil || tunnel == nil || envelope == nil || c.Send == nil {
		return false
	}
	timer := time.NewTimer(tunnelSendTimeout)
	defer timer.Stop()
	select {
	case c.Send <- envelope:
		return true
	case <-tunnel.Done():
		return false
	case <-c.Done():
		return false
	case <-timer.C:
		return false
	}
}

// URL - Get the c2 URL of the connection
func (c *Connection) URL() string {
	if c.uri == nil {
		return ""
	}
	return c.uri.String()
}

// ProxyURL - Get the c2 URL of the connection
func (c *Connection) ProxyURL() string {
	if c.proxyURL == nil {
		return ""
	}
	return c.proxyURL.String()
}

// Cleanup - Execute cleanup once
func (c *Connection) Cleanup() {
	c.initializeLifecycle()
	c.cleanupOnce.Do(func() {
		close(c.done)
		if c.cleanup != nil {
			c.cleanup()
		}
		c.IsOpen = false
		c.removeAndCloseAllTunnels()
	})
}

// Tunnel - Add tunnel to mapping
func (c *Connection) Tunnel(ID uint64) *Tunnel {
	if c == nil {
		return nil
	}
	c.initializeTunnelState()
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.tunnels[ID]
}

// TryAddTunnel adds a tunnel generation to the connection. Retired wire IDs
// remain claimed for the lifetime of the C2 connection. Once the bounded ID
// domain is exhausted, Cleanup is invoked after releasing mutex so tunnel
// cleanup cannot deadlock while re-entering the tunnel map.
func (c *Connection) TryAddTunnel(tun *Tunnel) TunnelAddResult {
	if c == nil || tun == nil {
		return TunnelAddInvalid
	}
	c.initializeLifecycle()
	c.initializeTunnelState()
	c.mutex.Lock()
	select {
	case <-c.done:
		c.mutex.Unlock()
		return TunnelAddConnectionClosed
	default:
	}
	if c.tunnels[tun.ID] != nil {
		c.mutex.Unlock()
		return TunnelAddDuplicate
	}
	if _, claimed := c.claimedTunnels[tun.ID]; claimed {
		c.mutex.Unlock()
		return TunnelAddDuplicate
	}
	if len(c.claimedTunnels) >= maxClaimedTunnelIDsPerConnection {
		c.mutex.Unlock()
		c.Cleanup()
		return TunnelAddCapacityExhausted
	}
	tun.setPeerCloseNotifier(func(sequence uint64) error {
		data, err := proto.Marshal(&pb.TunnelData{
			Closed:   true,
			TunnelID: tun.ID,
			Sequence: sequence,
		})
		if err != nil {
			return err
		}
		if !c.SendTunnelEnvelope(tun, &pb.Envelope{Type: pb.MsgTunnelClose, Data: data}) {
			return ErrTunnelClosed
		}
		return nil
	})
	c.tunnels[tun.ID] = tun
	c.claimedTunnels[tun.ID] = struct{}{}
	c.mutex.Unlock()
	return TunnelAdded
}

// AddTunnel is the compatibility boolean wrapper for callers that do not need
// to distinguish duplicate IDs from a closed ID domain.
func (c *Connection) AddTunnel(tun *Tunnel) bool {
	return c.TryAddTunnel(tun) == TunnelAdded
}

func (c *Connection) ownsTunnel(tunnel *Tunnel) bool {
	if c == nil || tunnel == nil {
		return false
	}
	c.initializeTunnelState()
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.tunnels[tunnel.ID] == tunnel
}

// QueueTunnelData combines exact-generation ownership, outbound sequencing,
// bounded delivery, and advancement after acceptance in one API.
func (c *Connection) QueueTunnelData(tunnel *Tunnel, build TunnelEnvelopeBuilder) error {
	if c == nil || tunnel == nil || build == nil {
		return ErrTunnelClosed
	}
	return tunnel.queueOutbound(func(sequence uint64) error {
		if !c.ownsTunnel(tunnel) {
			return ErrTunnelClosed
		}
		envelope, err := build(sequence, tunnel.ReadSequence())
		if err != nil {
			return err
		}
		if !c.SendTunnelEnvelope(tunnel, envelope) {
			return ErrTunnelClosed
		}
		return nil
	})
}

// QueueTunnelControl serializes resend/control frames with data and terminal
// close without consuming the next data sequence.
func (c *Connection) QueueTunnelControl(tunnel *Tunnel, build TunnelEnvelopeBuilder) error {
	if c == nil || tunnel == nil || build == nil {
		return ErrTunnelClosed
	}
	return tunnel.queueControl(func(sequence uint64, ack uint64) error {
		if !c.ownsTunnel(tunnel) {
			return ErrTunnelClosed
		}
		envelope, err := build(sequence, ack)
		if err != nil {
			return err
		}
		if !c.SendTunnelEnvelope(tunnel, envelope) {
			return ErrTunnelClosed
		}
		return nil
	})
}

// RemoveTunnel - Add tunnel to mapping
func (c *Connection) RemoveTunnel(ID uint64) {
	if c == nil {
		return
	}
	c.initializeTunnelState()
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.tunnels, ID)
}

// RemoveTunnelIf detaches ID only when it still refers to the expected tunnel.
// This prevents a stale local or peer close from deleting a reused generation.
func (c *Connection) RemoveTunnelIf(ID uint64, expected *Tunnel) bool {
	if c == nil || expected == nil {
		return false
	}
	c.initializeTunnelState()
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.tunnels[ID] != expected {
		return false
	}
	delete(c.tunnels, ID)
	return true
}

// CloseTunnelLocal atomically detaches one generation, emits its sequenced
// terminal notification exactly once, and fails the C2 closed if delivery is
// impossible. A disconnect then guarantees server-side relay cleanup.
func (c *Connection) CloseTunnelLocal(tunnel *Tunnel) bool {
	if tunnel == nil || !c.ownsTunnel(tunnel) {
		return false
	}
	initiated, err := tunnel.closeLocal()
	if !initiated || !c.RemoveTunnelIf(tunnel.ID, tunnel) {
		return false
	}
	if err != nil && !tunnel.PeerTeardownPending() {
		c.Cleanup()
	}
	return true
}

// CloseTunnelRemote atomically detaches a peer-closed generation without
// echoing another terminal frame.
func (c *Connection) CloseTunnelRemote(tunnel *Tunnel) bool {
	if tunnel == nil || !c.ownsTunnel(tunnel) {
		return false
	}
	tunnel.closeRemote()
	return c.RemoveTunnelIf(tunnel.ID, tunnel)
}

func (c *Connection) removeAndCloseAllTunnels() {
	if c == nil {
		return
	}
	c.initializeTunnelState()
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for id, tunnel := range c.tunnels {
		tunnel.Close()

		delete(c.tunnels, id)
	}
}
