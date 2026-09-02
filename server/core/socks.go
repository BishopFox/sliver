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
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

var (
	// SocksTunnels manages server-side duplex SOCKS tunnels.
	SocksTunnels = tcpTunnel{
		tunnels: map[uint64]*TcpTunnel{},
		mutex:   &sync.RWMutex{},
	}
)

const (
	// MaxSocksFrameBytes is the per-frame SOCKS payload limit in either
	// direction. It intentionally matches the generic tunnel wire contract.
	MaxSocksFrameBytes = sliverpb.MaxTunnelFrameBytes

	// A tunnel may retain at most one C2 transport window. The byte budget is
	// shared by frames waiting on a sequence gap, frames queued to a worker, and
	// the single frame currently being sent by that worker.
	maxSocksPendingFrames   = 128
	maxSocksPendingBytes    = maxSocksPendingFrames * MaxSocksFrameBytes
	maxSocksCredentialBytes = 255
	// Bound the CreateSocks-to-first-payload lease by count as well as time.
	// This prevents a valid but buggy or hostile operator from accumulating an
	// arbitrary number of short-lived, pre-greeting tunnels for one session.
	maxSocksTunnelsPerSession = 256
)

var (
	// ErrSocksSequenceConflict reports a duplicate sequence with different content.
	ErrSocksSequenceConflict = errors.New("conflicting duplicate SOCKS sequence")
	// ErrSocksTerminalPayload reports a terminal frame containing payload data.
	ErrSocksTerminalPayload = errors.New("SOCKS terminal frame carries data")
	// ErrSocksCredentialSize reports credentials exceeding the protocol limit.
	ErrSocksCredentialSize = errors.New("SOCKS credential exceeds the size limit")
	// ErrSocksTunnelLimit reports exhaustion of a session's SOCKS tunnel quota.
	ErrSocksTunnelLimit = errors.New("SOCKS tunnel limit reached for session")
	// ErrSocksCapabilityMismatch rejects a bind that does not echo the exact
	// capability set negotiated by CreateSocks.
	ErrSocksCapabilityMismatch = errors.New("SOCKS capability negotiation mismatch")
	// ErrSocksFlowControl reports an acknowledgement on a tunnel that did not
	// negotiate cumulative flow control.
	ErrSocksFlowControl = errors.New("SOCKS flow control was not negotiated")
	// ErrSocksAcknowledgement rejects zero or future cumulative acknowledgements.
	ErrSocksAcknowledgement = errors.New("SOCKS acknowledgement exceeds the sent sequence")
	// ErrSocksOwner rejects a flow-control message from a different tunnel owner.
	ErrSocksOwner = errors.New("SOCKS flow-control owner mismatch")
)

// socksAckMailbox retains only the greatest pending cumulative acknowledgement.
// Its fixed one-element channel makes ACK bursts O(1) per tunnel while allowing
// the directional relay worker to select controls independently of payloads.
type socksAckMailbox struct {
	mu      sync.Mutex
	latest  uint64
	pending chan uint64
}

func newSocksAckMailbox() *socksAckMailbox {
	return &socksAckMailbox{pending: make(chan uint64, 1)}
}

func (mailbox *socksAckMailbox) offer(ack uint64) {
	mailbox.mu.Lock()
	defer mailbox.mu.Unlock()
	if ack <= mailbox.latest {
		return
	}
	mailbox.latest = ack
	select {
	case <-mailbox.pending:
	default:
	}
	mailbox.pending <- ack
}

func (mailbox *socksAckMailbox) channel() <-chan uint64 {
	return mailbox.pending
}

// socksFrameQueue is one direction of the SOCKS framing protocol. All state
// belongs to an exact TcpTunnel pointer rather than a process-global tunnel-ID
// cache, so delayed work cannot poison a later generation reusing an ID.
type socksFrameQueue struct {
	mu             sync.Mutex
	nextSequence   uint64
	pending        map[uint64]*sliverpb.SocksData
	ready          chan *sliverpb.SocksData
	accepted       map[uint64]*sliverpb.SocksData
	reservations   map[uint64]int
	reservedFrames int
	reservedBytes  int
	maxFrames      int
	maxBytes       int
	spaceChanged   chan struct{}
}

func newSocksFrameQueue(maxFrames int, maxBytes int) *socksFrameQueue {
	return &socksFrameQueue{
		pending:      map[uint64]*sliverpb.SocksData{},
		ready:        make(chan *sliverpb.SocksData, maxFrames),
		accepted:     map[uint64]*sliverpb.SocksData{},
		reservations: map[uint64]int{},
		maxFrames:    maxFrames,
		maxBytes:     maxBytes,
		spaceChanged: make(chan struct{}),
	}
}

// spaceChange returns the current generation of the queue's capacity signal.
// Callers take this snapshot before attempting admission so a concurrent
// completion cannot be missed between observing pressure and beginning to wait.
func (q *socksFrameQueue) spaceChange() <-chan struct{} {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.spaceChanged
}

// signalSpaceChangeLocked broadcasts a reservation change to every waiter.
// The replacement channel prevents one waiter from consuming the only wakeup
// while capacity remains available for another waiter.
func (q *socksFrameQueue) signalSpaceChangeLocked() {
	close(q.spaceChanged)
	q.spaceChanged = make(chan struct{})
}

func validateSocksFrame(data *sliverpb.SocksData) error {
	if data == nil {
		return ErrTunnelClosed
	}
	if len(data.Data) > MaxSocksFrameBytes {
		return fmt.Errorf("%w: got %d bytes, limit %d", ErrTunnelFrameTooLarge, len(data.Data), MaxSocksFrameBytes)
	}
	if data.CloseConn && len(data.Data) != 0 {
		return ErrSocksTerminalPayload
	}
	return nil
}

func (q *socksFrameQueue) admit(tunnelID uint64, data *sliverpb.SocksData) error {
	return q.admitWithSequence(tunnelID, data, false)
}

func (q *socksFrameQueue) admitLegacyTerminal(tunnelID uint64, data *sliverpb.SocksData) error {
	return q.admitWithSequence(tunnelID, data, true)
}

func (q *socksFrameQueue) admitWithSequence(tunnelID uint64, data *sliverpb.SocksData, useNextSequence bool) error {
	if err := validateSocksFrame(data); err != nil {
		return err
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	sequence := data.Sequence
	if useNextSequence {
		sequence = q.nextSequence
	}
	if existing := q.accepted[sequence]; existing != nil {
		if equalSocksProtocolFrame(existing, data) {
			return nil
		}
		return fmt.Errorf("%w: sequence %d", ErrSocksSequenceConflict, sequence)
	}
	if sequence < q.nextSequence {
		// The accepted copy is retired only after the worker finishes with it,
		// so a sequence below nextSequence with no accepted entry is a genuinely
		// stale replay that cannot change retained state.
		return nil
	}
	if sequence-q.nextSequence >= uint64(q.maxFrames) {
		return fmt.Errorf("%w: got %d, expected %d", ErrTunnelSequenceWindow, sequence, q.nextSequence)
	}
	if q.reservedFrames >= q.maxFrames {
		return ErrTunnelIngressLimit
	}
	if q.reservedBytes+len(data.Data) > q.maxBytes {
		return ErrTunnelPendingBytes
	}

	payload := copySocksProtocolFrame(tunnelID, data)
	payload.Sequence = sequence
	q.pending[payload.Sequence] = payload
	q.accepted[payload.Sequence] = payload
	q.reservations[payload.Sequence] = len(payload.Data)
	q.reservedFrames++
	q.reservedBytes += len(payload.Data)
	for {
		ordered := q.pending[q.nextSequence]
		if ordered == nil {
			break
		}
		delete(q.pending, q.nextSequence)
		// The ready channel has the same capacity as the total reservation
		// window, so this send cannot block while q.mu is held.
		q.ready <- ordered
		q.nextSequence++
	}
	return nil
}

func (q *socksFrameQueue) complete(data *sliverpb.SocksData) {
	if data == nil {
		return
	}
	q.mu.Lock()
	size, ok := q.reservations[data.Sequence]
	if ok && q.accepted[data.Sequence] == data {
		delete(q.reservations, data.Sequence)
		delete(q.accepted, data.Sequence)
		q.reservedFrames--
		q.reservedBytes -= size
		q.signalSpaceChangeLocked()
	}
	q.mu.Unlock()
}

func (q *socksFrameQueue) clear() {
	q.mu.Lock()
	clear(q.pending)
	for {
		select {
		case <-q.ready:
		default:
			clear(q.accepted)
			clear(q.reservations)
			q.reservedFrames = 0
			q.reservedBytes = 0
			q.signalSpaceChangeLocked()
			q.mu.Unlock()
			return
		}
	}
}

func (q *socksFrameQueue) snapshot() (next uint64, frames int, size int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.nextSequence, q.reservedFrames, q.reservedBytes
}

func copySocksProtocolFrame(tunnelID uint64, data *sliverpb.SocksData) *sliverpb.SocksData {
	return &sliverpb.SocksData{
		TunnelID:  tunnelID,
		Sequence:  data.Sequence,
		Data:      append([]byte(nil), data.Data...),
		CloseConn: data.CloseConn,
	}
}

func equalSocksProtocolFrame(a *sliverpb.SocksData, b *sliverpb.SocksData) bool {
	return a != nil && b != nil &&
		a.Sequence == b.Sequence &&
		a.CloseConn == b.CloseConn &&
		bytes.Equal(a.Data, b.Data)
}

// SocksDataSender is the send half of a server-side SOCKS proxy stream.
type SocksDataSender interface {
	Send(*sliverpb.SocksData) error
}

// SocksClient serializes sends on one SocksProxy stream. gRPC permits one
// concurrent sender and one concurrent receiver, but not multiple senders.
type SocksClient struct {
	stream      SocksDataSender
	sendMu      sync.Mutex
	failureOnce sync.Once
	failureDone chan struct{}
	failureMu   sync.Mutex
	failureErr  error
}

// NewSocksClient creates a serialized sender for one SOCKS proxy stream.
func NewSocksClient(stream SocksDataSender) *SocksClient {
	return &SocksClient{stream: stream, failureDone: make(chan struct{})}
}

// Send serializes and forwards one SOCKS frame.
func (c *SocksClient) Send(data *sliverpb.SocksData) error {
	if err := c.Err(); err != nil {
		return err
	}
	c.sendMu.Lock()
	defer c.sendMu.Unlock()
	if err := c.Err(); err != nil {
		return err
	}
	err := c.stream.Send(data)
	if err != nil {
		c.Fail(err)
	}
	return err
}

// Fail records the first terminal stream error.
func (c *SocksClient) Fail(err error) {
	if c == nil || err == nil {
		return
	}
	c.failureOnce.Do(func() {
		c.failureMu.Lock()
		c.failureErr = err
		c.failureMu.Unlock()
		close(c.failureDone)
	})
}

// Done is closed after the first terminal stream error.
func (c *SocksClient) Done() <-chan struct{} {
	if c == nil {
		closed := make(chan struct{})
		close(closed)
		return closed
	}
	return c.failureDone
}

// Err returns the first terminal stream error, if any.
func (c *SocksClient) Err() error {
	if c == nil {
		return errors.New("nil SOCKS client")
	}
	select {
	case <-c.failureDone:
		c.failureMu.Lock()
		defer c.failureMu.Unlock()
		return c.failureErr
	default:
		return nil
	}
}

// TcpTunnel holds one server-side SOCKS tunnel generation.
//
//nolint:revive // Preserve the established exported API spelling.
type TcpTunnel struct {
	ID                  uint64
	ToImplantSequence   uint64
	FromImplantSequence uint64
	SessionID           string
	ToImplantMux        sync.Mutex
	FromImplantMux      sync.Mutex

	toImplant                *socksFrameQueue
	fromImplant              *socksFrameQueue
	done                     chan struct{}
	stateMu                  sync.RWMutex
	closed                   bool
	clientMu                 sync.RWMutex
	client                   *SocksClient
	capabilities             uint64
	username                 string
	password                 string
	boundAt                  time.Time
	lastActivity             time.Time
	receivedPayload          bool
	sendsTerminal            bool
	legacyTerminalMu         sync.Mutex
	legacyTerminalClaimed    bool
	legacyTerminalPending    bool
	legacyTerminalGeneration uint64
	legacyTerminalSignal     chan struct{}
	legacyTerminalChanged    chan struct{}
	// implantConnection is the immutable transport generation that owned this
	// tunnel when it was created. Session removal deletes the registry entry
	// before tunnel cleanup, so close paths must not rediscover this association
	// by session ID.
	implantConnection *ImplantConnection
	onSessionClose    func(*TcpTunnel)
	toImplantAcks     *socksAckMailbox
	toClientAcks      *socksAckMailbox
}

type tcpTunnel struct {
	tunnels map[uint64]*TcpTunnel
	mutex   *sync.RWMutex
}

func (t *tcpTunnel) Create(sessionID string, onSessionClose ...func(*TcpTunnel)) (*TcpTunnel, error) {
	session := Sessions.Get(sessionID)
	if session == nil {
		return nil, ErrInvalidSessionID
	}
	return t.CreateForSession(session, 0, onSessionClose...)
}

// CreateWithCapabilities creates a tunnel with the immutable capability set
// negotiated between its operator and exact implant session generation.
func (t *tcpTunnel) CreateWithCapabilities(sessionID string, capabilities uint64, onSessionClose ...func(*TcpTunnel)) (*TcpTunnel, error) {
	session := Sessions.Get(sessionID)
	if session == nil {
		return nil, ErrInvalidSessionID
	}
	return t.CreateForSession(session, capabilities, onSessionClose...)
}

// CreateForSession binds creation to an exact session pointer and transport
// generation. This prevents capabilities read from one same-ID session from
// being applied to a replacement found by a later registry lookup.
func (t *tcpTunnel) CreateForSession(session *Session, capabilities uint64, onSessionClose ...func(*TcpTunnel)) (*TcpTunnel, error) {
	if session == nil || session.ID == "" || session.Connection == nil || Sessions.Get(session.ID) != session {
		return nil, ErrInvalidSessionID
	}
	tunnelID := NewTunnelID()
	var sessionClose func(*TcpTunnel)
	if len(onSessionClose) > 0 {
		sessionClose = onSessionClose[0]
	}
	tunnel := newTCPTunnelWithCapabilities(tunnelID, session.ID, session.Connection, capabilities&sliverpb.CapabilitySocksFlowControlV1)
	tunnel.onSessionClose = sessionClose
	t.mutex.Lock()
	owned := 0
	for _, candidate := range t.tunnels {
		if candidate.SessionID == session.ID {
			owned++
		}
	}
	if owned >= maxSocksTunnelsPerSession {
		t.mutex.Unlock()
		tunnel.close()
		return nil, ErrSocksTunnelLimit
	}
	t.tunnels[tunnel.ID] = tunnel
	t.mutex.Unlock()

	// Session removal can race the gap between the initial lookup and registry
	// insertion. Revalidate the exact session generation after publication so a
	// SOCKS tunnel cannot appear after session cleanup has already completed.
	if Sessions.Get(session.ID) != session {
		t.CloseIf(tunnel)
		return nil, ErrInvalidSessionID
	}

	return tunnel, nil
}

func (t *tcpTunnel) Close(tunnelID uint64) error {
	t.mutex.Lock()
	tunnel := t.tunnels[tunnelID]
	if tunnel == nil {
		t.mutex.Unlock()
		return ErrInvalidTunnelID
	}
	delete(t.tunnels, tunnelID)
	t.mutex.Unlock()
	tunnel.close()
	return nil
}

// CloseIf removes and closes exactly this tunnel generation. It is safe for
// multiple lifecycle owners to race; only one caller wins.
func (t *tcpTunnel) CloseIf(tunnel *TcpTunnel) bool {
	if tunnel == nil {
		return false
	}
	t.mutex.Lock()
	if t.tunnels[tunnel.ID] != tunnel {
		t.mutex.Unlock()
		return false
	}
	delete(t.tunnels, tunnel.ID)
	t.mutex.Unlock()
	tunnel.close()
	return true
}

// CloseIfAndFinalize removes exactly this tunnel generation and runs the same
// finalizer used by session teardown. Protocol rejection paths use it when a
// synthetic test connection has no registered session cleanup; in production,
// connection cleanup normally wins the race first.
func (t *tcpTunnel) CloseIfAndFinalize(tunnel *TcpTunnel) bool {
	if !t.CloseIf(tunnel) {
		return false
	}
	if tunnel.onSessionClose != nil {
		tunnel.onSessionClose(tunnel)
	}
	return true
}

func (t *tcpTunnel) Get(tunnelID uint64) *TcpTunnel {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	return t.tunnels[tunnelID]
}

func (t *tcpTunnel) List() []*TcpTunnel {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	tunnels := make([]*TcpTunnel, 0, len(t.tunnels))
	for _, tunnel := range t.tunnels {
		tunnels = append(tunnels, tunnel)
	}
	return tunnels
}

// CloseForSession closes every SOCKS tunnel owned by one disconnected session.
// The operation is scoped and idempotent so connection failure, session removal,
// and proxy-stream teardown may race safely.
func (t *tcpTunnel) CloseForSession(sessionID string) int {
	if sessionID == "" {
		return 0
	}
	t.mutex.Lock()
	tunnels := []*TcpTunnel{}
	for tunnelID, tunnel := range t.tunnels {
		if tunnel.SessionID != sessionID {
			continue
		}
		delete(t.tunnels, tunnelID)
		tunnels = append(tunnels, tunnel)
	}
	t.mutex.Unlock()

	closed := 0
	var finalizers sync.WaitGroup
	for _, tunnel := range tunnels {
		if !tunnel.close() {
			continue
		}
		closed++
		if tunnel.onSessionClose != nil {
			finalizers.Add(1)
			go func() {
				defer finalizers.Done()
				tunnel.onSessionClose(tunnel)
			}()
		}
	}
	// Finalizers are independent and RPC bounds each stream notification. Run
	// them concurrently so one session with many stalled local SOCKS clients
	// consumes one bounded close interval rather than one interval per tunnel.
	finalizers.Wait()
	return closed
}

// Done is closed when this exact tunnel generation is retired.
func (t *TcpTunnel) Done() <-chan struct{} {
	return t.done
}

// ImplantConnection returns the exact transport generation associated with
// this tunnel at creation. The association is immutable even after the owning
// session is detached from the global registry.
func (t *TcpTunnel) ImplantConnection() *ImplantConnection {
	if t == nil {
		return nil
	}
	return t.implantConnection
}

//nolint:revive // Match the established TcpTunnel API spelling.
func newTcpTunnel(tunnelID uint64, sessionID string, implantConnection *ImplantConnection) *TcpTunnel {
	return newTCPTunnelWithCapabilities(tunnelID, sessionID, implantConnection, 0)
}

func newTCPTunnelWithCapabilities(tunnelID uint64, sessionID string, implantConnection *ImplantConnection, capabilities uint64) *TcpTunnel {
	return &TcpTunnel{
		ID:                    tunnelID,
		SessionID:             sessionID,
		implantConnection:     implantConnection,
		capabilities:          capabilities & sliverpb.CapabilitySocksFlowControlV1,
		toImplant:             newSocksFrameQueue(maxSocksPendingFrames, maxSocksPendingBytes),
		fromImplant:           newSocksFrameQueue(maxSocksPendingFrames, maxSocksPendingBytes),
		toImplantAcks:         newSocksAckMailbox(),
		toClientAcks:          newSocksAckMailbox(),
		done:                  make(chan struct{}),
		legacyTerminalSignal:  make(chan struct{}, 1),
		legacyTerminalChanged: make(chan struct{}),
	}
}

// AdmitToImplant validates and orders an operator frame without retaining its
// Request metadata or protobuf unknown fields.
func (t *TcpTunnel) AdmitToImplant(data *sliverpb.SocksData) error {
	return t.tryAdmitToImplant(data)
}

func (t *TcpTunnel) tryAdmitToImplant(data *sliverpb.SocksData) error {
	if t == nil || data == nil {
		return ErrTunnelClosed
	}
	t.stateMu.RLock()
	defer t.stateMu.RUnlock()
	if t.closed {
		return ErrTunnelClosed
	}
	err := t.toImplant.admit(t.ID, data)
	if err == nil {
		t.recordActivity(len(data.Data) > 0)
	}
	return err
}

// AdmitToImplantContext waits for bounded operator-to-implant capacity instead
// of treating ordinary transport backpressure as a terminal protocol failure.
// Validation, duplicate, and sequence-window errors remain fail-fast. The
// queue's frame and byte reservations remain unchanged while this call waits.
func (t *TcpTunnel) AdmitToImplantContext(ctx context.Context, data *sliverpb.SocksData) error {
	if t == nil || data == nil {
		return ErrTunnelClosed
	}
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		spaceChanged := t.toImplant.spaceChange()
		err := t.tryAdmitToImplant(data)
		if err == nil {
			return nil
		}
		if !errors.Is(err, ErrTunnelIngressLimit) && !errors.Is(err, ErrTunnelPendingBytes) {
			return err
		}
		// A single frame that cannot fit in an empty queue can never become
		// admissible after a completion. Preserve the fail-fast byte-limit result.
		if t.toImplant.maxFrames <= 0 || len(data.Data) > t.toImplant.maxBytes {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.done:
			return ErrTunnelClosed
		case <-spaceChanged:
		}
	}
}

// ToImplant returns ordered operator frames awaiting implant delivery.
func (t *TcpTunnel) ToImplant() <-chan *sliverpb.SocksData {
	return t.toImplant.ready
}

// CompleteToImplant releases the combined admission reservation after the
// worker has sent or discarded a frame.
func (t *TcpTunnel) CompleteToImplant(data *sliverpb.SocksData) {
	t.toImplant.complete(data)
}

func (t *TcpTunnel) claimLegacyImplantTerminal(data *sliverpb.SocksData) error {
	if err := validateSocksFrame(data); err != nil {
		return err
	}
	t.legacyTerminalMu.Lock()
	defer t.legacyTerminalMu.Unlock()
	if t.legacyTerminalClaimed {
		return nil
	}
	t.legacyTerminalClaimed = true
	t.legacyTerminalPending = true
	t.advanceLegacyTerminalGenerationLocked()
	select {
	case t.legacyTerminalSignal <- struct{}{}:
	default:
	}
	return nil
}

// advanceLegacyTerminalGenerationLocked wakes the exact actor generation that
// owns the current quiet-window timer. Closing and replacing the channel makes
// the notification a broadcast that cannot be missed between a state snapshot
// and the actor beginning its wait.
func (t *TcpTunnel) advanceLegacyTerminalGenerationLocked() {
	t.legacyTerminalGeneration++
	close(t.legacyTerminalChanged)
	t.legacyTerminalChanged = make(chan struct{})
}

func (t *TcpTunnel) admitLegacyImplantData(data *sliverpb.SocksData) error {
	// Serialize data admission with terminal materialization. This guarantees
	// that a payload accepted at the quiet-window boundary is either sequenced
	// before the terminal or advances the generation and earns a fresh window.
	t.legacyTerminalMu.Lock()
	defer t.legacyTerminalMu.Unlock()
	err := t.fromImplant.admit(t.ID, data)
	if err == nil && t.legacyTerminalPending {
		t.advanceLegacyTerminalGenerationLocked()
	}
	return err
}

// LegacyImplantTerminalPending is signaled once when a capability-zero
// implant emits its unsequenced terminal. The exact tunnel's scheduler owns the
// bounded reorder grace and terminal materialization.
func (t *TcpTunnel) LegacyImplantTerminalPending() <-chan struct{} {
	return t.legacyTerminalSignal
}

// LegacyImplantTerminalState snapshots the provisional terminal generation.
// Later admitted implant data advances the generation so an expired waiter
// cannot overtake it.
func (t *TcpTunnel) LegacyImplantTerminalState() (pending bool, generation uint64, changed <-chan struct{}) {
	if t == nil {
		return false, 0, nil
	}
	t.legacyTerminalMu.Lock()
	defer t.legacyTerminalMu.Unlock()
	return t.legacyTerminalPending, t.legacyTerminalGeneration, t.legacyTerminalChanged
}

// TryFlushLegacyImplantTerminal queues the provisional terminal only if no
// data has advanced its observed generation during the reorder grace. done is
// true when the actor has no further work or successfully queued the terminal.
func (t *TcpTunnel) TryFlushLegacyImplantTerminal(observedGeneration uint64) (done bool, err error) {
	if t == nil {
		return true, ErrTunnelClosed
	}
	t.stateMu.RLock()
	defer t.stateMu.RUnlock()
	if t.closed {
		return true, ErrTunnelClosed
	}
	t.legacyTerminalMu.Lock()
	defer t.legacyTerminalMu.Unlock()
	if !t.legacyTerminalPending {
		return true, nil
	}
	if t.legacyTerminalGeneration != observedGeneration {
		return false, nil
	}
	err = t.fromImplant.admitLegacyTerminal(t.ID, &sliverpb.SocksData{
		TunnelID:  t.ID,
		CloseConn: true,
	})
	if err != nil {
		return false, err
	}
	t.legacyTerminalPending = false
	return true, nil
}

// FromImplantSpaceChange returns the current capacity generation used by the
// provisional legacy-terminal actor when the data queue is full.
func (t *TcpTunnel) FromImplantSpaceChange() <-chan struct{} {
	return t.fromImplant.spaceChange()
}

// ProcessDataFromImplant validates and orders an implant frame. Delivery is
// non-blocking and does not rely on the global handler mutex.
func (t *TcpTunnel) ProcessDataFromImplant(data *sliverpb.SocksData) error {
	if t == nil || data == nil {
		return ErrTunnelClosed
	}
	t.stateMu.RLock()
	defer t.stateMu.RUnlock()
	if t.closed {
		return ErrTunnelClosed
	}

	flowControl := t.FlowControlEnabled()
	if data.CloseConn && !flowControl {
		// Legacy terminals are unsequenced and independently transported. Keep
		// the first one provisional so an overtaken data frame can be admitted
		// before the exact relay worker materializes terminal EOF.
		err := t.claimLegacyImplantTerminal(data)
		if err == nil {
			// Keep the longer idle fallback from preempting the reorder grace when
			// a close arrives after an otherwise idle legacy connection.
			t.recordActivity(false)
		}
		return err
	}
	var err error
	if flowControl {
		err = t.fromImplant.admit(t.ID, data)
	} else {
		err = t.admitLegacyImplantData(data)
	}
	if err == nil {
		t.recordActivity(false)
	}
	return err
}

// FromImplant returns ordered implant frames awaiting operator delivery.
func (t *TcpTunnel) FromImplant() <-chan *sliverpb.SocksData {
	return t.fromImplant.ready
}

// CompleteFromImplant releases the combined admission reservation after the
// operator-stream worker has sent or discarded a frame.
func (t *TcpTunnel) CompleteFromImplant(data *sliverpb.SocksData) {
	t.fromImplant.complete(data)
}

// DeliverFromImplant is retained as a compatibility wrapper. Callers that
// need to distinguish a stale replay from a protocol violation should use
// ProcessDataFromImplant directly.
func (t *TcpTunnel) DeliverFromImplant(data *sliverpb.SocksData) bool {
	return t.ProcessDataFromImplant(data) == nil
}

// BindClient binds the first stream that presents this tunnel. owned reports
// whether client owns the tunnel; newlyBound is true only for the first bind.
func (t *TcpTunnel) BindClient(client *SocksClient) (owned bool, newlyBound bool) {
	owned, newlyBound, _ = t.BindClientWithCredentials(client, "", "")
	return owned, newlyBound
}

// BindClientWithCredentials captures authentication exactly once from the
// ownership bind. RFC 1929 limits each username/password field to 255 octets;
// bounding them here prevents per-frame metadata from escaping Data budgets.
func (t *TcpTunnel) BindClientWithCredentials(client *SocksClient, username string, password string) (owned bool, newlyBound bool, err error) {
	return t.BindClientWithCapabilities(client, username, password, false)
}

// BindClientWithCapabilities binds a proxy stream and records whether its
// client emits explicit per-connection terminal frames. Legacy clients did not
// emit terminals and therefore retain a bounded inactivity lease; current
// clients may remain idle after their first protocol payload (for example RDP).
func (t *TcpTunnel) BindClientWithCapabilities(client *SocksClient, username string, password string, sendsTerminal bool) (owned bool, newlyBound bool, err error) {
	return t.BindClientWithNegotiatedCapabilities(client, username, password, sendsTerminal, 0)
}

// BindClientWithNegotiatedCapabilities requires the first ownership marker to
// echo the exact capability set returned by CreateSocks. Subsequent payloads do
// not repeat capability metadata; a repeated ownership marker must still match.
func (t *TcpTunnel) BindClientWithNegotiatedCapabilities(client *SocksClient, username string, password string, sendsTerminal bool, capabilities uint64) (owned bool, newlyBound bool, err error) {
	if client == nil {
		return false, false, nil
	}
	t.stateMu.RLock()
	defer t.stateMu.RUnlock()
	if t.closed {
		return false, false, ErrTunnelClosed
	}
	t.clientMu.Lock()
	defer t.clientMu.Unlock()
	if t.client == nil {
		if capabilities != t.capabilities || t.capabilities != 0 && !sendsTerminal {
			return false, false, ErrSocksCapabilityMismatch
		}
		if len(username) > maxSocksCredentialBytes || len(password) > maxSocksCredentialBytes {
			return false, false, ErrSocksCredentialSize
		}
		t.client = client
		t.username = username
		t.password = password
		t.boundAt = time.Now()
		t.lastActivity = t.boundAt
		t.sendsTerminal = sendsTerminal
		return true, true, nil
	}
	if t.client == client && sendsTerminal {
		if capabilities != t.capabilities {
			return true, false, ErrSocksCapabilityMismatch
		}
		t.sendsTerminal = true
	}
	return t.client == client, false, nil
}

// Capabilities returns the immutable per-tunnel negotiated capability set.
func (t *TcpTunnel) Capabilities() uint64 {
	if t == nil {
		return 0
	}
	t.clientMu.RLock()
	defer t.clientMu.RUnlock()
	return t.capabilities
}

// FlowControlEnabled reports whether both endpoints negotiated SOCKS flow
// control for this exact tunnel generation.
func (t *TcpTunnel) FlowControlEnabled() bool {
	return t.Capabilities()&sliverpb.CapabilitySocksFlowControlV1 != 0
}

// RelayClientAcknowledgement validates a cumulative ACK from the exact bound
// operator and coalesces it for delivery to the implant. FromImplantMux closes
// the small race between a successful stream Send and its high-water update.
func (t *TcpTunnel) RelayClientAcknowledgement(client *SocksClient, ack uint64) error {
	if t == nil {
		return ErrTunnelClosed
	}
	t.stateMu.RLock()
	defer t.stateMu.RUnlock()
	if t.closed {
		return ErrTunnelClosed
	}
	t.FromImplantMux.Lock()
	defer t.FromImplantMux.Unlock()
	t.clientMu.RLock()
	owned := t.client == client && client != nil
	flowControl := t.capabilities&sliverpb.CapabilitySocksFlowControlV1 != 0
	t.clientMu.RUnlock()
	if !owned {
		return ErrSocksOwner
	}
	if !flowControl {
		return ErrSocksFlowControl
	}
	highWater := atomic.LoadUint64(&t.FromImplantSequence)
	if ack == 0 || ack > highWater {
		return fmt.Errorf("%w: got %d, sent %d", ErrSocksAcknowledgement, ack, highWater)
	}
	t.toImplantAcks.offer(ack)
	return nil
}

// RelayImplantAcknowledgement validates a cumulative ACK from the exact implant
// connection and coalesces it for delivery to the bound operator.
func (t *TcpTunnel) RelayImplantAcknowledgement(connection *ImplantConnection, ack uint64) error {
	if t == nil {
		return ErrTunnelClosed
	}
	t.stateMu.RLock()
	defer t.stateMu.RUnlock()
	if t.closed {
		return ErrTunnelClosed
	}
	t.ToImplantMux.Lock()
	defer t.ToImplantMux.Unlock()
	t.clientMu.RLock()
	bound := t.client != nil
	flowControl := t.capabilities&sliverpb.CapabilitySocksFlowControlV1 != 0
	t.clientMu.RUnlock()
	if connection == nil || connection != t.implantConnection || !bound {
		return ErrSocksOwner
	}
	if !flowControl {
		return ErrSocksFlowControl
	}
	highWater := atomic.LoadUint64(&t.ToImplantSequence)
	if ack == 0 || ack > highWater {
		return fmt.Errorf("%w: got %d, sent %d", ErrSocksAcknowledgement, ack, highWater)
	}
	t.toClientAcks.offer(ack)
	return nil
}

// AcknowledgementsToImplant returns the fixed-size ACK control mailbox.
func (t *TcpTunnel) AcknowledgementsToImplant() <-chan uint64 {
	return t.toImplantAcks.channel()
}

// AcknowledgementsToClient returns the fixed-size ACK control mailbox.
func (t *TcpTunnel) AcknowledgementsToClient() <-chan uint64 {
	return t.toClientAcks.channel()
}

// SocksClientLifecycle is an atomic snapshot used by the RPC lifecycle
// monitor. Timestamps use the server's monotonic clock component.
type SocksClientLifecycle struct {
	BoundAt         time.Time
	LastActivity    time.Time
	ReceivedPayload bool
	SendsTerminal   bool
}

// ClientLifecycle returns an atomic snapshot of client lifecycle state.
func (t *TcpTunnel) ClientLifecycle() SocksClientLifecycle {
	if t == nil {
		return SocksClientLifecycle{}
	}
	t.clientMu.RLock()
	defer t.clientMu.RUnlock()
	return SocksClientLifecycle{
		BoundAt:         t.boundAt,
		LastActivity:    t.lastActivity,
		ReceivedPayload: t.receivedPayload,
		SendsTerminal:   t.sendsTerminal,
	}
}

func (t *TcpTunnel) recordActivity(payload bool) {
	t.clientMu.Lock()
	if t.client != nil {
		t.lastActivity = time.Now()
		if payload {
			t.receivedPayload = true
		}
	}
	t.clientMu.Unlock()
}

// Client returns the proxy stream currently bound to the tunnel.
func (t *TcpTunnel) Client() *SocksClient {
	t.clientMu.RLock()
	defer t.clientMu.RUnlock()
	return t.client
}

// Credentials returns the username and password captured at bind time.
func (t *TcpTunnel) Credentials() (string, string) {
	t.clientMu.RLock()
	defer t.clientMu.RUnlock()
	return t.username, t.password
}

func (t *TcpTunnel) close() bool {
	t.stateMu.Lock()
	if t.closed {
		t.stateMu.Unlock()
		return false
	}
	t.closed = true
	close(t.done)
	t.toImplant.clear()
	t.fromImplant.clear()
	t.ToImplantMux.Lock()
	t.FromImplantMux.Lock()
	t.clientMu.Lock()
	t.username = ""
	t.password = ""
	t.clientMu.Unlock()
	t.FromImplantMux.Unlock()
	t.ToImplantMux.Unlock()
	t.stateMu.Unlock()
	return true
}
