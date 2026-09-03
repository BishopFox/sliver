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
	"context"
	"net/url"
	"sync"
	"time"

	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

const (
	tunnelSendTimeout = 10 * time.Second

	// Bound live tunnel state independently from the replay tombstone window.
	// Retired IDs are remembered in FIFO order; once the window rotates, exact
	// pointer checks still prevent stale local work from deleting a replacement,
	// while random 64-bit wire IDs make an accidental post-window reuse remote.
	maxLiveTunnelIDsPerConnection    = 4096
	maxRetiredTunnelIDsPerConnection = 4096
)

// TunnelAddResult distinguishes a replayed ID from exhaustion of the bounded
// live-tunnel domain. Capacity exhaustion rejects only the new tunnel and
// leaves the C2 connection available for existing and later work.
type TunnelAddResult uint8

// TunnelAdded and the related results describe a tunnel publication attempt.
const (
	TunnelAdded TunnelAddResult = iota
	TunnelAddDuplicate
	TunnelAddCapacityExhausted
	TunnelAddConnectionClosed
	TunnelAddInvalid
	TunnelAddSetupCanceled
)

// PendingTunnel owns a not-yet-published tunnel setup for one C2 connection.
// Its wire ID is reserved before any external resource is opened, so an
// overtaking close can cancel the exact setup and place the ID in the bounded
// replay window before a later setup can reuse it.
type PendingTunnel struct {
	connection *Connection
	tunnelID   uint64
	ctx        context.Context
	cancel     context.CancelFunc
}

// Context is canceled when the setup is retired, its deadline expires, or its
// owning C2 connection closes.
func (p *PendingTunnel) Context() context.Context {
	if p == nil || p.ctx == nil {
		return context.Background()
	}
	return p.ctx
}

// Cancel retires this exact setup without affecting another tunnel ID.
func (p *PendingTunnel) Cancel() {
	if p == nil {
		return
	}
	if p.cancel != nil {
		p.cancel()
	}
	if p.connection != nil {
		p.connection.cancelPendingTunnelIf(p)
	}
}

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
	pendingTunnels map[uint64]*PendingTunnel
	retiredTunnels map[uint64]struct{}
	retiredOrder   []uint64
	retiredCursor  int
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
		if c.pendingTunnels == nil {
			c.pendingTunnels = map[uint64]*PendingTunnel{}
		}
		if c.retiredTunnels == nil {
			c.retiredTunnels = map[uint64]struct{}{}
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

// TryAddTunnel adds a tunnel generation to the connection. Recently retired
// wire IDs remain unavailable within a bounded replay window. Live-capacity
// exhaustion rejects this generation without failing unrelated C2 work.
func (c *Connection) TryAddTunnel(tun *Tunnel) TunnelAddResult {
	if c == nil || tun == nil {
		return TunnelAddInvalid
	}
	c.initializeLifecycle()
	c.initializeTunnelState()
	c.mutex.Lock()
	if result := c.tunnelAdmissionResultLocked(tun.ID); result != TunnelAdded {
		c.mutex.Unlock()
		return result
	}
	c.publishTunnelLocked(tun)
	c.mutex.Unlock()
	return TunnelAdded
}

// tunnelAdmissionResultLocked validates one new wire generation. The caller
// must hold c.mutex so live, pending, retired, and capacity checks observe one
// consistent connection state.
func (c *Connection) tunnelAdmissionResultLocked(tunnelID uint64) TunnelAddResult {
	select {
	case <-c.done:
		return TunnelAddConnectionClosed
	default:
	}
	if c.tunnels[tunnelID] != nil || c.pendingTunnels[tunnelID] != nil {
		return TunnelAddDuplicate
	}
	if _, retired := c.retiredTunnels[tunnelID]; retired {
		return TunnelAddDuplicate
	}
	if len(c.tunnels)+len(c.pendingTunnels) >= maxLiveTunnelIDsPerConnection {
		return TunnelAddCapacityExhausted
	}
	return TunnelAdded
}

// BeginTunnel reserves one tunnel generation before a handler opens an
// external resource. An overtaking peer close retires the wire ID into the
// bounded replay window and cancels the returned context. Timeout must be finite
// for network setup callers; non-positive values create an owner-cancelable
// reservation for callers that already impose a stricter deadline.
func (c *Connection) BeginTunnel(tunnelID uint64, timeout time.Duration) (*PendingTunnel, TunnelAddResult) {
	if c == nil {
		return nil, TunnelAddInvalid
	}
	c.initializeLifecycle()
	c.initializeTunnelState()

	var (
		ctx    context.Context
		cancel context.CancelFunc
	)
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	pending := &PendingTunnel{
		connection: c,
		tunnelID:   tunnelID,
		ctx:        ctx,
		cancel:     cancel,
	}

	c.mutex.Lock()
	if result := c.tunnelAdmissionResultLocked(tunnelID); result != TunnelAdded {
		c.mutex.Unlock()
		cancel()
		return nil, result
	}
	c.pendingTunnels[tunnelID] = pending
	c.mutex.Unlock()

	// Context cancellation must detach a non-cooperative setup even when the
	// handler has not returned from its dial call yet.
	go func() {
		select {
		case <-c.Done():
			pending.Cancel()
		case <-ctx.Done():
			pending.Cancel()
		}
	}()
	return pending, TunnelAdded
}

// PublishTunnel atomically replaces an exact pending setup with its active
// tunnel. A canceled, timed-out, disconnected, or superseded setup can never
// publish a late socket.
func (c *Connection) PublishTunnel(pending *PendingTunnel, tun *Tunnel) TunnelAddResult {
	if c == nil || pending == nil || tun == nil || pending.connection != c || pending.tunnelID != tun.ID {
		return TunnelAddInvalid
	}
	c.initializeLifecycle()
	c.initializeTunnelState()
	c.mutex.Lock()
	select {
	case <-c.done:
		c.mutex.Unlock()
		pending.Cancel()
		return TunnelAddConnectionClosed
	default:
	}
	if c.pendingTunnels[tun.ID] != pending {
		c.mutex.Unlock()
		pending.Cancel()
		return TunnelAddSetupCanceled
	}
	if pending.ctx.Err() != nil {
		delete(c.pendingTunnels, tun.ID)
		c.retireTunnelIDLocked(tun.ID)
		c.mutex.Unlock()
		pending.Cancel()
		return TunnelAddSetupCanceled
	}
	if c.tunnels[tun.ID] != nil {
		delete(c.pendingTunnels, tun.ID)
		c.mutex.Unlock()
		pending.Cancel()
		return TunnelAddDuplicate
	}
	delete(c.pendingTunnels, tun.ID)
	c.publishTunnelLocked(tun)
	c.mutex.Unlock()
	// Stop the setup deadline only after the active map owns the tunnel.
	pending.cancel()
	return TunnelAdded
}

// CancelPendingTunnel retires an unowned or pending wire ID and cancels the
// exact pending setup. If publication won the race, the active tunnel remains
// available for the caller to close normally.
func (c *Connection) CancelPendingTunnel(tunnelID uint64) bool {
	if c == nil {
		return false
	}
	c.initializeLifecycle()
	c.initializeTunnelState()
	var pending *PendingTunnel
	c.mutex.Lock()
	select {
	case <-c.done:
		c.mutex.Unlock()
		return false
	default:
	}
	if c.tunnels[tunnelID] != nil {
		c.mutex.Unlock()
		return false
	}
	pending = c.pendingTunnels[tunnelID]
	if pending != nil {
		delete(c.pendingTunnels, tunnelID)
		c.retireTunnelIDLocked(tunnelID)
	} else if _, retired := c.retiredTunnels[tunnelID]; !retired {
		c.retireTunnelIDLocked(tunnelID)
	}
	c.mutex.Unlock()
	if pending != nil {
		pending.cancel()
		return true
	}
	return false
}

func (c *Connection) cancelPendingTunnelIf(pending *PendingTunnel) bool {
	if c == nil || pending == nil {
		return false
	}
	c.initializeTunnelState()
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.pendingTunnels[pending.tunnelID] != pending {
		return false
	}
	delete(c.pendingTunnels, pending.tunnelID)
	c.retireTunnelIDLocked(pending.tunnelID)
	return true
}

// retireTunnelIDLocked records one completed or canceled wire generation in a
// fixed-size FIFO replay window. The caller must hold c.mutex.
func (c *Connection) retireTunnelIDLocked(tunnelID uint64) {
	if _, retired := c.retiredTunnels[tunnelID]; retired {
		return
	}
	if len(c.retiredOrder) < maxRetiredTunnelIDsPerConnection {
		c.retiredOrder = append(c.retiredOrder, tunnelID)
		c.retiredTunnels[tunnelID] = struct{}{}
		return
	}

	evicted := c.retiredOrder[c.retiredCursor]
	delete(c.retiredTunnels, evicted)
	c.retiredOrder[c.retiredCursor] = tunnelID
	c.retiredCursor = (c.retiredCursor + 1) % maxRetiredTunnelIDsPerConnection
	c.retiredTunnels[tunnelID] = struct{}{}
}

func (c *Connection) publishTunnelLocked(tun *Tunnel) {
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
}

// AddTunnel is the compatibility boolean wrapper for callers that do not need
// to distinguish duplicate IDs, live-capacity rejection, or connection close.
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

	if c.tunnels[ID] != nil {
		delete(c.tunnels, ID)
		c.retireTunnelIDLocked(ID)
	}
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
	c.retireTunnelIDLocked(ID)
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
	for id, pending := range c.pendingTunnels {
		pending.cancel()
		delete(c.pendingTunnels, id)
	}

	for id, tunnel := range c.tunnels {
		tunnel.Close()

		delete(c.tunnels, id)
	}
	clear(c.retiredTunnels)
	c.retiredOrder = nil
	c.retiredCursor = 0
}
