package rtunnels

import (
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

var (
	rtunnels = make(map[uint64]*RTunnel)
	mutex    sync.RWMutex
)

// ErrDuplicateTunnelID rejects reuse of an active reverse tunnel ID.
var ErrDuplicateTunnelID = errors.New("reverse tunnel ID is already registered")

const (
	maxReverseTunnelFrameBytes    = sliverpb.MaxTunnelFrameBytes
	maxReverseTunnelPendingFrames = 128
	maxReverseTunnelPendingBytes  = maxReverseTunnelFrameBytes * maxReverseTunnelPendingFrames

	maxReverseTunnelIngress               = 128
	maxPendingBytesPerAuthorization int64 = 16 * 1024 * 1024
	maxPendingBytesPerSession       int64 = 32 * 1024 * 1024
	maxPendingBytesGlobal           int64 = 128 * 1024 * 1024
	closeMatchingTunnelsTimeout           = 20 * time.Second
)

// ErrReverseTunnelFrameTooLarge and the related errors report rejected relay operations.
var (
	ErrReverseTunnelFrameTooLarge = errors.New("reverse tunnel frame exceeds the size limit")
	ErrReverseTunnelWindow        = errors.New("reverse tunnel sequence exceeds the pending window")
	ErrReverseTunnelPendingBytes  = errors.New("reverse tunnel pending data exceeds the byte limit")
	ErrReverseTunnelClosed        = errors.New("reverse tunnel is closed")
	ErrReverseTunnelIngressLimit  = errors.New("reverse tunnel inbound concurrency limit reached")
	ErrReverseTunnelAuthBudget    = errors.New("reverse tunnel authorization pending-data budget reached")
	ErrReverseTunnelSessionBudget = errors.New("reverse tunnel session pending-data budget reached")
	ErrReverseTunnelGlobalBudget  = errors.New("reverse tunnel global pending-data budget reached")
	ErrReverseTunnelTerminal      = errors.New("reverse tunnel terminal sequence is invalid")
)

type pendingRelayAuthorization struct {
	sessionID       string
	authorizationID AuthorizationID
}

// pendingRelayBudget bounds payloads admitted to relay handlers, including
// payloads waiting for the per-tunnel serialization lock. This prevents a
// blocked destination from multiplying the per-tunnel bound across every
// connection owned by one authorization or implant.
type pendingRelayBudget struct {
	mutex           sync.Mutex
	total           int64
	bySession       map[string]int64
	byAuthorization map[pendingRelayAuthorization]int64
}

func newPendingRelayBudget() *pendingRelayBudget {
	return &pendingRelayBudget{
		bySession:       map[string]int64{},
		byAuthorization: map[pendingRelayAuthorization]int64{},
	}
}

func (budget *pendingRelayBudget) reserve(sessionID string, authorizationID AuthorizationID, size int) error {
	if size <= 0 {
		return nil
	}
	amount := int64(size)
	key := pendingRelayAuthorization{sessionID: sessionID, authorizationID: authorizationID}

	budget.mutex.Lock()
	defer budget.mutex.Unlock()
	if budget.byAuthorization[key]+amount > maxPendingBytesPerAuthorization {
		return ErrReverseTunnelAuthBudget
	}
	if budget.bySession[sessionID]+amount > maxPendingBytesPerSession {
		return ErrReverseTunnelSessionBudget
	}
	if budget.total+amount > maxPendingBytesGlobal {
		return ErrReverseTunnelGlobalBudget
	}
	budget.byAuthorization[key] += amount
	budget.bySession[sessionID] += amount
	budget.total += amount
	return nil
}

func (budget *pendingRelayBudget) release(sessionID string, authorizationID AuthorizationID, size int) {
	if budget == nil || size <= 0 {
		return
	}
	amount := int64(size)
	key := pendingRelayAuthorization{sessionID: sessionID, authorizationID: authorizationID}

	budget.mutex.Lock()
	defer budget.mutex.Unlock()
	currentAuthorization := budget.byAuthorization[key]
	if currentAuthorization == 0 {
		return
	}
	if currentAuthorization < amount {
		amount = currentAuthorization
	}
	if currentAuthorization == amount {
		delete(budget.byAuthorization, key)
	} else {
		budget.byAuthorization[key] = currentAuthorization - amount
	}
	if current := budget.bySession[sessionID]; current <= amount {
		delete(budget.bySession, sessionID)
	} else {
		budget.bySession[sessionID] = current - amount
	}
	if budget.total <= amount {
		budget.total = 0
	} else {
		budget.total -= amount
	}
}

var defaultPendingRelayBudget = newPendingRelayBudget()

// RTunnel - Duplex byte read/write
type RTunnel struct {
	ID        uint64
	SessionID string
	// Reader       io.ReadCloser
	Readers      []io.ReadCloser
	readSequence uint64

	Writer          io.WriteCloser
	writeSequence   uint64
	outboundMutex   sync.Mutex
	outboundClosing bool

	mutex              *sync.RWMutex
	inboundMutex       sync.Mutex
	inboundAdmission   chan struct{}
	pendingInbound     map[uint64][]byte
	pendingBytes       int
	pendingBudget      *pendingRelayBudget
	closeOnce          sync.Once
	done               chan struct{}
	authorizationID    AuthorizationID
	peerCloseSet       atomic.Bool
	peerTeardown       atomic.Bool
	peerCloseSequence  uint64
	peerCloseTimerOnce sync.Once
	peerCloseNotifier  func(uint64) error
}

func NewRTunnel(id uint64, sID string, writer io.WriteCloser, readers ...io.ReadCloser) *RTunnel {
	return &RTunnel{
		ID:               id,
		SessionID:        sID,
		Readers:          readers,
		Writer:           writer,
		mutex:            &sync.RWMutex{},
		inboundAdmission: make(chan struct{}, maxReverseTunnelIngress),
		pendingInbound:   map[uint64][]byte{},
		pendingBudget:    defaultPendingRelayBudget,
		done:             make(chan struct{}),
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

// SetPeerCloseNotifier installs the bounded transport callback used when the
// server is the side that first closes this relay. It must be set before the
// tunnel is published.
func (c *RTunnel) SetPeerCloseNotifier(notifier func(uint64) error) {
	c.peerCloseNotifier = notifier
}

// QueueOutbound serializes server-to-implant frames with terminal close. The
// sequence advances only after the transport accepted the frame.
func (c *RTunnel) QueueOutbound(send func(uint64) error) error {
	if send == nil {
		return ErrReverseTunnelClosed
	}
	c.outboundMutex.Lock()
	defer c.outboundMutex.Unlock()
	if c.outboundClosing {
		return ErrReverseTunnelClosed
	}
	select {
	case <-c.done:
		return ErrReverseTunnelClosed
	default:
	}
	sequence := c.WriteSequence()
	if err := send(sequence); err != nil {
		return err
	}
	c.IncWriteSequence()
	return nil
}

// CloseLocal sends one terminal sequence before closing local resources. The
// exclusive sequence guarantees that a concurrently dispatched close cannot
// overtake already accepted data at the implant.
func (c *RTunnel) closeLocal() (bool, error) {
	c.outboundMutex.Lock()
	if c.outboundClosing {
		c.outboundMutex.Unlock()
		return false, nil
	}
	select {
	case <-c.done:
		c.outboundMutex.Unlock()
		return false, nil
	default:
	}
	c.outboundClosing = true
	sequence := c.WriteSequence()
	notifier := c.peerCloseNotifier
	var err error
	if notifier != nil && !c.PeerTeardownPending() {
		err = notifier(sequence)
	}
	c.outboundMutex.Unlock()
	c.Close()
	return true, err
}

// CloseRemote closes a relay without echoing another close to the peer.
func (c *RTunnel) closeRemote() bool {
	// Publish peer ownership before Close unblocks a local terminal notifier.
	// A simultaneous local close must not mistake the resulting send failure for
	// a broken C2 transport.
	c.peerTeardown.Store(true)
	c.Close()
	c.outboundMutex.Lock()
	initiated := !c.outboundClosing
	c.outboundClosing = true
	c.outboundMutex.Unlock()
	return initiated
}

// MarkPeerClose records the exclusive last sequence expected from the implant.
// Legacy zero-sequence closes after data retain their immediate-close behavior.
func (c *RTunnel) MarkPeerClose(sequence uint64) (bool, error) {
	c.inboundMutex.Lock()
	defer c.inboundMutex.Unlock()
	select {
	case <-c.done:
		return true, ErrReverseTunnelClosed
	default:
	}
	expected := c.ReadSequence()
	if sequence == 0 {
		if c.peerCloseSet.Load() && c.peerCloseSequence != expected {
			return false, ErrReverseTunnelTerminal
		}
		c.peerCloseSequence = expected
		c.peerTeardown.Store(true)
		c.peerCloseSet.Store(true)
		return true, nil
	}
	if sequence < expected || sequence-expected > maxReverseTunnelPendingFrames {
		return false, ErrReverseTunnelTerminal
	}
	if c.peerCloseSet.Load() && c.peerCloseSequence != sequence {
		return false, ErrReverseTunnelTerminal
	}
	for pendingSequence := range c.pendingInbound {
		if pendingSequence >= sequence {
			return false, ErrReverseTunnelTerminal
		}
	}
	c.peerCloseSequence = sequence
	c.peerTeardown.Store(true)
	c.peerCloseSet.Store(true)
	return expected >= sequence, nil
}

// PeerClosePending reports whether the peer supplied a terminal sequence.
func (c *RTunnel) PeerClosePending() bool {
	return c.peerCloseSet.Load()
}

// PeerTeardownPending reports peer-owned teardown without pretending that a
// sequenced terminal has been installed. Inbound ordering consults only
// PeerClosePending; notifier failure suppression uses this combined state.
func (c *RTunnel) PeerTeardownPending() bool {
	return c.peerTeardown.Load() || c.peerCloseSet.Load()
}

// PeerCloseReady reports whether all frames preceding the peer terminal arrived.
func (c *RTunnel) PeerCloseReady() bool {
	c.inboundMutex.Lock()
	defer c.inboundMutex.Unlock()
	return c.peerCloseSet.Load() && c.ReadSequence() >= c.peerCloseSequence
}

// StartPeerCloseDeadline runs expired if the terminal sequence remains incomplete.
func (c *RTunnel) StartPeerCloseDeadline(timeout time.Duration, expired func()) {
	if expired == nil {
		return
	}
	c.peerCloseTimerOnce.Do(func() {
		go func() {
			timer := time.NewTimer(timeout)
			defer timer.Stop()
			select {
			case <-c.Done():
			case <-timer.C:
				expired()
			}
		}()
	})
}

// ProcessInbound serializes one relay's inbound stream, bounds out-of-order
// buffering, and drains contiguous frames through write. Pending data is owned
// by this tunnel instance, so stale cleanup for a reused numeric ID cannot
// delete another generation's cache.
//
//nolint:gocyclo // Admission, ordering, budget, and terminal checks must remain one locked transaction.
func (c *RTunnel) ProcessInbound(sequence uint64, data []byte, write func([]byte) error) (int, error) {
	if write == nil {
		return 0, errors.New("reverse tunnel write function is nil")
	}
	if len(data) > maxReverseTunnelFrameBytes {
		return 0, fmt.Errorf("%w: got %d bytes, limit %d", ErrReverseTunnelFrameTooLarge, len(data), maxReverseTunnelFrameBytes)
	}
	select {
	case <-c.done:
		return 0, ErrReverseTunnelClosed
	default:
	}

	budget := c.pendingBudget
	if budget == nil {
		budget = defaultPendingRelayBudget
	}
	if err := budget.reserve(c.SessionID, c.authorizationID, len(data)); err != nil {
		return 0, err
	}
	reservationTransferred := false
	defer func() {
		if !reservationTransferred {
			budget.release(c.SessionID, c.authorizationID, len(data))
		}
	}()

	select {
	case c.inboundAdmission <- struct{}{}:
		defer func() { <-c.inboundAdmission }()
	default:
		return 0, ErrReverseTunnelIngressLimit
	}

	c.inboundMutex.Lock()
	defer c.inboundMutex.Unlock()
	select {
	case <-c.done:
		return len(c.pendingInbound), ErrReverseTunnelClosed
	default:
	}
	// Constructors initialize this map, but retaining the invariant here keeps a
	// zero-value or previously cleared tunnel from ever turning late data into a
	// process-wide nil-map panic.
	if c.pendingInbound == nil {
		c.pendingInbound = map[uint64][]byte{}
	}

	expected := c.ReadSequence()
	if c.peerCloseSet.Load() && sequence >= c.peerCloseSequence {
		return len(c.pendingInbound), ErrReverseTunnelTerminal
	}
	if sequence < expected {
		return len(c.pendingInbound), nil
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
		reservationTransferred = true
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
		budget.release(c.SessionID, c.authorizationID, len(payload))
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
		if c.inboundMutex.TryLock() {
			c.releasePendingLocked()
			c.inboundMutex.Unlock()
		} else {
			// A destination write can be slow or non-cooperative. Registry and
			// resource teardown must not wait for it; release its bounded pending
			// budget once the generation's inbound actor yields.
			go func() {
				c.inboundMutex.Lock()
				c.releasePendingLocked()
				c.inboundMutex.Unlock()
			}()
		}
	})
}

func (c *RTunnel) releasePendingLocked() {
	// Keep the map allocated so a retained closed pointer cannot turn late data
	// into a nil-map write even if a future caller bypasses the Done guard.
	clear(c.pendingInbound)
	budget := c.pendingBudget
	if budget == nil {
		budget = defaultPendingRelayBudget
	}
	budget.release(c.SessionID, c.authorizationID, c.pendingBytes)
	c.pendingBytes = 0
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

// CloseLocalIfActive selects local-close semantics only while this exact
// generation is still published. The terminal state is established before the
// registry entry disappears, so retained writers cannot enqueue after detach.
func CloseLocalIfActive(tunnel *RTunnel) (bool, error) {
	if tunnel == nil || GetRTunnel(tunnel.ID) != tunnel {
		return false, nil
	}
	initiated, err := tunnel.closeLocal()
	if !initiated || !RemoveRTunnelIf(tunnel.ID, tunnel) {
		return false, nil
	}
	return true, err
}

// CloseRemoteIfActive closes an exact peer-terminated generation without an
// echo and removes it only after its closed state is visible to retained work.
func CloseRemoteIfActive(tunnel *RTunnel) bool {
	if tunnel == nil || GetRTunnel(tunnel.ID) != tunnel {
		return false
	}
	initiated := tunnel.closeRemote()
	if !initiated {
		return false
	}
	return RemoveRTunnelIf(tunnel.ID, tunnel)
}

// CloseSession atomically detaches and closes only tunnels owned by sessionID.
// It is safe to call repeatedly during disconnect cleanup.
func CloseSession(sessionID string) int {
	return closeMatchingTunnels(false, func(tunnel *RTunnel) bool {
		return tunnel.SessionID == sessionID
	})
}

// CloseAuthorization atomically detaches and closes active relays created by a
// specific authorization. It never closes another session's tunnel.
func CloseAuthorization(sessionID string, authorizationID AuthorizationID) int {
	if authorizationID == "" {
		return 0
	}
	return closeMatchingTunnels(true, func(tunnel *RTunnel) bool {
		return tunnel.SessionID == sessionID && tunnel.AuthorizationID() == authorizationID
	})
}

func closeMatchingTunnels(notifyPeer bool, matches func(*RTunnel) bool) int {
	mutex.RLock()
	tunnels := make([]*RTunnel, 0)
	for _, tunnel := range rtunnels {
		if tunnel != nil && matches(tunnel) {
			tunnels = append(tunnels, tunnel)
		}
	}
	mutex.RUnlock()

	var closeWait sync.WaitGroup
	var closedCount atomic.Int64
	closeWait.Add(len(tunnels))
	for _, tunnel := range tunnels {
		go func(tunnel *RTunnel) {
			defer closeWait.Done()
			if notifyPeer {
				if closed, _ := CloseLocalIfActive(tunnel); closed {
					closedCount.Add(1)
				}
			} else if CloseRemoteIfActive(tunnel) {
				closedCount.Add(1)
			}
		}(tunnel)
	}
	closed := make(chan struct{})
	go func() {
		closeWait.Wait()
		close(closed)
	}()
	select {
	case <-closed:
	case <-time.After(closeMatchingTunnelsTimeout):
		// Each goroutine retains ownership of its exact generation. Return within
		// one batch deadline; bounded notifier callbacks or connection teardown
		// finish state-before-detach cleanup.
	}
	return int(closedCount.Load())
}
