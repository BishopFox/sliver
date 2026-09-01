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
	"fmt"
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

	// ErrTunnelClosed indicates that work retained an exact tunnel generation
	// after it had already been detached.
	ErrTunnelClosed = errors.New("tunnel is closed")

	// ErrTunnelFrameTooLarge rejects one wire frame before it can enter a
	// generation's reorder or resend state.
	ErrTunnelFrameTooLarge = errors.New("tunnel frame exceeds the size limit")

	// ErrTunnelSequenceWindow bounds untrusted out-of-order sequence numbers.
	ErrTunnelSequenceWindow = errors.New("tunnel sequence exceeds the pending window")

	// ErrTunnelPendingBytes bounds data retained while an earlier frame is
	// missing.
	ErrTunnelPendingBytes = errors.New("tunnel pending data exceeds the byte limit")

	// ErrTunnelIngressLimit bounds all admitted and retained frames for one
	// tunnel generation.
	ErrTunnelIngressLimit = errors.New("tunnel inbound frame limit reached")

	// ErrTunnelAcknowledgement rejects an acknowledgement for data that the
	// server has not assigned yet.
	ErrTunnelAcknowledgement = errors.New("tunnel acknowledgement exceeds the send sequence")
)

const (
	// delayBeforeClose - delay before closing the tunnel.
	// I assume 10 seconds may be an overkill for a good connection, but it looks good enough for less stable one.
	delayBeforeClose = 10 * time.Second

	// The C2 yamux transports admit at most 128 concurrent streams. Matching
	// that window permits legitimate handler reordering while bounding both the
	// pending receive actor and the useful outbound resend history.
	MaxTunnelFrameBytes    = sliverpb.MaxTunnelFrameBytes
	maxTunnelPendingFrames = 128
	maxTunnelPendingBytes  = maxTunnelPendingFrames * MaxTunnelFrameBytes
	maxTunnelResendFrames  = maxTunnelPendingFrames
)

// Tunnel  - Essentially just a mapping between a specific client and sliver
// with an identifier, these tunnels are full duplex. The server doesn't really
// care what data gets passed back and forth it just facilitates the connection
type Tunnel struct {
	ID        uint64
	SessionID string

	ToImplant         chan []byte
	ToImplantSequence uint64
	toImplantQueue    sync.Mutex

	FromImplant          chan *sliverpb.TunnelData
	FromImplantSequence  uint64
	fromImplantLifecycle sync.RWMutex
	fromImplantMutex     sync.Mutex
	fromImplantAdmission chan struct{}
	fromImplantBudget    sync.Mutex
	fromImplantFrames    int
	fromImplantBytes     int
	pendingFromImplant   map[uint64]*sliverpb.TunnelData
	pendingFromBytes     int

	toImplantMutex sync.Mutex
	toImplantAck   uint64
	toImplantCache map[uint64]*sliverpb.TunnelData

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
		ID:                   id,
		SessionID:            sessionID,
		ToImplant:            make(chan []byte),
		FromImplant:          make(chan *sliverpb.TunnelData),
		fromImplantAdmission: make(chan struct{}, maxTunnelPendingFrames),
		pendingFromImplant:   map[uint64]*sliverpb.TunnelData{},
		toImplantCache:       map[uint64]*sliverpb.TunnelData{},

		mutex:               &sync.RWMutex{},
		clientBound:         make(chan struct{}),
		done:                make(chan struct{}),
		lastDataMessageTime: time.Now(), // need to be initialized
	}
}

// ProcessDataFromImplant validates and serializes one generic tunnel frame.
// Reorder and pending-byte state belongs to this exact Tunnel pointer, so a
// retained handler can never poison a newer generation that happens to reuse
// the same numeric ID. Resend controls bypass data sequencing but share the
// same bounded admission and delivery actor.
func (t *Tunnel) ProcessDataFromImplant(tunnelData *sliverpb.TunnelData) error {
	if t == nil || tunnelData == nil {
		return ErrTunnelClosed
	}
	if len(tunnelData.Data) > MaxTunnelFrameBytes {
		return fmt.Errorf("%w: got %d bytes, limit %d", ErrTunnelFrameTooLarge, len(tunnelData.Data), MaxTunnelFrameBytes)
	}
	t.fromImplantLifecycle.RLock()
	defer t.fromImplantLifecycle.RUnlock()
	select {
	case <-t.done:
		return ErrTunnelClosed
	default:
	}
	if err := t.reserveFromImplant(len(tunnelData.Data)); err != nil {
		return err
	}
	reservationTransferred := false
	defer func() {
		if !reservationTransferred {
			t.releaseFromImplant(1, len(tunnelData.Data))
		}
	}()
	select {
	case t.fromImplantAdmission <- struct{}{}:
		defer func() { <-t.fromImplantAdmission }()
	default:
		return ErrTunnelIngressLimit
	}

	t.setLastMessageTime()
	defer t.setLastMessageTime()

	t.fromImplantMutex.Lock()
	defer t.fromImplantMutex.Unlock()
	select {
	case <-t.done:
		return ErrTunnelClosed
	default:
	}

	if tunnelData.Resend {
		return t.deliverFromImplantLocked(t.copyDataFromImplant(tunnelData))
	}

	expected := t.FromImplantSequence
	if tunnelData.Sequence < expected {
		return nil
	}
	if tunnelData.Sequence-expected >= maxTunnelPendingFrames {
		return fmt.Errorf("%w: got %d, expected %d", ErrTunnelSequenceWindow, tunnelData.Sequence, expected)
	}
	if _, duplicate := t.pendingFromImplant[tunnelData.Sequence]; !duplicate {
		if t.pendingFromBytes+len(tunnelData.Data) > maxTunnelPendingBytes {
			return ErrTunnelPendingBytes
		}
		payload := t.copyDataFromImplant(tunnelData)
		t.pendingFromImplant[tunnelData.Sequence] = payload
		t.pendingFromBytes += len(payload.Data)
		reservationTransferred = true
	}

	for {
		payload, ok := t.pendingFromImplant[expected]
		if !ok {
			break
		}
		if err := t.deliverFromImplantLocked(payload); err != nil {
			return err
		}
		delete(t.pendingFromImplant, expected)
		t.pendingFromBytes -= len(payload.Data)
		t.releaseFromImplant(1, len(payload.Data))
		t.FromImplantSequence++
		expected++
	}
	return nil
}

func (t *Tunnel) reserveFromImplant(size int) error {
	t.fromImplantBudget.Lock()
	defer t.fromImplantBudget.Unlock()
	if t.fromImplantFrames >= maxTunnelPendingFrames {
		return ErrTunnelIngressLimit
	}
	if t.fromImplantBytes+size > maxTunnelPendingBytes {
		return ErrTunnelPendingBytes
	}
	t.fromImplantFrames++
	t.fromImplantBytes += size
	return nil
}

func (t *Tunnel) releaseFromImplant(frames int, size int) {
	t.fromImplantBudget.Lock()
	t.fromImplantFrames -= frames
	t.fromImplantBytes -= size
	t.fromImplantBudget.Unlock()
}

// copyDataFromImplant retains only protocol fields consumed by the generic
// tunnel actor. In particular, protobuf unknown fields and caller-controlled
// metadata cannot escape the frame-byte accounting through proto.Clone.
func (t *Tunnel) copyDataFromImplant(tunnelData *sliverpb.TunnelData) *sliverpb.TunnelData {
	return &sliverpb.TunnelData{
		Data:      append([]byte(nil), tunnelData.Data...),
		Closed:    tunnelData.Closed,
		Sequence:  tunnelData.Sequence,
		Ack:       tunnelData.Ack,
		Resend:    tunnelData.Resend,
		TunnelID:  t.ID,
		SessionID: t.SessionID,
	}
}

func (t *Tunnel) deliverFromImplantLocked(tunnelData *sliverpb.TunnelData) error {
	select {
	case <-t.done:
		return ErrTunnelClosed
	default:
	}
	select {
	case t.FromImplant <- tunnelData:
		return nil
	case <-t.done:
		return ErrTunnelClosed
	}
}

// NextDataToImplant assigns one outbound sequence and retains only the history
// that can still be useful to a peer with the advertised 128-frame receive
// window. The returned message is immutable and may be marshaled directly.
func (t *Tunnel) NextDataToImplant(data []byte) (*sliverpb.TunnelData, error) {
	if t == nil {
		return nil, ErrTunnelClosed
	}
	if len(data) > MaxTunnelFrameBytes {
		return nil, fmt.Errorf("%w: got %d bytes, limit %d", ErrTunnelFrameTooLarge, len(data), MaxTunnelFrameBytes)
	}

	t.toImplantMutex.Lock()
	defer t.toImplantMutex.Unlock()
	select {
	case <-t.done:
		return nil, ErrTunnelClosed
	default:
	}

	sequence := t.ToImplantSequence
	tunnelData := &sliverpb.TunnelData{
		Sequence:  sequence,
		TunnelID:  t.ID,
		SessionID: t.SessionID,
		Data:      append([]byte(nil), data...),
		Closed:    false,
	}
	t.ToImplantSequence++
	// Keep the actor's retained copy private from callers that marshal the
	// returned message after this lock is released.
	t.toImplantCache[sequence] = cloneTunnelData(tunnelData)
	if sequence >= maxTunnelResendFrames {
		delete(t.toImplantCache, sequence-maxTunnelResendFrames)
	}
	return tunnelData, nil
}

// AcknowledgeDataToImplant cumulatively retires frames below ack. A future ACK
// is a protocol violation; stale ACKs are harmless and idempotent.
func (t *Tunnel) AcknowledgeDataToImplant(ack uint64) error {
	if t == nil {
		return ErrTunnelClosed
	}
	t.toImplantMutex.Lock()
	defer t.toImplantMutex.Unlock()
	select {
	case <-t.done:
		return ErrTunnelClosed
	default:
	}
	if ack > t.ToImplantSequence {
		return fmt.Errorf("%w: got %d, next sequence %d", ErrTunnelAcknowledgement, ack, t.ToImplantSequence)
	}
	if ack <= t.toImplantAck {
		return nil
	}
	for sequence := range t.toImplantCache {
		if sequence < ack {
			delete(t.toImplantCache, sequence)
		}
	}
	t.toImplantAck = ack
	return nil
}

// ResendDataToImplant returns an immutable cached frame when it remains inside
// the useful receive window. Older evicted requests fail without growing state.
func (t *Tunnel) ResendDataToImplant(sequence uint64) (*sliverpb.TunnelData, bool, error) {
	if t == nil {
		return nil, false, ErrTunnelClosed
	}
	t.toImplantMutex.Lock()
	defer t.toImplantMutex.Unlock()
	select {
	case <-t.done:
		return nil, false, ErrTunnelClosed
	default:
	}
	if sequence >= t.ToImplantSequence {
		return nil, false, fmt.Errorf("%w: resend %d, next sequence %d", ErrTunnelAcknowledgement, sequence, t.ToImplantSequence)
	}
	tunnelData, ok := t.toImplantCache[sequence]
	if !ok {
		return nil, false, nil
	}
	return cloneTunnelData(tunnelData), true, nil
}

func cloneTunnelData(tunnelData *sliverpb.TunnelData) *sliverpb.TunnelData {
	return &sliverpb.TunnelData{
		Data:          append([]byte(nil), tunnelData.Data...),
		Closed:        tunnelData.Closed,
		Sequence:      tunnelData.Sequence,
		Ack:           tunnelData.Ack,
		Resend:        tunnelData.Resend,
		CreateReverse: tunnelData.CreateReverse,
		TunnelID:      tunnelData.TunnelID,
		SessionID:     tunnelData.SessionID,
	}
}

// BindClient reserves this tunnel for the first client stream.
func (t *Tunnel) BindClient(client rpcpb.SliverRPC_TunnelDataServer) bool {
	t.clientMutex.Lock()
	defer t.clientMutex.Unlock()
	select {
	case <-t.done:
		return false
	default:
	}
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
	select {
	case <-t.done:
		return false
	default:
	}
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

// SendDataFromImplant forwards tunnelData to the client stream and reports
// whether the send was accepted rather than canceled by tunnel closure.
func (t *Tunnel) SendDataFromImplant(tunnelData *sliverpb.TunnelData) bool {
	return t.ProcessDataFromImplant(tunnelData) == nil
}

// SendDataToImplant queues tunnel data unless the tunnel has already closed.
func (t *Tunnel) SendDataToImplant(data []byte) bool {
	t.toImplantQueue.Lock()
	defer t.toImplantQueue.Unlock()

	select {
	case <-t.done:
		return false
	default:
	}
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

// Close publishes Done, which unblocks channel operations, then synchronously
// joins and clears this generation's protocol actors. It does not depend on an
// external client or implant reader making progress.
func (t *Tunnel) Close() {
	t.closeOnce.Do(func() {
		close(t.done)
		t.clearProtocolState()
	})
}

func (t *Tunnel) clearProtocolState() {
	// Done is already closed, so blocked actor operations wake without needing a
	// consumer. Waiting for their locks here guarantees no state mutation or
	// channel delivery can outlive registry detachment.
	t.fromImplantLifecycle.Lock()
	t.fromImplantMutex.Lock()
	pendingFrames := len(t.pendingFromImplant)
	pendingBytes := t.pendingFromBytes
	clear(t.pendingFromImplant)
	t.pendingFromBytes = 0
	t.fromImplantMutex.Unlock()
	t.releaseFromImplant(pendingFrames, pendingBytes)
	t.fromImplantLifecycle.Unlock()

	t.toImplantMutex.Lock()
	clear(t.toImplantCache)
	t.toImplantMutex.Unlock()

	t.toImplantQueue.Lock()
	t.toImplantQueue.Unlock()

	t.clientMutex.Lock()
	t.clientMutex.Unlock()
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
	t.ScheduleCloseTunnel(tunnel)
}

// ScheduleCloseTunnel schedules a quiet-period close for one exact generation.
// A delayed scheduler retained from an older generation must never close a
// newer tunnel that happens to reuse the same numeric ID.
func (t *tunnels) ScheduleCloseTunnel(tunnel *Tunnel) {
	if tunnel == nil || t.Get(tunnel.ID) != tunnel {
		return
	}

	timeDelta := time.Since(tunnel.GetLastMessageTime())

	coreLog.Printf("Scheduled close for channel %d (delta: %v)", tunnel.ID, timeDelta)

	if timeDelta >= delayBeforeClose {
		coreLog.Printf("Closing channel %d", tunnel.ID)
		t.CloseIf(tunnel)
	} else {
		// Reschedule
		coreLog.Printf("Rescheduling closing channel %d", tunnel.ID)
		time.Sleep(delayBeforeClose - timeDelta + time.Second)
		go t.ScheduleCloseTunnel(tunnel)
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
	// Publish and join the actor's closed state before detaching it. A handler
	// that retained this pointer before registry removal must observe Done and
	// cannot mutate a detached generation.
	tunnel.Close()
	delete(t.tunnels, tunnelID)
	t.mutex.Unlock()
	return nil
}

// CloseIf atomically detaches and closes only the expected tunnel generation.
func (t *tunnels) CloseIf(tunnel *Tunnel) bool {
	if tunnel == nil {
		return false
	}
	t.mutex.Lock()
	if t.tunnels[tunnel.ID] != tunnel {
		t.mutex.Unlock()
		return false
	}
	// Publish closed state while the generation is still protected by the
	// registry lock, then detach it atomically from future lookups.
	tunnel.Close()
	delete(t.tunnels, tunnel.ID)
	t.mutex.Unlock()
	return true
}

// CloseForClient closes every tunnel owned by a terminated client stream.
// Closed state is published and its bounded actor operations are joined under
// the registry lock before each generation is detached.
func (t *tunnels) CloseForClient(client rpcpb.SliverRPC_TunnelDataServer) {
	if client == nil {
		return
	}

	t.mutex.Lock()
	for tunnelID, tunnel := range t.tunnels {
		if tunnel.IsClient(client) {
			tunnel.Close()
			delete(t.tunnels, tunnelID)
		}
	}
	t.mutex.Unlock()
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
