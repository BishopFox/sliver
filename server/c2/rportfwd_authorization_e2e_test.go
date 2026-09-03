//go:build server && go_sqlite && sliver_e2e

package c2_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"os"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	uuid "uuid"

	implantMTLS "github.com/bishopfox/sliver/implant/sliver/transports/mtls"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/c2"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/core/rtunnels"
	"github.com/bishopfox/sliver/server/transport"
	"github.com/hashicorp/yamux"
	"google.golang.org/protobuf/proto"
)

const rportAuthorizationE2ETimeout = 15 * time.Second

// TestMTLSYamux_ReversePortForwardAuthorizationCannotBePoisonedOrReplayed
// exercises the production RPC, C2 dispatch, cross-session listener isolation,
// authorization broker, TCP dial, and relay path with protocol peers that are
// intentionally not the production implant implementation. This matters for
// GHSA-w4h3-gj8x-mqr5: an honest implant cannot reproduce destination poisoning
// in a regression test.
func TestMTLSYamux_ReversePortForwardAuthorizationCannotBePoisonedOrReplayed(t *testing.T) {
	authorizedPayload := []byte("operator-authorized-target\x00\xff")
	authorizedTarget := startRportAuthorizationTarget(t, authorizedPayload, true)
	poisonTarget := startRportAuthorizationTarget(t, nil, false)

	grpcServer, grpcListener, err := transport.LocalListener()
	if err != nil {
		t.Fatalf("start local grpc listener: %v", err)
	}
	t.Cleanup(func() {
		grpcServer.Stop()
		_ = grpcListener.Close()
	})

	owner := startRportProtocolPeer(t, "authorization-owner", poisonTarget.address(), 73)
	attacker := startRportProtocolPeer(t, "authorization-attacker", "", 0)
	ownerSession := waitForRportProtocolSession(t, owner.registrationUUID)
	attackerSession := waitForRportProtocolSession(t, attacker.registrationUUID)
	if ownerSession.ID == attackerSession.ID || ownerSession.Connection.ID == attackerSession.Connection.ID {
		t.Fatalf("protocol peers did not receive independent connection/session identities: owner=%s/%s attacker=%s/%s",
			ownerSession.ID, ownerSession.Connection.ID, attackerSession.ID, attackerSession.Connection.ID)
	}
	t.Cleanup(func() {
		rtunnels.DefaultRegistry.RevokeSession(ownerSession.ID)
		rtunnels.CloseSession(ownerSession.ID)
		rtunnels.DefaultRegistry.RevokeSession(attackerSession.ID)
		rtunnels.CloseSession(attackerSession.ID)
		core.Sessions.Remove(ownerSession.ID)
		core.Sessions.Remove(attackerSession.ID)
	})

	rpcConnection, err := dialBufConn(context.Background(), grpcListener)
	if err != nil {
		t.Fatalf("dial grpc/bufconn: %v", err)
	}
	t.Cleanup(func() { _ = rpcConnection.Close() })
	rpcClient := rpcpb.NewSliverRPCClient(rpcConnection)

	operatorBindAddress := "127.0.0.1:31337"
	ctx, cancel := context.WithTimeout(context.Background(), rportAuthorizationE2ETimeout)
	listener, err := rpcClient.StartRportFwdListener(ctx, &sliverpb.RportFwdStartListenerReq{
		BindAddress:    operatorBindAddress,
		ForwardAddress: authorizedTarget.address(),
		KeepAlive:      5,
		Request: &commonpb.Request{
			SessionID: ownerSession.ID,
			Timeout:   int64(10 * time.Second),
		},
	})
	cancel()
	if err != nil {
		t.Fatalf("start reverse port forward through production RPC/C2 path: %v", err)
	}

	startRequest := receiveRportStartRequest(t, owner.startRequests)
	if startRequest.GetForwardAddress() != authorizedTarget.address() {
		t.Fatalf("implant received forward address %q, want operator target %q", startRequest.GetForwardAddress(), authorizedTarget.address())
	}
	if startRequest.GetAuthorizationID() == "" {
		t.Fatal("implant did not receive a server-issued authorization ID")
	}
	if listener.GetAuthorizationID() != startRequest.GetAuthorizationID() {
		t.Fatalf("RPC authorization ID = %q, want server-issued %q", listener.GetAuthorizationID(), startRequest.GetAuthorizationID())
	}
	if listener.GetForwardAddress() != authorizedTarget.address() {
		t.Fatalf("implant response poisoned RPC destination: got %q, want %q", listener.GetForwardAddress(), authorizedTarget.address())
	}
	if listener.GetForwardAddress() == poisonTarget.address() {
		t.Fatalf("RPC returned implant-controlled destination %q", listener.GetForwardAddress())
	}

	ctx, cancel = context.WithTimeout(context.Background(), rportAuthorizationE2ETimeout)
	inventory, err := rpcClient.GetRportFwdListeners(ctx, &sliverpb.RportFwdListenersReq{
		Request: &commonpb.Request{SessionID: ownerSession.ID},
	})
	cancel()
	if err != nil {
		t.Fatalf("get authoritative reverse port forward inventory: %v", err)
	}
	if len(inventory.GetListeners()) != 1 || inventory.GetListeners()[0].GetForwardAddress() != authorizedTarget.address() {
		t.Fatalf("authoritative inventory = %+v, want only operator target %q", inventory.GetListeners(), authorizedTarget.address())
	}

	// Listener IDs are implant-local numeric handles and can collide across
	// sessions. A stop issued through another valid session must be routed only
	// to that session's implant. It may succeed as a best-effort legacy cleanup,
	// but it must never revoke or mutate the owner's server-side authorization.
	ctx, cancel = context.WithTimeout(context.Background(), rportAuthorizationE2ETimeout)
	attackerStopResponse, err := rpcClient.StopRportFwdListener(ctx, &sliverpb.RportFwdStopListenerReq{
		ID: listener.GetID(),
		Request: &commonpb.Request{
			SessionID: attackerSession.ID,
			Timeout:   int64(10 * time.Second),
		},
	})
	cancel()
	if err != nil {
		t.Fatalf("stop owner listener ID through valid attacker session: %v", err)
	}
	attackerStopRequest := receiveRportStopRequest(t, attacker.stopRequests)
	if attackerStopRequest.GetID() != listener.GetID() {
		t.Fatalf("attacker implant stop ID = %d, want owner numeric ID %d", attackerStopRequest.GetID(), listener.GetID())
	}
	if attackerStopResponse.GetID() != listener.GetID() || attackerStopResponse.GetBindAddress() != "" || attackerStopResponse.GetForwardAddress() != "" || attackerStopResponse.GetAuthorizationID() != "" {
		t.Fatalf("cross-session cleanup response retained metadata or wrong ID: %+v", attackerStopResponse)
	}
	ownerAuthorization, ownerStillActive := rtunnels.DefaultRegistry.LookupListener(ownerSession.ID, listener.GetID())
	if !ownerStillActive {
		t.Fatal("valid attacker session revoked the owner's listener authorization")
	}
	if ownerAuthorization.State != rtunnels.AuthorizationActive || ownerAuthorization.AuthorizationID.String() != listener.GetAuthorizationID() || ownerAuthorization.Address != authorizedTarget.address() {
		t.Fatalf("valid attacker session mutated owner authorization: %+v", ownerAuthorization)
	}
	if _, attackerAuthorizationCreated := rtunnels.DefaultRegistry.LookupListener(attackerSession.ID, listener.GetID()); attackerAuthorizationCreated {
		t.Fatal("cross-session cleanup created an attacker authorization")
	}

	poisonHost, poisonPort := splitRportTargetAddress(t, poisonTarget.address())
	const replayTunnelID = uint64(0x7a110001)
	attacker.sendTunnelData(t, &sliverpb.TunnelData{
		TunnelID:      replayTunnelID,
		CreateReverse: true,
		Data:          []byte("cross-session-capability-replay"),
		Rportfwd: &sliverpb.RPortfwd{
			AuthorizationID: listener.GetAuthorizationID(),
			Host:            poisonHost,
			Port:            poisonPort,
		},
	})
	replayRejection := receiveRportTunnelMessage(t, attacker.tunnelCloses, replayTunnelID)
	if !replayRejection.GetClosed() {
		t.Fatalf("cross-session capability replay response = %+v, want closed rejection", replayRejection)
	}
	if got := authorizedTarget.accepted.Load(); got != 0 {
		t.Fatalf("cross-session capability replay dialed operator target %d times", got)
	}
	if got := poisonTarget.accepted.Load(); got != 0 {
		t.Fatalf("cross-session capability replay dialed implant target %d times", got)
	}

	// The owner now sends exactly the pair of malicious fields from the
	// advisory: its listener response already claimed target B, and this create
	// message also claims target B. The server must resolve the capability's
	// immutable target A, never either implant-controlled value.
	const ownerTunnelID = uint64(0x7a110002)
	owner.sendTunnelData(t, &sliverpb.TunnelData{
		TunnelID:      ownerTunnelID,
		CreateReverse: true,
		Data:          authorizedPayload,
		Rportfwd: &sliverpb.RPortfwd{
			AuthorizationID: listener.GetAuthorizationID(),
			Host:            poisonHost,
			Port:            poisonPort,
		},
	})

	receivedAtAuthorizedTarget := receiveRportTargetPayload(t, authorizedTarget)
	if !bytes.Equal(receivedAtAuthorizedTarget, authorizedPayload) {
		t.Fatalf("authorized target payload = %x, want %x", receivedAtAuthorizedTarget, authorizedPayload)
	}
	relayedBack := receiveRportTunnelMessage(t, owner.tunnelData, ownerTunnelID)
	if !bytes.Equal(relayedBack.GetData(), authorizedPayload) {
		t.Fatalf("relayed response = %x, want %x", relayedBack.GetData(), authorizedPayload)
	}
	if got := authorizedTarget.accepted.Load(); got != 1 {
		t.Fatalf("operator target dial count = %d, want 1", got)
	}
	if got := poisonTarget.accepted.Load(); got != 0 {
		t.Fatalf("implant-controlled target dial count = %d, want 0", got)
	}
}

type rportProtocolPeer struct {
	rawConnection    net.Conn
	mux              *yamux.Session
	registrationUUID string
	poisonAddress    string
	listenerID       uint32
	startRequests    chan *sliverpb.RportFwdStartListenerReq
	stopRequests     chan *sliverpb.RportFwdStopListenerReq
	tunnelData       chan *sliverpb.TunnelData
	tunnelCloses     chan *sliverpb.TunnelData
	done             chan struct{}
}

func startRportProtocolPeer(t *testing.T, name string, poisonAddress string, listenerID uint32) *rportProtocolPeer {
	t.Helper()
	serverConnection, implantConnection := net.Pipe()
	go c2.HandleSliverConnectionForTest(serverConnection)

	if _, err := implantConnection.Write([]byte(implantMTLS.YamuxPreface)); err != nil {
		_ = implantConnection.Close()
		t.Fatalf("write yamux preface for %s: %v", name, err)
	}
	config := yamux.DefaultConfig()
	config.LogOutput = io.Discard
	muxSession, err := yamux.Client(implantConnection, config)
	if err != nil {
		_ = implantConnection.Close()
		t.Fatalf("start yamux protocol peer %s: %v", name, err)
	}

	peer := &rportProtocolPeer{
		rawConnection:    implantConnection,
		mux:              muxSession,
		registrationUUID: uuid.NewV4().String(),
		poisonAddress:    poisonAddress,
		listenerID:       listenerID,
		startRequests:    make(chan *sliverpb.RportFwdStartListenerReq, 1),
		stopRequests:     make(chan *sliverpb.RportFwdStopListenerReq, 1),
		tunnelData:       make(chan *sliverpb.TunnelData, 8),
		tunnelCloses:     make(chan *sliverpb.TunnelData, 8),
		done:             make(chan struct{}),
	}

	registerData, err := proto.Marshal(&sliverpb.Register{
		Name:         name,
		Hostname:     "localhost",
		Uuid:         peer.registrationUUID,
		Username:     "protocol-test",
		Os:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		Pid:          int32(os.Getpid()),
		Filename:     "rportfwd-malicious-protocol-peer",
		ActiveC2:     "mtls://protocol-test",
		Version:      "protocol-test",
		Locale:       "en_US",
		Capabilities: sliverpb.CapabilityTunnelTerminalV1,
	})
	if err != nil {
		peer.close()
		t.Fatalf("marshal %s registration: %v", name, err)
	}
	if err := sendYamuxEnvelope(peer.mux, &sliverpb.Envelope{Type: sliverpb.MsgRegister, Data: registerData}); err != nil {
		peer.close()
		t.Fatalf("send %s registration: %v", name, err)
	}

	go peer.acceptLoop()
	t.Cleanup(peer.close)
	return peer
}

func (peer *rportProtocolPeer) acceptLoop() {
	defer close(peer.done)
	for {
		stream, err := peer.mux.Accept()
		if err != nil {
			return
		}
		go peer.handleServerEnvelope(stream)
	}
}

func (peer *rportProtocolPeer) handleServerEnvelope(stream net.Conn) {
	defer stream.Close()
	envelope, err := implantMTLS.ReadEnvelope(stream)
	if err != nil || envelope == nil {
		return
	}

	switch envelope.Type {
	case sliverpb.MsgRportFwdStartListenerReq:
		request := &sliverpb.RportFwdStartListenerReq{}
		if proto.Unmarshal(envelope.Data, request) != nil {
			return
		}
		select {
		case peer.startRequests <- request:
		default:
		}
		responseData, marshalErr := proto.Marshal(&sliverpb.RportFwdListener{
			ID:              peer.listenerID,
			BindAddress:     "implant-controlled-bind:1",
			ForwardAddress:  peer.poisonAddress,
			AuthorizationID: request.GetAuthorizationID(),
			Response:        &commonpb.Response{},
		})
		if marshalErr == nil {
			_ = sendYamuxEnvelope(peer.mux, &sliverpb.Envelope{ID: envelope.ID, Data: responseData})
		}
	case sliverpb.MsgRportFwdStopListenerReq:
		request := &sliverpb.RportFwdStopListenerReq{}
		if proto.Unmarshal(envelope.Data, request) != nil {
			return
		}
		select {
		case peer.stopRequests <- request:
		default:
		}
		responseData, marshalErr := proto.Marshal(&sliverpb.RportFwdListener{
			ID:       request.GetID(),
			Response: &commonpb.Response{},
		})
		if marshalErr == nil {
			_ = sendYamuxEnvelope(peer.mux, &sliverpb.Envelope{ID: envelope.ID, Data: responseData})
		}
	case sliverpb.MsgTunnelData:
		data := &sliverpb.TunnelData{}
		if proto.Unmarshal(envelope.Data, data) == nil {
			select {
			case peer.tunnelData <- data:
			default:
			}
		}
	case sliverpb.MsgTunnelClose:
		data := &sliverpb.TunnelData{}
		if proto.Unmarshal(envelope.Data, data) == nil {
			select {
			case peer.tunnelCloses <- data:
			default:
			}
		}
	default:
		if envelope.ID != 0 {
			_ = sendYamuxEnvelope(peer.mux, &sliverpb.Envelope{ID: envelope.ID, UnknownMessageType: true})
		}
	}
}

func (peer *rportProtocolPeer) sendTunnelData(t *testing.T, tunnelData *sliverpb.TunnelData) {
	t.Helper()
	data, err := proto.Marshal(tunnelData)
	if err != nil {
		t.Fatalf("marshal tunnel data: %v", err)
	}
	if err := sendYamuxEnvelope(peer.mux, &sliverpb.Envelope{Type: sliverpb.MsgTunnelData, Data: data}); err != nil {
		t.Fatalf("send tunnel data: %v", err)
	}
}

func (peer *rportProtocolPeer) close() {
	_ = peer.mux.Close()
	_ = peer.rawConnection.Close()
	select {
	case <-peer.done:
	case <-time.After(5 * time.Second):
	}
}

func waitForRportProtocolSession(t *testing.T, registrationUUID string) *core.Session {
	t.Helper()
	deadline := time.Now().Add(rportAuthorizationE2ETimeout)
	for time.Now().Before(deadline) {
		for _, session := range core.Sessions.All() {
			if session.UUID == registrationUUID {
				return session
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for protocol session UUID %s", registrationUUID)
	return nil
}

func receiveRportStartRequest(t *testing.T, requests <-chan *sliverpb.RportFwdStartListenerReq) *sliverpb.RportFwdStartListenerReq {
	t.Helper()
	select {
	case request := <-requests:
		return request
	case <-time.After(rportAuthorizationE2ETimeout):
		t.Fatal("timed out waiting for reverse port forward start request")
		return nil
	}
}

func receiveRportStopRequest(t *testing.T, requests <-chan *sliverpb.RportFwdStopListenerReq) *sliverpb.RportFwdStopListenerReq {
	t.Helper()
	select {
	case request := <-requests:
		return request
	case <-time.After(rportAuthorizationE2ETimeout):
		t.Fatal("timed out waiting for reverse port forward stop request")
		return nil
	}
}

func receiveRportTunnelMessage(t *testing.T, messages <-chan *sliverpb.TunnelData, tunnelID uint64) *sliverpb.TunnelData {
	t.Helper()
	timer := time.NewTimer(rportAuthorizationE2ETimeout)
	defer timer.Stop()
	for {
		select {
		case message := <-messages:
			if message.GetTunnelID() == tunnelID {
				return message
			}
		case <-timer.C:
			t.Fatalf("timed out waiting for reverse tunnel %d message", tunnelID)
			return nil
		}
	}
}

func splitRportTargetAddress(t *testing.T, address string) (string, uint32) {
	t.Helper()
	host, portText, err := net.SplitHostPort(address)
	if err != nil {
		t.Fatalf("split target address %q: %v", address, err)
	}
	port, err := strconv.ParseUint(portText, 10, 16)
	if err != nil {
		t.Fatalf("parse target port %q: %v", portText, err)
	}
	return host, uint32(port)
}

type rportAuthorizationTarget struct {
	listener    net.Listener
	wantPayload []byte
	echo        bool
	payloads    chan []byte
	errors      chan error
	done        chan struct{}
	accepted    atomic.Int64
	connections sync.Map
}

func startRportAuthorizationTarget(t *testing.T, wantPayload []byte, echo bool) *rportAuthorizationTarget {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for reverse port forward target: %v", err)
	}
	target := &rportAuthorizationTarget{
		listener:    listener,
		wantPayload: append([]byte(nil), wantPayload...),
		echo:        echo,
		payloads:    make(chan []byte, 8),
		errors:      make(chan error, 8),
		done:        make(chan struct{}),
	}
	go target.serve()
	t.Cleanup(target.close)
	return target
}

func (target *rportAuthorizationTarget) address() string {
	return target.listener.Addr().String()
}

func (target *rportAuthorizationTarget) serve() {
	defer close(target.done)
	for {
		connection, err := target.listener.Accept()
		if err != nil {
			if !isClosedRportTargetError(err) {
				select {
				case target.errors <- err:
				default:
				}
			}
			return
		}
		target.accepted.Add(1)
		target.connections.Store(connection, struct{}{})
		go target.handleConnection(connection)
	}
}

func (target *rportAuthorizationTarget) handleConnection(connection net.Conn) {
	defer target.connections.Delete(connection)
	defer connection.Close()
	if len(target.wantPayload) == 0 {
		return
	}
	_ = connection.SetDeadline(time.Now().Add(rportAuthorizationE2ETimeout))
	payload := make([]byte, len(target.wantPayload))
	if _, err := io.ReadFull(connection, payload); err != nil {
		select {
		case target.errors <- err:
		default:
		}
		return
	}
	select {
	case target.payloads <- payload:
	default:
	}
	if target.echo {
		if _, err := connection.Write(payload); err != nil {
			select {
			case target.errors <- err:
			default:
			}
		}
	}
}

func (target *rportAuthorizationTarget) close() {
	_ = target.listener.Close()
	target.connections.Range(func(connection, _ any) bool {
		_ = connection.(net.Conn).Close()
		return true
	})
	select {
	case <-target.done:
	case <-time.After(5 * time.Second):
	}
}

func receiveRportTargetPayload(t *testing.T, target *rportAuthorizationTarget) []byte {
	t.Helper()
	select {
	case payload := <-target.payloads:
		return payload
	case err := <-target.errors:
		t.Fatalf("operator-authorized target relay error: %v", err)
		return nil
	case <-time.After(rportAuthorizationE2ETimeout):
		t.Fatal("timed out waiting for data at operator-authorized target")
		return nil
	}
}

func isClosedRportTargetError(err error) bool {
	return errors.Is(err, net.ErrClosed)
}
