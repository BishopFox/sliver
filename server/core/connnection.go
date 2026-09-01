package core

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	"errors"
	"sync"
	"time"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/gofrs/uuid"
)

// maxClaimedReverseTunnelIDsPerConnection bounds the permanent replay-defense
// history for one C2 connection. Reverse tunnel IDs cannot be reused safely
// because the wire protocol has no generation number. A connection that uses
// the entire history is therefore closed and must establish a fresh ID domain.
// The limit is deliberately much larger than the concurrent reverse-tunnel
// quota while still bounding attacker-controlled memory growth.
const (
	maxClaimedReverseTunnelIDsPerConnection = 4096

	// DefaultImplantSendTimeout bounds server producers when a transport has
	// stopped consuming outbound envelopes without closing its connection yet.
	DefaultImplantSendTimeout = 10 * time.Second

	implantSendRetryInterval = time.Millisecond
)

var (
	// ErrImplantConnectionClosed indicates that an outbound envelope could not
	// be queued because the connection is closing or already closed.
	ErrImplantConnectionClosed = errors.New("implant connection closed")

	// ErrImplantSendTimeout indicates that an outbound transport queue did not
	// accept an envelope before its bounded deadline.
	ErrImplantSendTimeout = errors.New("implant send timeout")

	// ErrInvalidImplantSend indicates a nil connection, envelope, or send queue.
	ErrInvalidImplantSend = errors.New("invalid implant send")
)

// ReverseTunnelIDClaimResult distinguishes a replayed ID from exhaustion of
// the per-connection ID domain. Callers should reject duplicates while leaving
// the connection live; capacity exhaustion has already failed it closed.
type ReverseTunnelIDClaimResult uint8

const (
	// ReverseTunnelIDClaimed indicates that the connection accepted a new ID.
	ReverseTunnelIDClaimed ReverseTunnelIDClaimResult = iota
	// ReverseTunnelIDDuplicate indicates that the connection already owns the ID.
	ReverseTunnelIDDuplicate
	// ReverseTunnelIDCapacityExhausted indicates that no more IDs may be claimed.
	ReverseTunnelIDCapacityExhausted
	// ReverseTunnelIDConnectionClosed indicates that the connection is closing.
	ReverseTunnelIDConnectionClosed
)

// ImplantConnection - Abstract connection to an implant
type ImplantConnection struct {
	ID               string
	Send             chan *sliverpb.Envelope
	RespMutex        *sync.RWMutex
	Resp             map[int64]chan *sliverpb.Envelope
	Transport        string
	RemoteAddress    string
	LastMessage      time.Time
	LastMessageMutex *sync.RWMutex
	lifecycleOnce    sync.Once
	lifecycleMutex   sync.Mutex
	done             chan struct{}
	closed           bool
	cleanupSet       bool
	cleanup          func()
	claimedTunnelIDs map[uint64]struct{}
}

// TryClaimReverseTunnelID permanently reserves a wire tunnel ID for this C2
// connection. The protocol has no generation number, so never reusing an ID is
// what prevents a delayed frame from targeting a newer relay generation. Once
// the bounded ID domain is exhausted, the connection is failed closed after
// releasing lifecycleMutex so cleanup can safely re-enter connection methods.
func (c *ImplantConnection) TryClaimReverseTunnelID(tunnelID uint64) ReverseTunnelIDClaimResult {
	if c == nil {
		return ReverseTunnelIDConnectionClosed
	}
	c.initializeLifecycle()
	c.lifecycleMutex.Lock()
	if c.closed {
		c.lifecycleMutex.Unlock()
		return ReverseTunnelIDConnectionClosed
	}
	if c.claimedTunnelIDs == nil {
		c.claimedTunnelIDs = map[uint64]struct{}{}
	}
	if _, claimed := c.claimedTunnelIDs[tunnelID]; claimed {
		c.lifecycleMutex.Unlock()
		return ReverseTunnelIDDuplicate
	}
	if len(c.claimedTunnelIDs) >= maxClaimedReverseTunnelIDsPerConnection {
		c.lifecycleMutex.Unlock()
		c.Close()
		return ReverseTunnelIDCapacityExhausted
	}
	c.claimedTunnelIDs[tunnelID] = struct{}{}
	c.lifecycleMutex.Unlock()
	return ReverseTunnelIDClaimed
}

// ClaimReverseTunnelID is the compatibility boolean wrapper for callers that
// do not need to distinguish duplicate IDs from a closed ID domain.
func (c *ImplantConnection) ClaimReverseTunnelID(tunnelID uint64) bool {
	return c.TryClaimReverseTunnelID(tunnelID) == ReverseTunnelIDClaimed
}

func (c *ImplantConnection) initializeLifecycle() {
	c.lifecycleOnce.Do(func() {
		c.done = make(chan struct{})
	})
}

// Close marks the implant connection closed and runs its cleanup callback. The
// Done signal is closed first so blocked work can fail closed even if cleanup
// takes time. It is safe for transport, protocol, and rejection paths to call
// Close concurrently; cleanup is performed exactly once.
func (c *ImplantConnection) Close() {
	c.initializeLifecycle()
	c.lifecycleMutex.Lock()
	if c.closed {
		c.lifecycleMutex.Unlock()
		return
	}
	c.closed = true
	close(c.done)
	cleanup := c.cleanup
	c.cleanup = nil
	c.lifecycleMutex.Unlock()
	if cleanup != nil {
		cleanup()
	}
}

// Done is closed when the implant connection begins closing.
func (c *ImplantConnection) Done() <-chan struct{} {
	c.initializeLifecycle()
	return c.done
}

// SetCleanup installs the connection's one-shot cleanup callback. It returns
// false for a duplicate registration or a connection that already closed.
// Callers must install cleanup before publishing connection-owned state.
func (c *ImplantConnection) SetCleanup(callback func()) bool {
	if callback == nil {
		return false
	}
	c.initializeLifecycle()
	c.lifecycleMutex.Lock()
	if c.cleanupSet || c.closed {
		c.lifecycleMutex.Unlock()
		return false
	}
	c.cleanupSet = true
	c.cleanup = callback
	c.lifecycleMutex.Unlock()
	return true
}

// SendEnvelope queues an outbound envelope while the connection remains live.
// Every producer must use this method instead of writing to Send directly so a
// stalled transport cannot strand the producer forever.
func (c *ImplantConnection) SendEnvelope(envelope *sliverpb.Envelope, timeout time.Duration) error {
	return c.SendEnvelopeUntil(envelope, nil, timeout)
}

// SendEnvelopeUntil is SendEnvelope with an additional owner lifecycle. This
// is used by tunnel and pivot producers, where the narrower owner may close
// while the underlying C2 connection remains healthy.
func (c *ImplantConnection) SendEnvelopeUntil(envelope *sliverpb.Envelope, ownerDone <-chan struct{}, timeout time.Duration) error {
	if c == nil || envelope == nil || c.Send == nil {
		return ErrInvalidImplantSend
	}
	c.initializeLifecycle()
	if sent, err := c.trySendEnvelope(envelope, ownerDone); sent || err != nil {
		return err
	}
	if timeout <= 0 {
		timeout = DefaultImplantSendTimeout
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	retry := time.NewTicker(implantSendRetryInterval)
	defer retry.Stop()

	for {
		select {
		case <-c.done:
			return ErrImplantConnectionClosed
		case <-ownerDone:
			return ErrImplantConnectionClosed
		case <-timer.C:
			return ErrImplantSendTimeout
		case <-retry.C:
		}
		if sent, err := c.trySendEnvelope(envelope, ownerDone); sent || err != nil {
			return err
		}
	}
}

func (c *ImplantConnection) trySendEnvelope(envelope *sliverpb.Envelope, ownerDone <-chan struct{}) (bool, error) {
	// Close and successful queueing share lifecycleMutex as their linearization
	// point. The non-blocking send keeps Close responsive even when an
	// unbuffered transport queue is not being consumed.
	c.lifecycleMutex.Lock()
	defer c.lifecycleMutex.Unlock()
	if c.closed {
		return false, ErrImplantConnectionClosed
	}
	select {
	case <-ownerDone:
		return false, ErrImplantConnectionClosed
	default:
	}
	select {
	case c.Send <- envelope:
		return true, nil
	default:
		return false, nil
	}
}

// DeliverResponse completes a pending synchronous request without ever
// blocking a transport reader. Request response channels are single-message
// buffers, so a duplicate or late delivery is safely rejected.
func (c *ImplantConnection) DeliverResponse(envelope *sliverpb.Envelope) bool {
	if c == nil || envelope == nil || envelope.ID == 0 || c.RespMutex == nil || c.Resp == nil {
		return false
	}
	c.initializeLifecycle()
	c.lifecycleMutex.Lock()
	defer c.lifecycleMutex.Unlock()
	if c.closed {
		return false
	}
	c.RespMutex.RLock()
	response, ok := c.Resp[envelope.ID]
	c.RespMutex.RUnlock()
	if !ok || response == nil {
		return false
	}
	select {
	case response <- envelope:
		return true
	default:
		return false
	}
}

// GetLastMessage - Retrieves the last message time
func (c *ImplantConnection) GetLastMessage() time.Time {
	c.LastMessageMutex.RLock()
	defer c.LastMessageMutex.RUnlock()

	return c.LastMessage
}

// UpdateLastMessage - Updates the last message time
func (c *ImplantConnection) UpdateLastMessage() {
	c.LastMessageMutex.Lock()
	defer c.LastMessageMutex.Unlock()

	c.LastMessage = time.Now()
}

// NewImplantConnection - Creates a new implant connection
func NewImplantConnection(transport string, remoteAddress string) *ImplantConnection {
	connection := &ImplantConnection{
		ID:               generateImplantConnectionID(),
		Send:             make(chan *sliverpb.Envelope),
		RespMutex:        &sync.RWMutex{},
		LastMessageMutex: &sync.RWMutex{},
		Resp:             map[int64]chan *sliverpb.Envelope{},
		Transport:        transport,
		RemoteAddress:    remoteAddress,
		claimedTunnelIDs: map[uint64]struct{}{},
	}
	connection.initializeLifecycle()
	return connection
}

func generateImplantConnectionID() string {
	id, _ := uuid.NewV4()
	return id.String()
}

func (c *ImplantConnection) RequestResend(data []byte) {
	_ = c.SendEnvelope(&sliverpb.Envelope{
		Type: sliverpb.MsgTunnelData,
		Data: data,
	}, DefaultImplantSendTimeout)
}
