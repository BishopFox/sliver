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
	"bytes"
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

	// ErrTunnelSequenceConflict rejects two different payloads claiming the
	// same inbound sequence number.
	ErrTunnelSequenceConflict = errors.New("tunnel sequence conflicts with an accepted frame")

	// ErrTunnelPendingBytes bounds data retained while an earlier frame is
	// missing.
	ErrTunnelPendingBytes = errors.New("tunnel pending data exceeds the byte limit")

	// ErrTunnelIngressLimit bounds all admitted and retained frames for one
	// tunnel generation.
	ErrTunnelIngressLimit = errors.New("tunnel inbound frame limit reached")

	// ErrTunnelAcknowledgement rejects an acknowledgement for data that the
	// server has not assigned yet.
	ErrTunnelAcknowledgement = errors.New("tunnel acknowledgement exceeds the send sequence")

	// ErrTunnelTerminal rejects contradictory terminal sequence state or data
	// at or beyond an accepted exclusive terminal sequence.
	ErrTunnelTerminal = errors.New("tunnel terminal sequence is invalid")
)

const (
	// delayBeforeClose - delay before closing the tunnel.
	// I assume 10 seconds may be an overkill for a good connection, but it looks good enough for less stable one.
	delayBeforeClose = 10 * time.Second

	// tunnelClientBindLease bounds how long a tunnel created by the unary RPC
	// may remain in the registry without completing its client-stream bind.
	tunnelClientBindLease = 30 * time.Second
	// tunnelTerminalCloseTimeout fails the exact implant connection closed when
	// a capability-bearing terminal never receives every preceding data frame.
	tunnelTerminalCloseTimeout = 10 * time.Second

	// MaxTunnelFrameBytes limits one generic tunnel data frame.
	MaxTunnelFrameBytes = sliverpb.MaxTunnelFrameBytes
	// The C2 yamux transports admit at most 128 concurrent streams. Matching
	// that window permits legitimate handler reordering while bounding both the
	// pending receive actor and the useful outbound resend history.
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
	// implantConnection is the immutable transport generation that owned this
	// tunnel when it was created. Session registry entries can disappear or be
	// replaced while relay workers are still finalizing the exact tunnel.
	implantConnection *ImplantConnection
	capabilities      uint64

	ToImplant         chan []byte
	ToImplantSequence uint64
	toImplantQueue    sync.Mutex
	toImplantPending  sync.WaitGroup
	toImplantClosing  bool

	FromImplant                  chan *sliverpb.TunnelData
	FromImplantSequence          uint64
	fromImplantLifecycle         sync.RWMutex
	fromImplantMutex             sync.Mutex
	fromImplantAdmission         chan struct{}
	fromImplantBudget            sync.Mutex
	fromImplantFrames            int
	fromImplantBytes             int
	pendingFromImplant           map[uint64]*sliverpb.TunnelData
	pendingFromBytes             int
	fromImplantTerminalSet       bool
	fromImplantTerminalSequence  uint64
	fromImplantTerminalMutex     sync.Mutex
	fromImplantForwardedSequence uint64
	fromImplantTerminalReady     chan struct{}
	fromImplantTerminalReadyOnce sync.Once
	fromImplantTerminalWaitOnce  sync.Once

	toImplantMutex             sync.Mutex
	toImplantAck               uint64
	toImplantForwardedSequence uint64
	toImplantCache             map[uint64]*sliverpb.TunnelData

	Client rpcpb.SliverRPC_TunnelDataServer

	mutex                *sync.RWMutex
	clientMutex          sync.RWMutex
	clientBound          chan struct{}
	clientBoundOnce      sync.Once
	clientBindExpired    bool
	closeOnce            sync.Once
	toImplantCloseOnce   sync.Once
	fromImplantCloseOnce sync.Once
	implantTerminalOnce  sync.Once
	clientTerminalOnce   sync.Once
	done                 chan struct{}
	lastToImplantTime    time.Time
	lastFromImplantTime  time.Time
}

func NewTunnel(id uint64, sessionID string) *Tunnel {
	return newTunnel(id, sessionID, nil)
}

func newTunnel(id uint64, sessionID string, implantConnection *ImplantConnection) *Tunnel {
	return newTunnelWithCapabilities(id, sessionID, implantConnection, 0)
}

func newTunnelWithCapabilities(id uint64, sessionID string, implantConnection *ImplantConnection, capabilities uint64) *Tunnel {
	createdAt := time.Now()
	return &Tunnel{
		ID:                       id,
		SessionID:                sessionID,
		implantConnection:        implantConnection,
		capabilities:             capabilities & sliverpb.CapabilityTunnelTerminalV1,
		ToImplant:                make(chan []byte),
		FromImplant:              make(chan *sliverpb.TunnelData),
		fromImplantAdmission:     make(chan struct{}, maxTunnelPendingFrames),
		pendingFromImplant:       map[uint64]*sliverpb.TunnelData{},
		fromImplantTerminalReady: make(chan struct{}),
		toImplantCache:           map[uint64]*sliverpb.TunnelData{},

		mutex:               &sync.RWMutex{},
		clientBound:         make(chan struct{}),
		done:                make(chan struct{}),
		lastToImplantTime:   createdAt,
		lastFromImplantTime: createdAt,
	}
}

// ImplantConnection returns the exact transport generation that owned this
// tunnel at creation. The association is immutable after publication.
func (t *Tunnel) ImplantConnection() *ImplantConnection {
	if t == nil {
		return nil
	}
	return t.implantConnection
}

// TunnelTerminalEnabled reports whether the exact implant generation waits for
// an exclusive terminal sequence before detaching a generic tunnel.
func (t *Tunnel) TunnelTerminalEnabled() bool {
	if t == nil {
		return false
	}
	return t.capabilities&sliverpb.CapabilityTunnelTerminalV1 != 0
}

// ToImplantTerminalSequence returns the exclusive successfully-enqueued prefix
// understood by a capability-bearing implant. An assigned frame whose bounded
// transport send failed must not make the implant wait for data that was never
// enqueued. Legacy implants require sequence zero.
func (t *Tunnel) ToImplantTerminalSequence() uint64 {
	if !t.TunnelTerminalEnabled() {
		return 0
	}
	t.toImplantMutex.Lock()
	defer t.toImplantMutex.Unlock()
	return t.toImplantForwardedSequence
}

// CompleteDataToImplantForward advances the contiguous prefix after the exact
// frame has been accepted by the implant transport. The tunnel has one
// client-to-implant forwarding worker, so completion must remain sequential.
func (t *Tunnel) CompleteDataToImplantForward(sequence uint64) error {
	if t == nil {
		return ErrTunnelClosed
	}
	t.toImplantMutex.Lock()
	defer t.toImplantMutex.Unlock()
	if sequence != t.toImplantForwardedSequence || sequence >= t.ToImplantSequence {
		return fmt.Errorf("%w: completed sequence %d, forwarded %d, assigned %d", ErrTunnelSequenceConflict, sequence, t.toImplantForwardedSequence, t.ToImplantSequence)
	}
	t.toImplantForwardedSequence++
	return nil
}

// ClaimImplantTerminalDelivery lets one concurrent tunnel owner publish the
// exact generation's terminal to its implant peer.
func (t *Tunnel) ClaimImplantTerminalDelivery() bool {
	if t == nil {
		return false
	}
	claimed := false
	t.implantTerminalOnce.Do(func() { claimed = true })
	return claimed
}

// ClaimClientTerminalDelivery lets one concurrent tunnel owner publish the
// exact generation's terminal to its operator peer. Once a client owns the
// tunnel, a different stream that merely retained the pointer cannot consume
// the owner's terminal claim. An unbound racing client may still be notified.
func (t *Tunnel) ClaimClientTerminalDelivery(client rpcpb.SliverRPC_TunnelDataServer) bool {
	if t == nil || client == nil {
		return false
	}
	t.clientMutex.Lock()
	defer t.clientMutex.Unlock()
	if t.Client != nil && t.Client != client {
		return false
	}
	claimed := false
	t.clientTerminalOnce.Do(func() { claimed = true })
	return claimed
}

// ProcessDataFromImplant validates and serializes one generic tunnel frame.
// Reorder and pending-byte state belongs to this exact Tunnel pointer, so a
// retained handler can never poison a newer generation that happens to reuse
// the same numeric ID. Resend controls bypass data sequencing but share the
// same bounded admission and delivery actor.
//
//nolint:gocyclo // Admission, sequencing, resend controls, and bounded delivery form one state transition.
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

	t.touchFromImplant()
	defer t.touchFromImplant()

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
	t.fromImplantTerminalMutex.Lock()
	terminalSet := t.fromImplantTerminalSet
	terminalSequence := t.fromImplantTerminalSequence
	t.fromImplantTerminalMutex.Unlock()
	if terminalSet && tunnelData.Sequence >= terminalSequence {
		return fmt.Errorf("%w: data sequence %d, terminal %d", ErrTunnelTerminal, tunnelData.Sequence, terminalSequence)
	}
	if tunnelData.Sequence < expected {
		return nil
	}
	if tunnelData.Sequence-expected >= maxTunnelPendingFrames {
		return fmt.Errorf("%w: got %d, expected %d", ErrTunnelSequenceWindow, tunnelData.Sequence, expected)
	}
	if existing := t.pendingFromImplant[tunnelData.Sequence]; existing != nil {
		if existing.Ack != tunnelData.Ack || existing.Closed != tunnelData.Closed ||
			existing.Resend != tunnelData.Resend || !bytes.Equal(existing.Data, tunnelData.Data) {
			return fmt.Errorf("%w: sequence %d", ErrTunnelSequenceConflict, tunnelData.Sequence)
		}
	} else {
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

// MarkFromImplantTerminal records the exclusive final data sequence supplied
// by a capability-bearing implant. It serializes with data admission so a
// terminal can neither contradict retained data nor race a frame at or beyond
// the terminal boundary.
//
//nolint:gocyclo // Terminal validation and exact-generation lifecycle checks form one state transition.
func (t *Tunnel) MarkFromImplantTerminal(terminal *sliverpb.TunnelData) (bool, error) {
	if t == nil || terminal == nil {
		return false, ErrTunnelClosed
	}
	if !terminal.Closed || len(terminal.Data) != 0 || terminal.Ack != 0 || terminal.Resend ||
		terminal.CreateReverse || terminal.Rportfwd != nil {
		return false, ErrTunnelTerminal
	}

	t.fromImplantLifecycle.RLock()
	defer t.fromImplantLifecycle.RUnlock()
	t.fromImplantMutex.Lock()
	defer t.fromImplantMutex.Unlock()
	select {
	case <-t.done:
		return true, ErrTunnelClosed
	default:
	}

	expected := t.FromImplantSequence
	sequence := terminal.Sequence
	if sequence < expected || sequence-expected > maxTunnelPendingFrames {
		return false, fmt.Errorf("%w: terminal %d, expected %d", ErrTunnelTerminal, sequence, expected)
	}
	t.fromImplantTerminalMutex.Lock()
	defer t.fromImplantTerminalMutex.Unlock()
	if t.fromImplantTerminalSet {
		if t.fromImplantTerminalSequence != sequence {
			return false, fmt.Errorf("%w: terminal changed from %d to %d", ErrTunnelTerminal, t.fromImplantTerminalSequence, sequence)
		}
		t.signalFromImplantTerminalReadyLocked()
		return t.fromImplantForwardedSequence >= sequence, nil
	}
	for pendingSequence := range t.pendingFromImplant {
		if pendingSequence >= sequence {
			return false, fmt.Errorf("%w: retained data sequence %d at terminal %d", ErrTunnelTerminal, pendingSequence, sequence)
		}
	}
	t.fromImplantTerminalSet = true
	t.fromImplantTerminalSequence = sequence
	t.signalFromImplantTerminalReadyLocked()
	return t.fromImplantForwardedSequence >= sequence, nil
}

func (t *Tunnel) signalFromImplantTerminalReadyLocked() {
	if t.fromImplantTerminalSet && t.fromImplantForwardedSequence >= t.fromImplantTerminalSequence {
		t.fromImplantTerminalReadyOnce.Do(func() { close(t.fromImplantTerminalReady) })
	}
}

// FromImplantTerminalReady closes after every sequence below the accepted
// capability-bearing terminal has been sent successfully to the operator.
func (t *Tunnel) FromImplantTerminalReady() <-chan struct{} {
	if t == nil {
		return nil
	}
	return t.fromImplantTerminalReady
}

// CompleteDataFromImplantForward records one ordered frame after the operator
// stream Send has succeeded. It intentionally does not take fromImplantMutex:
// ProcessDataFromImplant holds that mutex while handing frames to the unbuffered
// worker channel, so coupling completion to the producer lock would deadlock
// whenever more than one queued frame drains in a single pass.
func (t *Tunnel) CompleteDataFromImplantForward(sequence uint64) error {
	if t == nil {
		return ErrTunnelClosed
	}
	t.fromImplantTerminalMutex.Lock()
	defer t.fromImplantTerminalMutex.Unlock()
	select {
	case <-t.done:
		return ErrTunnelClosed
	default:
	}
	if sequence != t.fromImplantForwardedSequence {
		return fmt.Errorf("%w: forwarded %d, expected %d", ErrTunnelSequenceConflict, sequence, t.fromImplantForwardedSequence)
	}
	if t.fromImplantTerminalSet && sequence >= t.fromImplantTerminalSequence {
		return fmt.Errorf("%w: forwarded data sequence %d, terminal %d", ErrTunnelTerminal, sequence, t.fromImplantTerminalSequence)
	}
	t.fromImplantForwardedSequence++
	t.signalFromImplantTerminalReadyLocked()
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
	if t.clientBindExpired || t.Client != nil {
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
	if t.clientBindExpired || t.Client != client {
		return false
	}

	t.clientBoundOnce.Do(func() { close(t.clientBound) })
	return true
}

// expireClientBindLease atomically wins or loses against client reservation
// and acknowledgement. Once expiry wins, a concurrently received bind frame
// cannot revive this tunnel while its exact registry generation is detached.
func (t *Tunnel) expireClientBindLease() bool {
	t.clientMutex.Lock()
	defer t.clientMutex.Unlock()
	select {
	case <-t.done:
		return false
	default:
	}
	select {
	case <-t.clientBound:
		return false
	default:
	}
	if t.clientBindExpired {
		return false
	}
	t.clientBindExpired = true
	return true
}

// ClientBindLeaseExpired reports whether bind expiry won before the client
// completed its stream acknowledgement.
func (t *Tunnel) ClientBindLeaseExpired() bool {
	t.clientMutex.RLock()
	defer t.clientMutex.RUnlock()
	return t.clientBindExpired
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

func (t *Tunnel) touchToImplant() {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.lastToImplantTime = time.Now()
}

func (t *Tunnel) touchFromImplant() {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.lastFromImplantTime = time.Now()
}

// TouchToImplant starts a fresh grace period for a client-requested close. Only
// later client-to-implant data may extend that grace period; target keepalives
// traveling in the opposite direction must not keep the target socket alive.
func (t *Tunnel) TouchToImplant() {
	t.touchToImplant()
}

// ClaimToImplantClose records the first client close request for this exact
// tunnel generation. Duplicate unary requests must not refresh the quiet
// period or create additional close schedulers.
func (t *Tunnel) ClaimToImplantClose() bool {
	if t == nil {
		return false
	}
	claimed := false
	t.toImplantCloseOnce.Do(func() {
		t.touchToImplant()
		claimed = true
	})
	return claimed
}

// TouchFromImplant starts a fresh grace period for an implant terminal close.
// Later implant-to-client data may extend it while concurrently-dispatched
// terminal and data envelopes are reordered.
func (t *Tunnel) TouchFromImplant() {
	t.touchFromImplant()
}

// ClaimFromImplantClose records the first terminal frame for this exact tunnel
// generation. Duplicate terminal envelopes must not refresh the quiet period
// or create additional close schedulers.
func (t *Tunnel) ClaimFromImplantClose() bool {
	if t == nil {
		return false
	}
	claimed := false
	t.fromImplantCloseOnce.Do(func() {
		t.touchFromImplant()
		claimed = true
	})
	return claimed
}

// Touch records implant-to-client activity for the legacy implant-originated
// close path.
//
// Deprecated: use TouchToImplant or TouchFromImplant explicitly.
func (t *Tunnel) Touch() {
	t.TouchFromImplant()
}

// LastToImplantTime returns the last client-to-implant activity used to delay a
// client-requested close.
func (t *Tunnel) LastToImplantTime() time.Time {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	return t.lastToImplantTime
}

// LastFromImplantTime returns the last implant-to-client activity used to
// delay an implant terminal close.
func (t *Tunnel) LastFromImplantTime() time.Time {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	return t.lastFromImplantTime
}

// GetLastMessageTime returns implant-to-client activity for the legacy
// implant-originated close path.
//
// Deprecated: use LastToImplantTime or LastFromImplantTime explicitly.
func (t *Tunnel) GetLastMessageTime() time.Time {
	return t.LastFromImplantTime()
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

	if t.toImplantClosing {
		return false
	}
	select {
	case <-t.done:
		return false
	default:
	}
	t.touchToImplant()
	defer t.touchToImplant()
	t.toImplantPending.Add(1)
	transferred := false
	defer func() {
		if !transferred {
			t.toImplantPending.Done()
		}
	}()

	select {
	case t.ToImplant <- data:
		transferred = true
		return true
	case <-t.done:
		return false
	}
}

// CompleteDataToImplant releases one client-to-implant forwarding reservation.
// The TunnelData worker calls it only after the corresponding bounded implant
// send has completed or failed.
func (t *Tunnel) CompleteDataToImplant() {
	if t == nil {
		return
	}
	t.toImplantPending.Done()
}

// QuiesceDataToImplant prevents new client payload admission and joins every
// payload already handed to the TunnelData forwarding worker. It is used only
// for a graceful client-requested close; failure and session teardown still
// publish Done immediately so bounded sends are canceled promptly.
func (t *Tunnel) QuiesceDataToImplant() {
	if t == nil {
		return
	}
	t.toImplantQueue.Lock()
	t.toImplantClosing = true
	t.toImplantQueue.Unlock()
	t.toImplantPending.Wait()
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
	// Completion deliberately cannot take fromImplantLifecycle while the
	// producer is handing a batch to the unbuffered worker channel. Join that
	// independent post-Send state here so no completion mutation outlives Close.
	t.fromImplantTerminalMutex.Lock()
	t.fromImplantTerminalSet = false
	t.fromImplantTerminalSequence = 0
	t.fromImplantForwardedSequence = 0
	t.fromImplantTerminalMutex.Unlock()

	t.toImplantMutex.Lock()
	clear(t.toImplantCache)
	t.toImplantMutex.Unlock()

	t.toImplantQueue.Lock()
	t.toImplantQueue.Unlock() //nolint:staticcheck // Intentional barrier joins any in-flight outbound producer.

	t.clientMutex.Lock()
	t.clientMutex.Unlock() //nolint:staticcheck // Intentional barrier joins any in-flight client binding.
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

	tunnel := newTunnelWithCapabilities(
		tunnelID,
		session.ID,
		session.Connection,
		session.Capabilities,
	)

	t.mutex.Lock()
	t.tunnels[tunnel.ID] = tunnel
	t.mutex.Unlock()

	// Session removal can race the gap between the initial lookup and registry
	// insertion. Revalidate the exact session generation after publication so a
	// tunnel can never appear after CloseForSession already finished.
	if Sessions.Get(sessionID) != session {
		t.CloseIf(tunnel)
		return nil, ErrInvalidSessionID
	}
	t.armImplantConnectionWatcher(tunnel)
	t.armClientBindLease(tunnel)

	return tunnel, nil
}

func (t *tunnels) armImplantConnectionWatcher(tunnel *Tunnel) {
	if tunnel == nil || tunnel.ImplantConnection() == nil {
		return
	}
	go t.waitForImplantConnection(tunnel)
}

func (t *tunnels) waitForImplantConnection(tunnel *Tunnel) {
	if tunnel == nil {
		return
	}
	connection := tunnel.ImplantConnection()
	if connection == nil {
		return
	}
	select {
	case <-tunnel.Done():
		return
	case <-connection.Done():
		// CloseIf protects a same-ID replacement in the registry. If another
		// generation already replaced this pointer, still retire the detached old
		// object so none of its workers or pending data outlive the connection.
		if !t.CloseIf(tunnel) {
			tunnel.Close()
		}
	}
}

func (t *tunnels) armClientBindLease(tunnel *Tunnel) {
	go func() {
		timer := time.NewTimer(tunnelClientBindLease)
		defer timer.Stop()
		t.waitForClientBindLease(tunnel, timer.C)
	}()
}

// waitForClientBindLease accepts an expiry channel so tests can drive the
// transition synchronously without changing the production timeout.
func (t *tunnels) waitForClientBindLease(tunnel *Tunnel, expiry <-chan time.Time) {
	if tunnel == nil || expiry == nil {
		return
	}
	select {
	case <-tunnel.ClientBound():
		return
	case <-tunnel.Done():
		return
	case <-expiry:
		if tunnel.expireClientBindLease() {
			t.CloseIf(tunnel)
		}
	}
}

// ScheduleClose schedules an implant-originated close for a tunnel. Retain this
// ID-based entry point for callers that do not already hold the exact tunnel
// generation.
func (t *tunnels) ScheduleClose(tunnelID uint64) {
	tunnel := t.Get(tunnelID)
	if tunnel == nil {
		return
	}
	t.ScheduleCloseTunnelFromImplant(tunnel)
}

// ScheduleCloseTunnelToImplant delays a client-requested close only for
// client-to-implant traffic that the unary CloseTunnel RPC may have overtaken.
func (t *tunnels) ScheduleCloseTunnelToImplant(tunnel *Tunnel) {
	if tunnel == nil {
		return
	}
	t.scheduleCloseTunnel(tunnel, "to implant", tunnel.LastToImplantTime, tunnel.QuiesceDataToImplant)
}

// ScheduleCloseTunnelFromImplant delays an implant terminal close only for
// implant-to-client data whose independently-dispatched envelope it may have
// overtaken.
func (t *tunnels) ScheduleCloseTunnelFromImplant(tunnel *Tunnel) {
	if tunnel == nil {
		return
	}
	t.scheduleCloseTunnel(tunnel, "from implant", tunnel.LastFromImplantTime, nil)
}

// ArmFromImplantTerminalClose owns the bounded close actor for one accepted
// capability-bearing terminal. Complete ordering closes the tunnel promptly;
// an incomplete terminal fails its exact creating connection closed.
func (t *tunnels) ArmFromImplantTerminalClose(tunnel *Tunnel) {
	t.armFromImplantTerminalClose(tunnel, nil)
}

// armFromImplantTerminalClose accepts an expiry channel for deterministic
// package tests. A nil channel uses the production timeout.
func (t *tunnels) armFromImplantTerminalClose(tunnel *Tunnel, expiry <-chan time.Time) {
	if tunnel == nil {
		return
	}
	tunnel.fromImplantTerminalWaitOnce.Do(func() {
		go func() {
			var timer *time.Timer
			if expiry == nil {
				timer = time.NewTimer(tunnelTerminalCloseTimeout)
				expiry = timer.C
			}
			if timer != nil {
				defer timer.Stop()
			}
			select {
			case <-tunnel.Done():
				return
			case <-tunnel.FromImplantTerminalReady():
				t.CloseIf(tunnel)
			case <-expiry:
				select {
				case <-tunnel.Done():
					return
				default:
				}
				if !t.CloseIf(tunnel) {
					tunnel.Close()
				}
				if connection := tunnel.ImplantConnection(); connection != nil {
					connection.Close()
				}
			}
		}()
	})
}

// ScheduleCloseTunnel schedules the legacy implant-originated close path.
//
// Deprecated: use ScheduleCloseTunnelToImplant or
// ScheduleCloseTunnelFromImplant explicitly.
func (t *tunnels) ScheduleCloseTunnel(tunnel *Tunnel) {
	t.ScheduleCloseTunnelFromImplant(tunnel)
}

// scheduleCloseTunnel waits for the selected direction's quiet period on one
// exact generation. A delayed scheduler retained from an older generation must
// never close a newer tunnel that happens to reuse the same numeric ID.
func (t *tunnels) scheduleCloseTunnel(tunnel *Tunnel, direction string, lastActivity func() time.Time, beforeClose func()) {
	for tunnel != nil && t.Get(tunnel.ID) == tunnel {
		timeDelta := time.Since(lastActivity())
		coreLog.Printf("Scheduled close for channel %d after %s activity (delta: %v)", tunnel.ID, direction, timeDelta)

		if timeDelta >= delayBeforeClose {
			coreLog.Printf("Closing channel %d", tunnel.ID)
			if beforeClose != nil {
				beforeClose()
			}
			t.CloseIf(tunnel)
			return
		}

		coreLog.Printf("Rescheduling closing channel %d", tunnel.ID)
		timer := time.NewTimer(delayBeforeClose - timeDelta + time.Second)
		select {
		case <-timer.C:
		case <-tunnel.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return
		}
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

// CloseForSession closes every generic tunnel owned by a disconnected
// session. The operator TunnelData stream is shared across sessions, so losing
// one implant must close only that session's relays and leave the stream and
// unrelated tunnels intact.
func (t *tunnels) CloseForSession(sessionID string) int {
	if sessionID == "" {
		return 0
	}
	t.mutex.Lock()
	defer t.mutex.Unlock()
	closed := 0
	for tunnelID, tunnel := range t.tunnels {
		if tunnel.SessionID != sessionID {
			continue
		}
		tunnel.Close()
		delete(t.tunnels, tunnelID)
		closed++
	}
	return closed
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
