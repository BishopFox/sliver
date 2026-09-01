package transports

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
	ErrTunnelClosed           = errors.New("tunnel is closed")
	ErrTunnelTerminalSequence = errors.New("tunnel terminal sequence is invalid")
	ErrTunnelFrameTooLarge    = errors.New("tunnel frame exceeds the size limit")
	ErrTunnelSequenceWindow   = errors.New("tunnel sequence exceeds the pending window")
	ErrTunnelPendingBytes     = errors.New("tunnel pending data exceeds the byte limit")
)

const (
	// MTLS and WireGuard accept up to 128 concurrent streams. The receive window
	// must admit that much legitimate handler reordering while remaining bounded.
	maxTunnelPendingFrames = 128
	maxTunnelFrameBytes    = sliverpb.MaxTunnelFrameBytes
	maxTunnelPendingBytes  = maxTunnelPendingFrames * maxTunnelFrameBytes
)

// Tunnel - Duplex byte read/write
type Tunnel struct {
	ID uint64

	// Reader       io.ReadCloser
	Readers      []io.ReadCloser
	readSequence uint64

	Writer        io.WriteCloser
	writeSequence uint64
	outboundMutex sync.Mutex
	outboundClose bool

	mutex          *sync.RWMutex
	inboundMutex   sync.Mutex
	pendingInbound map[uint64][]byte
	pendingBytes   int
	closeOnce      sync.Once
	done           chan struct{}

	peerCloseSet       atomic.Bool
	peerTeardown       atomic.Bool
	peerCloseSequence  uint64
	peerCloseTimerOnce sync.Once
	peerCloseNotifier  func(uint64) error
	reverse            bool
}

// NewReverseTunnel marks a generation as reverse-port-forward traffic. Reverse
// relays use reliable transports and bounded reordering, not the legacy generic
// tunnel resend/cache protocol.
func NewReverseTunnel(id uint64, writer io.WriteCloser, readers ...io.ReadCloser) *Tunnel {
	tunnel := NewTunnel(id, writer, readers...)
	tunnel.reverse = true
	return tunnel
}

func (c *Tunnel) IsReverse() bool {
	return c != nil && c.reverse
}

func NewTunnel(id uint64, writer io.WriteCloser, readers ...io.ReadCloser) *Tunnel {
	return &Tunnel{
		ID:             id,
		Readers:        readers,
		Writer:         writer,
		mutex:          &sync.RWMutex{},
		pendingInbound: map[uint64][]byte{},
		done:           make(chan struct{}),
	}
}

// Done is closed when this exact tunnel generation is detached.
func (c *Tunnel) Done() <-chan struct{} {
	return c.done
}

// SetPeerCloseNotifier installs the bounded transport callback used when this
// tunnel generation closes locally. It must be set before publication.
func (c *Tunnel) setPeerCloseNotifier(notifier func(uint64) error) {
	c.peerCloseNotifier = notifier
}

// QueueOutbound serializes data frames with the terminal close notification.
// A sequence is advanced only after the transport accepted the frame.
func (c *Tunnel) queueOutbound(send func(uint64) error) error {
	if send == nil {
		return ErrTunnelClosed
	}
	c.outboundMutex.Lock()
	defer c.outboundMutex.Unlock()
	if c.outboundClose {
		return ErrTunnelClosed
	}
	select {
	case <-c.done:
		return ErrTunnelClosed
	default:
	}
	sequence := c.WriteSequence()
	if err := send(sequence); err != nil {
		return err
	}
	c.incWriteSequence()
	return nil
}

// queueControl serializes a non-data control frame (currently resend) with
// data and terminal close, without consuming a data sequence number.
func (c *Tunnel) queueControl(send func(uint64, uint64) error) error {
	if send == nil {
		return ErrTunnelClosed
	}
	c.outboundMutex.Lock()
	defer c.outboundMutex.Unlock()
	if c.outboundClose {
		return ErrTunnelClosed
	}
	select {
	case <-c.done:
		return ErrTunnelClosed
	default:
	}
	return send(c.WriteSequence(), c.ReadSequence())
}

// CloseLocal emits one exclusive terminal sequence after every accepted data
// frame, then closes this generation. If the peer already initiated teardown,
// it closes without echoing another terminal frame.
func (c *Tunnel) closeLocal() (bool, error) {
	c.outboundMutex.Lock()
	if c.outboundClose {
		c.outboundMutex.Unlock()
		return false, nil
	}
	select {
	case <-c.done:
		c.outboundMutex.Unlock()
		return false, nil
	default:
	}
	c.outboundClose = true
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

// CloseRemote closes a generation without echoing a close to its peer.
func (c *Tunnel) closeRemote() {
	// Publish peer ownership before Close unblocks a local terminal notifier.
	// A simultaneous local close must treat ErrTunnelClosed from that notifier
	// as normal peer teardown, not as a transport delivery failure.
	c.peerTeardown.Store(true)
	c.Close()
	c.outboundMutex.Lock()
	c.outboundClose = true
	c.outboundMutex.Unlock()
}

// MarkPeerClose records an exclusive final inbound sequence. A zero sequence
// received after data preserves legacy close semantics and closes immediately.
func (c *Tunnel) MarkPeerClose(sequence uint64) (bool, error) {
	c.inboundMutex.Lock()
	defer c.inboundMutex.Unlock()
	select {
	case <-c.done:
		return true, ErrTunnelClosed
	default:
	}
	expected := c.ReadSequence()
	if sequence == 0 {
		if c.peerCloseSet.Load() && c.peerCloseSequence != expected {
			return false, ErrTunnelTerminalSequence
		}
		c.peerCloseSequence = expected
		c.peerTeardown.Store(true)
		c.peerCloseSet.Store(true)
		return true, nil
	}
	if sequence < expected || sequence-expected > maxTunnelPendingFrames {
		return false, ErrTunnelTerminalSequence
	}
	if c.peerCloseSet.Load() && c.peerCloseSequence != sequence {
		return false, ErrTunnelTerminalSequence
	}
	for pendingSequence := range c.pendingInbound {
		if pendingSequence >= sequence {
			return false, ErrTunnelTerminalSequence
		}
	}
	c.peerCloseSequence = sequence
	c.peerTeardown.Store(true)
	c.peerCloseSet.Store(true)
	return expected >= sequence, nil
}

func (c *Tunnel) PeerClosePending() bool {
	return c.peerCloseSet.Load()
}

// PeerTeardownPending reports that the peer owns teardown, whether it supplied
// a sequenced terminal or the remote-close actor is already closing resources.
// It is deliberately separate from PeerClosePending: only a validated terminal
// may constrain inbound sequence acceptance.
func (c *Tunnel) PeerTeardownPending() bool {
	return c.peerTeardown.Load() || c.peerCloseSet.Load()
}

func (c *Tunnel) PeerCloseReady() bool {
	c.inboundMutex.Lock()
	defer c.inboundMutex.Unlock()
	return c.peerCloseSet.Load() && c.ReadSequence() >= c.peerCloseSequence
}

// StartPeerCloseDeadline runs expired once if a terminal frame remains
// incomplete. Closing the tunnel cancels the deadline.
func (c *Tunnel) StartPeerCloseDeadline(timeout time.Duration, expired func()) {
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

func (c *Tunnel) ReadSequence() uint64 {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.readSequence
}

func (c *Tunnel) WriteSequence() uint64 {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.writeSequence
}

// ProcessInbound serializes one generation's complete accept/cache/drain
// transaction. Pending frames live on the generation rather than in an ID-only
// global cache, so stale handlers cannot corrupt a reused numeric tunnel ID.
func (c *Tunnel) ProcessInbound(sequence uint64, data []byte, write func([]byte) error) (int, error) {
	if write == nil {
		return 0, errors.New("tunnel write function is nil")
	}
	if len(data) > maxTunnelFrameBytes {
		return 0, fmt.Errorf("%w: got %d bytes, limit %d", ErrTunnelFrameTooLarge, len(data), maxTunnelFrameBytes)
	}

	c.inboundMutex.Lock()
	defer c.inboundMutex.Unlock()
	select {
	case <-c.done:
		return len(c.pendingInbound), ErrTunnelClosed
	default:
	}
	if c.pendingInbound == nil {
		c.pendingInbound = map[uint64][]byte{}
	}

	expected := c.ReadSequence()
	if c.peerCloseSet.Load() && sequence >= c.peerCloseSequence {
		return len(c.pendingInbound), ErrTunnelTerminalSequence
	}
	if sequence < expected {
		return len(c.pendingInbound), nil
	}
	if sequence-expected >= maxTunnelPendingFrames {
		return len(c.pendingInbound), fmt.Errorf("%w: got %d, expected %d", ErrTunnelSequenceWindow, sequence, expected)
	}
	if _, duplicate := c.pendingInbound[sequence]; !duplicate {
		if c.pendingBytes+len(data) > maxTunnelPendingBytes {
			return len(c.pendingInbound), ErrTunnelPendingBytes
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

func (c *Tunnel) incWriteSequence() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.writeSequence++
}

// IncReadSequence advances the expected inbound sequence number.
func (c *Tunnel) IncReadSequence() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.readSequence++
}

// Close - close tunnel reader and writer
func (c *Tunnel) Close() {
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
	})
}
