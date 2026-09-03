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
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util/leaky"
)

var (
	// SocksProxies - Struct instance that holds all the portfwds
	SocksProxies = socksProxy{
		tcpProxies: map[uint64]*SocksProxy{},
		mutex:      &sync.RWMutex{},
	}
	SocksConnPool = sync.Map{}
	SocksProxyID  = (uint64)(0)
)

// PortfwdMeta - Metadata about a portfwd listener
type SocksProxyMeta struct {
	ID        uint64
	SessionID string
	BindAddr  string
	Username  string
	Password  string
}

type socksConnectionState struct {
	conn     net.Conn
	receiver *socksReceiveQueue
	flow     *socksSendFlow
}

type TcpProxy struct {
	Rpc     rpcpb.SliverRPCClient
	Session *clientpb.Session

	Username        string
	Password        string
	BindAddr        string
	Listener        net.Listener
	KeepAlivePeriod time.Duration
	DialTimeout     time.Duration

	connectionsMu  sync.Mutex
	connections    map[uint64]socksConnectionState
	lifecycleCtx   context.Context
	lifecycleStop  context.CancelFunc
	started        bool
	stopped        bool
	stopOnce       sync.Once
	stopErr        error
	lifecycleErrMu sync.Mutex
	lifecycleErr   error
	receiveWG      sync.WaitGroup
	deliveryWG     sync.WaitGroup
	connectWG      sync.WaitGroup
	closeTimeout   time.Duration
	firstReadLease time.Duration
}

func (tcp *TcpProxy) Stop() error {
	tcp.beginStop()
	tcp.receiveWG.Wait()
	tcp.deliveryWG.Wait()
	tcp.connectWG.Wait()
	return errors.Join(tcp.stopErr, tcp.getLifecycleError())
}

func (tcp *TcpProxy) setLifecycleError(err error) {
	if err == nil {
		return
	}
	tcp.lifecycleErrMu.Lock()
	if tcp.lifecycleErr == nil {
		tcp.lifecycleErr = err
	}
	tcp.lifecycleErrMu.Unlock()
}

func (tcp *TcpProxy) getLifecycleError() error {
	tcp.lifecycleErrMu.Lock()
	defer tcp.lifecycleErrMu.Unlock()
	return tcp.lifecycleErr
}

func (tcp *TcpProxy) isStopping() bool {
	tcp.connectionsMu.Lock()
	defer tcp.connectionsMu.Unlock()
	return tcp.stopped
}

func (tcp *TcpProxy) beginStop() {
	tcp.stopOnce.Do(func() {
		tcp.connectionsMu.Lock()
		tcp.stopped = true
		lifecycleStop := tcp.lifecycleStop
		connections := tcp.connections
		tcp.connections = nil
		for tunnelID, state := range connections {
			SocksConnPool.CompareAndDelete(tunnelID, state.conn)
		}
		tcp.connectionsMu.Unlock()

		if lifecycleStop != nil {
			lifecycleStop()
		}
		if tcp.Listener != nil {
			tcp.stopErr = tcp.Listener.Close()
		}
		for _, state := range connections {
			state.receiver.stop()
		}
		for _, state := range connections {
			state.flow.close()
		}
		for _, state := range connections {
			_ = state.conn.Close()
		}
	})
}

func (tcp *TcpProxy) startContext() (context.Context, error) {
	tcp.connectionsMu.Lock()
	defer tcp.connectionsMu.Unlock()
	if tcp.stopped {
		return nil, net.ErrClosed
	}
	if tcp.started {
		return nil, errors.New("SOCKS proxy already started")
	}
	tcp.started = true
	tcp.lifecycleCtx, tcp.lifecycleStop = context.WithCancel(context.Background())
	return tcp.lifecycleCtx, nil
}

func (tcp *TcpProxy) addConnection(tunnelID uint64, connection net.Conn) bool {
	tcp.connectionsMu.Lock()
	if tcp.stopped {
		tcp.connectionsMu.Unlock()
		_ = connection.Close()
		return false
	}
	if tcp.connections == nil {
		tcp.connections = map[uint64]socksConnectionState{}
	}
	if _, exists := tcp.connections[tunnelID]; exists {
		tcp.connectionsMu.Unlock()
		_ = connection.Close()
		return false
	}
	state := socksConnectionState{
		conn:     connection,
		receiver: newSocksReceiveQueue(connection, socksReceiveFrameLimit, socksReceiveByteLimit),
		flow:     newSocksSendFlow(),
	}
	tcp.connections[tunnelID] = state
	tcp.deliveryWG.Add(1)
	// Retain the process-wide map for compatibility with ResetClientState. All
	// proxy routing and stop behavior is scoped through tcp.connections.
	SocksConnPool.Store(tunnelID, connection)
	tcp.connectionsMu.Unlock()

	go tcp.runReceiveQueue(tunnelID, state)
	return true
}

func (tcp *TcpProxy) getConnection(tunnelID uint64) (net.Conn, bool) {
	state, ok := tcp.getConnectionState(tunnelID, nil)
	if !ok {
		return nil, false
	}
	return state.conn, true
}

func (tcp *TcpProxy) getConnectionState(tunnelID uint64, expected net.Conn) (socksConnectionState, bool) {
	tcp.connectionsMu.Lock()
	defer tcp.connectionsMu.Unlock()
	state, ok := tcp.connections[tunnelID]
	if !ok || (expected != nil && state.conn != expected) {
		return socksConnectionState{}, false
	}
	return state, true
}

func (tcp *TcpProxy) removeConnection(tunnelID uint64, expected net.Conn) bool {
	tcp.connectionsMu.Lock()
	state, ok := tcp.connections[tunnelID]
	if !ok || (expected != nil && state.conn != expected) {
		tcp.connectionsMu.Unlock()
		return false
	}
	delete(tcp.connections, tunnelID)
	SocksConnPool.CompareAndDelete(tunnelID, state.conn)
	tcp.connectionsMu.Unlock()

	state.receiver.stop()
	state.flow.close()
	_ = state.conn.Close()
	state.receiver.wait()
	return true
}

func (tcp *TcpProxy) getReceiveQueue(tunnelID uint64) (net.Conn, *socksReceiveQueue, bool) {
	state, ok := tcp.getConnectionState(tunnelID, nil)
	if !ok {
		return nil, nil, false
	}
	return state.conn, state.receiver, true
}

func (tcp *TcpProxy) getSendFlow(tunnelID uint64) (*socksSendFlow, bool) {
	state, ok := tcp.getConnectionState(tunnelID, nil)
	if !ok {
		return nil, false
	}
	return state.flow, true
}

func (tcp *TcpProxy) runReceiveQueue(tunnelID uint64, state socksConnectionState) {
	defer tcp.deliveryWG.Done()
	defer state.receiver.finish()
	// An inbound terminal, local write failure, or acknowledgement-send failure
	// ends both halves of this exact tunnel. Wake an outbound sender that may be
	// blocked at the negotiated credit limit before closing its local socket.
	err := state.receiver.run()
	state.flow.close()
	if err != nil && !errors.Is(err, net.ErrClosed) {
		log.Printf("[socks] failed to deliver tunnel %d data: %s", tunnelID, err)
	}
	_ = state.conn.Close()
}

func (tcp *TcpProxy) startReceiveLoop(stream socksDataReceiver) bool {
	tcp.connectionsMu.Lock()
	defer tcp.connectionsMu.Unlock()
	if tcp.stopped {
		return false
	}
	tcp.receiveWG.Add(1)
	go func() {
		defer tcp.receiveWG.Done()
		receiveSocksData(tcp, stream)
	}()
	return true
}

func (tcp *TcpProxy) startConnectionWorker(connection net.Conn, stream *serializedSocksStream, frame *sliverpb.SocksData) bool {
	tcp.connectionsMu.Lock()
	defer tcp.connectionsMu.Unlock()
	if tcp.stopped {
		return false
	}
	tcp.connectWG.Add(1)
	go func() {
		defer tcp.connectWG.Done()
		connect(tcp, connection, stream, frame)
	}()
	return true
}

func (tcp *TcpProxy) tunnelCloseTimeout() time.Duration {
	if tcp.closeTimeout > 0 {
		return tcp.closeTimeout
	}
	return socksTunnelCloseTimeout
}

func (tcp *TcpProxy) firstPayloadTimeout() time.Duration {
	if tcp.firstReadLease > 0 {
		return tcp.firstReadLease
	}
	return socksFirstPayloadTimeout
}

const (
	// Keep one bounded SOCKS protocol window per local connection. HTTP C2 may
	// complete up to 64 implant POSTs concurrently, while the vendored SOCKS
	// relay emits 32 KiB frames. This window absorbs that ordinary transport
	// burst without coupling SOCKS to the much smaller generic tunnel startup
	// buffer. The stream receive loop still performs nonblocking admission, so a
	// connection whose peer stops reading cannot hold up unrelated tunnels.
	socksReceiveFrameLimit = 128
	socksReceiveByteLimit  = socksReceiveFrameLimit * sliverpb.MaxTunnelFrameBytes
	// Flow-control acknowledgements are deliberately less frequent than data
	// frames while still leaving several updates inside the fixed send window.
	socksFlowControlWindow      = sliverpb.SocksFlowControlWindowV1
	socksFlowAcknowledgementGap = sliverpb.SocksFlowControlAckBatchV1
)

var (
	errSocksReceiveQueueFull     = errors.New("SOCKS receive queue capacity exceeded")
	errSocksReceiveFrameTooLarge = errors.New("SOCKS receive frame exceeds payload limit")
	errSocksTerminalPayload      = errors.New("SOCKS terminal frame carries data")
)

type socksReceiveFrame struct {
	sequence uint64
	data     []byte
	terminal bool
}

// socksSendFlow applies per-tunnel TCP backpressure before the client reads
// another local frame. Cumulative implant acknowledgements wake only this
// connection, so one stalled SOCKS destination cannot stop sibling tunnels on
// the shared gRPC stream.
type socksSendFlow struct {
	mu      sync.Mutex
	enabled bool
	closed  bool
	sent    uint64
	acked   uint64
	changed chan struct{}
}

func newSocksSendFlow() *socksSendFlow {
	return &socksSendFlow{changed: make(chan struct{})}
}

func (flow *socksSendFlow) enable() {
	if flow == nil {
		return
	}
	flow.mu.Lock()
	flow.enabled = true
	flow.signalLocked()
	flow.mu.Unlock()
}

func (flow *socksSendFlow) wait() error {
	if flow == nil {
		return nil
	}
	for {
		flow.mu.Lock()
		if flow.closed {
			flow.mu.Unlock()
			return net.ErrClosed
		}
		if !flow.enabled || flow.sent-flow.acked < socksFlowControlWindow {
			flow.mu.Unlock()
			return nil
		}
		changed := flow.changed
		flow.mu.Unlock()
		<-changed
	}
}

func (flow *socksSendFlow) recordSent(sequence uint64) error {
	if flow == nil {
		return nil
	}
	flow.mu.Lock()
	defer flow.mu.Unlock()
	if flow.closed {
		return net.ErrClosed
	}
	if !flow.enabled {
		return nil
	}
	if sequence != flow.sent {
		return fmt.Errorf("SOCKS flow send sequence %d, want %d", sequence, flow.sent)
	}
	flow.sent++
	return nil
}

func (flow *socksSendFlow) acknowledge(ack uint64) error {
	if flow == nil || ack == 0 {
		return errors.New("invalid SOCKS flow acknowledgement")
	}
	flow.mu.Lock()
	defer flow.mu.Unlock()
	if flow.closed {
		return net.ErrClosed
	}
	if !flow.enabled {
		return errors.New("SOCKS flow acknowledgement was not negotiated")
	}
	if ack > flow.sent {
		return fmt.Errorf("SOCKS flow acknowledgement %d exceeds sent sequence %d", ack, flow.sent)
	}
	if ack <= flow.acked {
		return nil
	}
	flow.acked = ack
	flow.signalLocked()
	return nil
}

func (flow *socksSendFlow) close() {
	if flow == nil {
		return
	}
	flow.mu.Lock()
	if !flow.closed {
		flow.closed = true
		flow.signalLocked()
	}
	flow.mu.Unlock()
}

func (flow *socksSendFlow) signalLocked() {
	close(flow.changed)
	flow.changed = make(chan struct{})
}

// socksReceiveQueue is a fixed-size, single-writer delivery actor. Its frame
// and byte reservations include the item currently blocked in net.Conn.Write,
// not just items waiting in the channel.
type socksReceiveQueue struct {
	connection net.Conn
	queue      chan socksReceiveFrame
	done       chan struct{}
	finished   chan struct{}
	stopOnce   sync.Once

	mu            sync.Mutex
	stopped       bool
	terminal      bool
	pendingFrames int
	pendingBytes  int
	maxFrames     int
	maxBytes      int
	flowEnabled   bool
	nextSequence  uint64
	consumed      uint64
	lastAck       uint64
	acknowledge   func(uint64) error
}

func newSocksReceiveQueue(connection net.Conn, maxFrames int, maxBytes int) *socksReceiveQueue {
	return &socksReceiveQueue{
		connection: connection,
		queue:      make(chan socksReceiveFrame, maxFrames),
		done:       make(chan struct{}),
		finished:   make(chan struct{}),
		maxFrames:  maxFrames,
		maxBytes:   maxBytes,
	}
}

func (receiver *socksReceiveQueue) enableFlowControl(acknowledge func(uint64) error) error {
	if receiver == nil || acknowledge == nil {
		return errors.New("invalid SOCKS receive flow control")
	}
	receiver.mu.Lock()
	defer receiver.mu.Unlock()
	if receiver.stopped || receiver.pendingFrames != 0 || receiver.nextSequence != 0 {
		return net.ErrClosed
	}
	receiver.flowEnabled = true
	receiver.acknowledge = acknowledge
	return nil
}

func validateSocksReceiveData(data *sliverpb.SocksData) error {
	if data == nil {
		return net.ErrClosed
	}
	if len(data.Data) > sliverpb.MaxTunnelFrameBytes {
		return errSocksReceiveFrameTooLarge
	}
	if data.CloseConn && len(data.Data) != 0 {
		return errSocksTerminalPayload
	}
	return nil
}

func (receiver *socksReceiveQueue) validateAdmissionLocked(data *sliverpb.SocksData) error {
	if receiver.stopped || receiver.terminal {
		return net.ErrClosed
	}
	if receiver.pendingFrames >= receiver.maxFrames || receiver.pendingBytes+len(data.Data) > receiver.maxBytes {
		return errSocksReceiveQueueFull
	}
	if receiver.flowEnabled && data.Sequence != receiver.nextSequence {
		return fmt.Errorf("SOCKS receive sequence %d, want %d", data.Sequence, receiver.nextSequence)
	}
	return nil
}

func (receiver *socksReceiveQueue) enqueueLocked(frame socksReceiveFrame) error {
	receiver.pendingFrames++
	receiver.pendingBytes += len(frame.data)
	if receiver.flowEnabled && !frame.terminal {
		receiver.nextSequence++
	}
	if frame.terminal {
		receiver.terminal = true
	}
	select {
	case receiver.queue <- frame:
		return nil
	default:
		receiver.pendingFrames--
		receiver.pendingBytes -= len(frame.data)
		if receiver.flowEnabled && !frame.terminal {
			receiver.nextSequence--
		}
		if frame.terminal {
			receiver.terminal = false
		}
		return errSocksReceiveQueueFull
	}
}

func (receiver *socksReceiveQueue) admit(data *sliverpb.SocksData) error {
	if err := validateSocksReceiveData(data); err != nil {
		return err
	}

	receiver.mu.Lock()
	defer receiver.mu.Unlock()
	if err := receiver.validateAdmissionLocked(data); err != nil {
		return err
	}
	return receiver.enqueueLocked(socksReceiveFrame{
		sequence: data.Sequence,
		data:     append([]byte(nil), data.Data...),
		terminal: data.CloseConn,
	})
}

func (receiver *socksReceiveQueue) complete(frame socksReceiveFrame) {
	receiver.mu.Lock()
	receiver.pendingFrames--
	receiver.pendingBytes -= len(frame.data)
	receiver.mu.Unlock()
}

func (receiver *socksReceiveQueue) acknowledgeConsumed(next uint64, force bool) error {
	receiver.mu.Lock()
	if !receiver.flowEnabled {
		receiver.mu.Unlock()
		return nil
	}
	if next > receiver.consumed {
		receiver.consumed = next
	}
	if receiver.consumed == 0 || receiver.consumed <= receiver.lastAck {
		receiver.mu.Unlock()
		return nil
	}
	if !force && receiver.consumed-receiver.lastAck < socksFlowAcknowledgementGap {
		receiver.mu.Unlock()
		return nil
	}
	ack := receiver.consumed
	acknowledge := receiver.acknowledge
	receiver.lastAck = ack
	receiver.mu.Unlock()
	if acknowledge == nil {
		return errors.New("missing SOCKS receive acknowledgement sender")
	}
	return acknowledge(ack)
}

func (receiver *socksReceiveQueue) flushAcknowledgement() error {
	receiver.mu.Lock()
	next := receiver.consumed
	last := receiver.lastAck
	receiver.mu.Unlock()
	if next == 0 || next == last {
		return nil
	}
	return receiver.acknowledgeConsumed(next, true)
}

func (receiver *socksReceiveQueue) stop() {
	receiver.stopOnce.Do(func() {
		receiver.mu.Lock()
		receiver.stopped = true
		close(receiver.done)
		receiver.mu.Unlock()
	})
}

func (receiver *socksReceiveQueue) finish() {
	// The one goroutine that calls run owns this completion signal.
	close(receiver.finished)
}

func (receiver *socksReceiveQueue) wait() {
	<-receiver.finished
}

func (receiver *socksReceiveQueue) run() error {
	for {
		select {
		case <-receiver.done:
			return net.ErrClosed
		default:
		}

		select {
		case <-receiver.done:
			return net.ErrClosed
		case frame := <-receiver.queue:
			if frame.terminal {
				ackErr := receiver.flushAcknowledgement()
				receiver.complete(frame)
				return ackErr
			}
			err := writeSocksReceiveFrame(receiver.connection, frame.data)
			receiver.complete(frame)
			if err != nil {
				return err
			}
			if err := receiver.acknowledgeConsumed(frame.sequence+1, false); err != nil {
				return err
			}
		}
	}
}

func writeSocksReceiveFrame(connection net.Conn, data []byte) error {
	for len(data) > 0 {
		written, err := connection.Write(data)
		if written < 0 || written > len(data) {
			return io.ErrShortWrite
		}
		if written > 0 {
			data = data[written:]
		}
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrNoProgress
		}
	}
	return nil
}

// SocksProxy - Tracks portfwd<->tcpproxy
type SocksProxy struct {
	ID           uint64
	ChannelProxy *TcpProxy
}

// GetMetadata - Get metadata about the portfwd
func (p *SocksProxy) GetMetadata() *SocksProxyMeta {
	return &SocksProxyMeta{
		ID:        p.ID,
		SessionID: p.ChannelProxy.Session.ID,
		BindAddr:  p.ChannelProxy.BindAddr,
		Username:  p.ChannelProxy.Username,
		Password:  p.ChannelProxy.Password,
	}
}

type socksProxy struct {
	tcpProxies map[uint64]*SocksProxy
	mutex      *sync.RWMutex
}

// Add - Add a TCP proxy instance
func (f *socksProxy) Add(tcpProxy *TcpProxy) *SocksProxy {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	Sockser := &SocksProxy{
		ID:           nextSocksProxyID(),
		ChannelProxy: tcpProxy,
	}
	f.tcpProxies[Sockser.ID] = Sockser

	return Sockser
}

func (f *socksProxy) Start(tcpProxy *TcpProxy) error {
	// A proxy may be removed immediately after it is registered but before this
	// blocking Start call is scheduled. Always remove this exact instance on
	// exit, including startup failures, so it cannot remain in the inventory.
	defer f.removeChannelProxy(tcpProxy)
	ctx, err := tcpProxy.startContext()
	if err != nil {
		return err
	}
	var stream *serializedSocksStream
	defer func() {
		// Cancel the stream and any in-flight CreateSocks RPC before waiting on
		// CloseSend. This also closes the listener and accepted connections.
		_ = tcpProxy.Stop()
		if stream != nil {
			_ = stream.CloseSend()
		}
	}()
	proxy, err := tcpProxy.Rpc.SocksProxy(ctx)
	if err != nil {
		return err
	}
	stream = &serializedSocksStream{stream: proxy}
	if !tcpProxy.startReceiveLoop(proxy) {
		return net.ErrClosed
	}
	for {
		connection, err := tcpProxy.Listener.Accept()
		if err != nil {
			log.Printf("Failed to accept new listener, probably already closed: %s\n", err)
			if ctx.Err() == nil && !tcpProxy.isStopping() {
				acceptErr := fmt.Errorf("accept SOCKS proxy connection: %w", err)
				tcpProxy.setLifecycleError(acceptErr)
				return acceptErr
			}
			return tcpProxy.getLifecycleError()
		}
		rpcSocks, err := tcpProxy.Rpc.CreateSocks(ctx, &sliverpb.Socks{
			SessionID:    tcpProxy.Session.ID,
			Capabilities: sliverpb.CapabilitySocksFlowControlV1,
		})
		if err != nil {
			log.Printf("Failed rpc call to create SOCKS tunnel for accepted connection: %s\n", err)
			_ = connection.Close()
			if ctx.Err() != nil || tcpProxy.isStopping() {
				return tcpProxy.getLifecycleError()
			}
			continue
		}
		if rpcSocks == nil {
			log.Printf("Failed rpc call to create SOCKS tunnel for accepted connection: empty response\n")
			_ = connection.Close()
			if ctx.Err() != nil || tcpProxy.isStopping() {
				return tcpProxy.getLifecycleError()
			}
			continue
		}

		frame := &sliverpb.SocksData{
			Username:     tcpProxy.Username,
			Password:     tcpProxy.Password,
			TunnelID:     rpcSocks.TunnelID,
			Capabilities: rpcSocks.Capabilities & sliverpb.CapabilitySocksFlowControlV1,
			Request:      &commonpb.Request{SessionID: rpcSocks.SessionID},
		}
		if !tcpProxy.startConnectionWorker(connection, stream, frame) {
			_ = connection.Close()
			// Stop can win after CreateSocks has published a server tunnel but
			// before this connection worker can bind it to the proxy stream. An
			// unbound tunnel is invisible to stream teardown, so retire it through
			// the same bounded unary fallback used by a failed worker terminal.
			closeSocksTunnel(tcpProxy.Rpc, rpcSocks.TunnelID, rpcSocks.SessionID, tcpProxy.tunnelCloseTimeout())
			return tcpProxy.getLifecycleError()
		}
	}
}

type socksDataReceiver interface {
	Recv() (*sliverpb.SocksData, error)
}

func receiveSocksData(tcpProxy *TcpProxy, stream socksDataReceiver) {
	for {
		socksData, err := stream.Recv()
		if err != nil {
			log.Printf("Failed to Recv from proxy, %s\n", err)
			if !tcpProxy.isStopping() {
				tcpProxy.setLifecycleError(fmt.Errorf("receive SOCKS proxy stream: %w", err))
			}
			// This loop is part of Stop's receive wait group, so it initiates
			// teardown without recursively waiting for itself.
			tcpProxy.beginStop()
			return
		}
		if socksData == nil {
			log.Printf("Failed to Recv from proxy: empty SOCKS frame")
			continue
		}
		tcpProxy.receiveSocksFrame(socksData)
	}
}

func isCanonicalSocksAcknowledgement(data *sliverpb.SocksData) bool {
	return len(data.Data) == 0 && !data.CloseConn && data.Sequence == 0 &&
		data.Capabilities == 0 && data.Username == "" && data.Password == "" &&
		data.Request == nil
}

func (tcp *TcpProxy) receiveSocksAcknowledgement(data *sliverpb.SocksData, state socksConnectionState) {
	if !isCanonicalSocksAcknowledgement(data) {
		log.Printf("[socks] closing tunnel %d after malformed acknowledgement", data.TunnelID)
		tcp.removeConnection(data.TunnelID, state.conn)
		return
	}
	if err := state.flow.acknowledge(data.Ack); err != nil {
		log.Printf("[socks] closing tunnel %d after invalid acknowledgement: %s", data.TunnelID, err)
		tcp.removeConnection(data.TunnelID, state.conn)
	}
}

func (tcp *TcpProxy) receiveSocksFrame(data *sliverpb.SocksData) {
	state, ok := tcp.getConnectionState(data.TunnelID, nil)
	if !ok {
		return
	}
	if data.Ack != 0 {
		tcp.receiveSocksAcknowledgement(data, state)
		return
	}
	if data.Capabilities != 0 {
		log.Printf("[socks] closing tunnel %d after unexpected capability metadata", data.TunnelID)
		tcp.removeConnection(data.TunnelID, state.conn)
		return
	}
	log.Printf(
		"[socks] agent to client tunnel %d sequence %d, data size %d, closed %t",
		data.TunnelID,
		data.Sequence,
		len(data.Data),
		data.CloseConn,
	)
	if err := state.receiver.admit(data); err != nil {
		log.Printf("[socks] closing tunnel %d after receive admission failed: %s", data.TunnelID, err)
		tcp.removeConnection(data.TunnelID, state.conn)
	}
}

func (f *socksProxy) removeChannelProxy(tcpProxy *TcpProxy) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	for id, registered := range f.tcpProxies {
		if registered.ChannelProxy == tcpProxy {
			delete(f.tcpProxies, id)
		}
	}
}

// Remove - Remove a TCP proxy instance
func (f *socksProxy) Remove(socksID uint64) bool {
	f.mutex.Lock()
	registered, ok := f.tcpProxies[socksID]
	if ok {
		delete(f.tcpProxies, socksID)
	}
	f.mutex.Unlock()
	if !ok {
		return false
	}

	// Stop may wait for connection workers and bounded close RPCs. Do not hold
	// the process-wide inventory lock while those independent lifecycles drain.
	_ = registered.ChannelProxy.Stop()
	return true
}

// List - List all TCP proxy instances
func (f *socksProxy) List() []*SocksProxyMeta {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	socksProxy := []*SocksProxyMeta{}
	for _, socks := range f.tcpProxies {
		socksProxy = append(socksProxy, socks.GetMetadata())
	}
	return socksProxy
}

func nextSocksProxyID() uint64 {
	return atomic.AddUint64(&SocksProxyID, 1)
}

const (
	// Read a full protocol frame at a time. The former 4,108-byte
	// Shadowsocks-era buffer could turn one ordinary multi-megabyte TCP write
	// into thousands of SOCKS messages and exhaust the bounded server/implant
	// admission windows even while the target was actively draining the
	// connection.
	leakyBufSize = sliverpb.MaxTunnelFrameBytes
	// Retain at most one SOCKS admission window of idle buffers. Active
	// connections may allocate independently, but the pool cannot pin the old
	// 2,048-buffer count (128 MiB at the protocol frame size).
	leakyBufPoolSize = 128
)

var leakyBuf = leaky.NewLeakyBuf(leakyBufPoolSize, leakyBufSize)

const (
	socksTunnelCloseTimeout = 5 * time.Second
	// socksLifecycleBindSequence is outside the reachable sequence space of a
	// practical connection. Current servers consume this empty frame as an
	// ownership/capability marker. Older servers merely retain it as an
	// out-of-order frame while continuing to process real data from sequence 0.
	// This makes the marker backward compatible without overloading Request
	// metadata or changing the protobuf schema.
	socksLifecycleBindSequence = ^uint64(0)
	// CreateSocks necessarily precedes the first SOCKS protocol read. Bound that
	// gap locally so a peer that connects without sending a greeting cannot keep
	// either its local socket or the server's unbound tunnel alive indefinitely.
	// The deadline is cleared after the first payload, so established RDP and
	// other legitimately idle connections are unaffected.
	socksFirstPayloadTimeout = 10 * time.Second
)

type socksDataStream interface {
	Send(*sliverpb.SocksData) error
	CloseSend() error
}

type serializedSocksStream struct {
	stream socksDataStream
	sendMu sync.Mutex
}

func (stream *serializedSocksStream) Send(frame *sliverpb.SocksData) error {
	stream.sendMu.Lock()
	defer stream.sendMu.Unlock()
	return stream.stream.Send(frame)
}

func (stream *serializedSocksStream) CloseSend() error {
	stream.sendMu.Lock()
	defer stream.sendMu.Unlock()
	return stream.stream.CloseSend()
}

func closeSocksTunnel(rpc rpcpb.SliverRPCClient, tunnelID uint64, sessionID string, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if _, err := rpc.CloseSocks(ctx, &sliverpb.Socks{TunnelID: tunnelID, SessionID: sessionID}); err != nil {
		log.Printf("[socks] failed to close server tunnel %d: %s", tunnelID, err)
	}
}

func socksFrameSessionID(frame *sliverpb.SocksData) string {
	if frame.Request != nil {
		return frame.Request.SessionID
	}
	return ""
}

func closeSocksConnection(tcpProxy *TcpProxy, tunnelID uint64, connection net.Conn) {
	if !tcpProxy.removeConnection(tunnelID, connection) {
		_ = connection.Close()
	}
	log.Printf("[socks] connection closed")
}

func enableSocksFlowControl(
	flow *socksSendFlow,
	receiver *socksReceiveQueue,
	stream *serializedSocksStream,
	frame *sliverpb.SocksData,
	sessionID string,
) error {
	if frame.Capabilities&sliverpb.CapabilitySocksFlowControlV1 == 0 {
		return nil
	}
	flow.enable()
	return receiver.enableFlowControl(func(ack uint64) error {
		return stream.Send(&sliverpb.SocksData{
			TunnelID: frame.TunnelID,
			Ack:      ack,
			Request:  &commonpb.Request{SessionID: sessionID},
		})
	})
}

func bindSocksConnection(stream *serializedSocksStream, frame *sliverpb.SocksData, sessionID string) error {
	// Bind immediately so a current server can associate this local connection
	// with the stream and apply its first-payload lease. The reserved sequence is
	// intentionally not zero: old servers retain the marker out of order and
	// still forward the first real SOCKS payload at sequence zero.
	return stream.Send(&sliverpb.SocksData{
		Username:     frame.Username,
		Password:     frame.Password,
		Sequence:     socksLifecycleBindSequence,
		Capabilities: frame.Capabilities,
		TunnelID:     frame.TunnelID,
		Request:      &commonpb.Request{SessionID: sessionID},
	})
}

func newSocksPayloadFrame(frame *sliverpb.SocksData, sessionID string, sequence uint64, data []byte) *sliverpb.SocksData {
	payload := &sliverpb.SocksData{
		// SendMsg may retain messages for tracing and stats after it
		// returns. Give every send immutable message and payload
		// ownership instead of exposing the reusable read buffer.
		Data:     append([]byte(nil), data...),
		Sequence: sequence,
		TunnelID: frame.TunnelID,
		Request:  &commonpb.Request{SessionID: sessionID},
	}
	if sequence == 0 {
		// Current servers retain credentials from sequence zero. Include
		// them on the first data frame for compatibility with older servers.
		payload.Username = frame.Username
		payload.Password = frame.Password
	}
	return payload
}

func relaySocksConnection(
	conn net.Conn,
	stream *serializedSocksStream,
	frame *sliverpb.SocksData,
	flow *socksSendFlow,
	sessionID string,
) uint64 {
	firstPayload := true
	var toImplantSequence uint64
	buff := leakyBuf.Get()
	defer leakyBuf.Put(buff)
	for {
		if err := flow.wait(); err != nil {
			return toImplantSequence
		}
		n, readErr := conn.Read(buff)
		var deadlineErr error
		if n > 0 {
			if firstPayload {
				// Clear only the startup lease. Established SOCKS connections may be
				// idle indefinitely (notably interactive RDP sessions). A Reader may
				// return final bytes together with EOF, and a concurrently closing
				// socket can reject this deadline change; always forward n before
				// applying either error so the terminal cannot overtake final data.
				deadlineErr = conn.SetReadDeadline(time.Time{})
				firstPayload = false
			}
			dataFrame := newSocksPayloadFrame(frame, sessionID, toImplantSequence, buff[:n])
			log.Printf("[socks] (User to Client) to Server to agent  Data Sequence %d , Data Size %d \n", toImplantSequence, len(dataFrame.Data))
			if err := flow.recordSent(toImplantSequence); err != nil {
				log.Printf("[socks] failed to reserve flow-control sequence %d: %s", toImplantSequence, err)
				return toImplantSequence
			}
			if err := stream.Send(dataFrame); err != nil {
				log.Printf("[socks] (User to Client) failed to send data, %s ", err)
				return toImplantSequence
			}
			toImplantSequence++
		}
		if deadlineErr != nil {
			log.Printf("[socks] failed to clear first-payload lease for tunnel %d: %s", frame.TunnelID, deadlineErr)
			return toImplantSequence
		}
		if readErr != nil {
			log.Printf("[socks] (User to Client) failed to read data, %s ", readErr)
			// A Reader may legally return final bytes with EOF. Process n before
			// applying the terminal error so those bytes precede CloseConn.
			return toImplantSequence
		}
	}
}

func sendSocksTerminal(stream *serializedSocksStream, frame *sliverpb.SocksData, sessionID string, sequence uint64) bool {
	terminal := &sliverpb.SocksData{
		CloseConn: true,
		Sequence:  sequence,
		TunnelID:  frame.TunnelID,
		Request:   &commonpb.Request{SessionID: sessionID},
	}
	if err := stream.Send(terminal); err != nil {
		log.Printf("[socks] failed to send terminal frame for tunnel %d: %s", frame.TunnelID, err)
		return false
	}
	return true
}

func connect(tcpProxy *TcpProxy, conn net.Conn, stream *serializedSocksStream, frame *sliverpb.SocksData) {
	sessionID := socksFrameSessionID(frame)
	terminalSent := false
	defer func() {
		if !terminalSent {
			closeSocksTunnel(tcpProxy.Rpc, frame.TunnelID, sessionID, tcpProxy.tunnelCloseTimeout())
		}
	}()
	if !tcpProxy.addConnection(frame.TunnelID, conn) {
		return
	}
	// Close and remove this proxy's connection once the tunnel is done.
	defer closeSocksConnection(tcpProxy, frame.TunnelID, conn)
	state, ok := tcpProxy.getConnectionState(frame.TunnelID, conn)
	if !ok {
		return
	}
	if err := enableSocksFlowControl(state.flow, state.receiver, stream, frame, sessionID); err != nil {
		log.Printf("[socks] failed to enable flow control for tunnel %d: %s", frame.TunnelID, err)
		return
	}

	if err := bindSocksConnection(stream, frame, sessionID); err != nil {
		log.Printf("[socks] failed to bind tunnel %d to proxy stream: %s", frame.TunnelID, err)
		return
	}
	if err := conn.SetReadDeadline(time.Now().Add(tcpProxy.firstPayloadTimeout())); err != nil {
		log.Printf("[socks] failed to apply first-payload lease for tunnel %d: %s", frame.TunnelID, err)
		return
	}

	log.Printf("tcp conn %q<--><-->%q \n", conn.LocalAddr(), conn.RemoteAddr())
	toImplantSequence := relaySocksConnection(conn, stream, frame, state.flow, sessionID)
	terminalSent = sendSocksTerminal(stream, frame, sessionID, toImplantSequence)
}
