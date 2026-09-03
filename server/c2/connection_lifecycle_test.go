package c2

import (
	"bytes"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	implantCrypto "github.com/bishopfox/sliver/implant/sliver/cryptography"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/core/rtunnels"
	serverCrypto "github.com/bishopfox/sliver/server/cryptography"
	serverHandlers "github.com/bishopfox/sliver/server/handlers"
	"github.com/hashicorp/yamux"
)

type lifecycleWriteCloser struct {
	closed atomic.Bool
}

type lifecycleIdentityEncoder struct{}

func (lifecycleIdentityEncoder) Encode(data []byte) ([]byte, error) { return data, nil }
func (lifecycleIdentityEncoder) Decode(data []byte) ([]byte, error) { return data, nil }

func (writer *lifecycleWriteCloser) Write(data []byte) (int, error) {
	return len(data), nil
}

func (writer *lifecycleWriteCloser) Close() error {
	writer.closed.Store(true)
	return nil
}

func addLifecycleReverseState(t *testing.T, sessionID string) (rtunnels.AuthorizationID, *rtunnels.RTunnel, *lifecycleWriteCloser) {
	t.Helper()
	authorizationID, err := rtunnels.DefaultRegistry.BeginSpec(sessionID, "127.0.0.1:4443", "127.0.0.1:4444", 0)
	if err != nil {
		t.Fatalf("begin reverse authorization: %s", err)
	}
	if err := rtunnels.DefaultRegistry.Activate(sessionID, authorizationID, 1); err != nil {
		t.Fatalf("activate reverse authorization: %s", err)
	}

	writer := &lifecycleWriteCloser{}
	tunnel := rtunnels.NewAuthorizedRTunnel(core.NewTunnelID(), sessionID, authorizationID, writer)
	if !rtunnels.TryAddRTunnel(tunnel) {
		t.Fatal("failed to add reverse tunnel")
	}
	t.Cleanup(func() {
		rtunnels.DefaultRegistry.RevokeSession(sessionID)
		rtunnels.CloseSession(sessionID)
	})
	return authorizationID, tunnel, writer
}

func assertLifecycleReverseStateClosed(t *testing.T, sessionID string, authorizationID rtunnels.AuthorizationID, tunnel *rtunnels.RTunnel, writer *lifecycleWriteCloser) {
	t.Helper()
	if _, ok := rtunnels.DefaultRegistry.Lookup(sessionID, authorizationID); ok {
		t.Fatal("reverse authorization survived connection cleanup")
	}
	if authorizations := rtunnels.DefaultRegistry.List(sessionID); len(authorizations) != 0 {
		t.Fatalf("%d reverse authorizations survived connection cleanup", len(authorizations))
	}
	if got := rtunnels.GetRTunnel(tunnel.ID); got != nil {
		t.Fatal("reverse tunnel survived connection cleanup")
	}
	select {
	case <-tunnel.Done():
	default:
		t.Fatal("reverse tunnel Done was not closed")
	}
	if !writer.closed.Load() {
		t.Fatal("reverse tunnel writer was not closed")
	}
}

func TestHTTPCloseHandlerClosesCoreAndReverseState(t *testing.T) {
	connection := core.NewImplantConnection("http", "test")
	coreSession := core.NewSession(connection)
	core.Sessions.Add(coreSession)
	t.Cleanup(func() { core.Sessions.Remove(coreSession.ID) })

	var cleanupCalls atomic.Int32
	if !connection.SetCleanup(func() {
		cleanupCalls.Add(1)
		core.Sessions.Remove(coreSession.ID)
	}) {
		t.Fatal("failed to install HTTP cleanup")
	}
	authorizationID, tunnel, writer := addLifecycleReverseState(t, coreSession.ID)

	httpSession := &HTTPSession{ID: connection.ID, ImplantConn: connection}
	server := &SliverHTTPC2{
		HTTPSessions: &HTTPSessions{
			active: map[string]*HTTPSession{},
			mutex:  &sync.RWMutex{},
		},
	}
	server.HTTPSessions.Add(httpSession)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/close", nil)
	server.closeHandler(recorder, request, httpSession)
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("close status = %d, want %d", recorder.Code, http.StatusAccepted)
	}
	if got := server.HTTPSessions.Get(httpSession.ID); got != nil {
		t.Fatal("HTTP session survived close handler")
	}
	if got := core.Sessions.Get(coreSession.ID); got != nil {
		t.Fatal("core session survived close handler")
	}
	select {
	case <-connection.Done():
	default:
		t.Fatal("implant connection Done was not closed")
	}
	assertLifecycleReverseStateClosed(t, coreSession.ID, authorizationID, tunnel, writer)

	server.closeHandler(httptest.NewRecorder(), request, httpSession)
	if got := cleanupCalls.Load(); got != 1 {
		t.Fatalf("cleanup called %d times, want 1", got)
	}
}

func TestHTTPConnectionCloseRemovesTransportSessionAndPoll(t *testing.T) {
	connection := core.NewImplantConnection("http", "test")
	t.Cleanup(connection.Close)
	httpSession := &HTTPSession{ID: connection.ID, ImplantConn: connection}
	server := &SliverHTTPC2{
		ServerConf: &clientpb.HTTPListenerReq{
			LongPollTimeout: int64(time.Hour),
		},
		HTTPSessions: &HTTPSessions{
			active: map[string]*HTTPSession{},
			mutex:  &sync.RWMutex{},
		},
	}
	server.addHTTPSession(httpSession)

	pollDone := make(chan struct{})
	go func() {
		defer close(pollDone)
		server.pollHandler(
			httptest.NewRecorder(),
			httptest.NewRequest(http.MethodGet, "/poll", nil),
			httpSession,
			lifecycleIdentityEncoder{},
		)
	}()
	deadline := time.Now().Add(time.Second)
	for connection.GetLastMessage().IsZero() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if connection.GetLastMessage().IsZero() {
		t.Fatal("HTTP poll did not enter the long-poll path before close")
	}
	connection.Close()

	select {
	case <-pollDone:
	case <-time.After(time.Second):
		t.Fatal("HTTP poll survived implant connection close")
	}
	deadline = time.Now().Add(time.Second)
	for server.HTTPSessions.Get(httpSession.ID) != nil && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if got := server.HTTPSessions.Get(httpSession.ID); got != nil {
		t.Fatal("HTTP transport session survived implant connection close")
	}

	request := httptest.NewRequest(http.MethodGet, "/session", nil)
	request.AddCookie(&http.Cookie{Value: httpSession.ID})
	if got := server.getHTTPSession(request); got != nil {
		t.Fatal("closed HTTP transport session was accepted")
	}
}

func TestYamuxTransportStopsDispatchAfterImplantConnectionClose(t *testing.T) {
	tests := []struct {
		name    string
		handler func(net.Conn, *core.ImplantConnection)
	}{
		{name: "mtls", handler: handleSliverConnectionYamux},
		{name: "wireguard", handler: handleWGSliverConnectionYamux},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			serverSide, implantSide := net.Pipe()
			t.Cleanup(func() {
				_ = serverSide.Close()
				_ = implantSide.Close()
			})
			connection := core.NewImplantConnection(test.name, "test")
			returned := make(chan struct{})
			go func() {
				defer close(returned)
				test.handler(serverSide, connection)
			}()

			clientSession, err := yamux.Client(implantSide, nil)
			if err != nil {
				t.Fatalf("create yamux client: %s", err)
			}
			t.Cleanup(func() { _ = clientSession.Close() })
			connection.Close()

			select {
			case <-returned:
			case <-time.After(time.Second):
				t.Fatal("yamux transport survived implant connection close")
			}
			type openResult struct {
				connection net.Conn
				err        error
			}
			opened := make(chan openResult, 1)
			go func() {
				stream, openErr := clientSession.Open()
				opened <- openResult{connection: stream, err: openErr}
			}()
			select {
			case result := <-opened:
				if result.connection != nil {
					_ = result.connection.Close()
				}
				if result.err == nil {
					t.Fatal("yamux transport accepted a stream after implant connection close")
				}
			case <-time.After(time.Second):
				t.Fatal("yamux stream open did not return after implant connection close")
			}
		})
	}
}

//nolint:gocyclo // This table-driven lifecycle test keeps both transports on the same synchronized scenario.
func TestYamuxInFlightStreamDoesNotDispatchAfterImplantConnectionClose(t *testing.T) {
	const messageType = uint32(0x7ffffffe)
	frame, keyID, publicKey := newImplantSignedEnvelopeFrame(t, "lifecycle-dispatch-peer", &sliverpb.Envelope{
		Type: messageType,
		Data: []byte("must-not-dispatch"),
	})
	tests := []struct {
		name    string
		handler func(net.Conn, *core.ImplantConnection, map[uint32]serverHandlers.ServerHandler, func(), func())
	}{
		{name: "mtls", handler: handleSliverConnectionYamuxWithDispatch},
		{name: "wireguard", handler: handleWGSliverConnectionYamuxWithDispatch},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clearMTLSImplantSigKeyCache(t)
			mtlsImplantSigKeyCache.Store(keyID, publicKey)
			t.Cleanup(func() { mtlsImplantSigKeyCache.Delete(keyID) })

			serverSide, implantSide := net.Pipe()
			t.Cleanup(func() {
				_ = serverSide.Close()
				_ = implantSide.Close()
			})
			connection := core.NewImplantConnection(test.name, "test")
			t.Cleanup(connection.Close)
			var handlerCalls atomic.Int32
			handlers := map[uint32]serverHandlers.ServerHandler{
				messageType: func(*core.ImplantConnection, []byte) *sliverpb.Envelope {
					handlerCalls.Add(1)
					return nil
				},
			}
			readComplete := make(chan struct{})
			releaseDispatch := make(chan struct{})
			dispatchReleased := make(chan struct{})
			streamFinished := make(chan struct{})
			beforeDispatch := func() {
				close(readComplete)
				<-releaseDispatch
				close(dispatchReleased)
			}
			returned := make(chan struct{})
			go func() {
				defer close(returned)
				test.handler(serverSide, connection, handlers, beforeDispatch, func() { close(streamFinished) })
			}()

			clientSession, err := yamux.Client(implantSide, nil)
			if err != nil {
				t.Fatalf("create yamux client: %s", err)
			}
			t.Cleanup(func() { _ = clientSession.Close() })
			type openResult struct {
				stream net.Conn
				err    error
			}
			opened := make(chan openResult, 1)
			go func() {
				stream, openErr := clientSession.Open()
				opened <- openResult{stream: stream, err: openErr}
			}()
			var stream net.Conn
			select {
			case result := <-opened:
				if result.err != nil {
					t.Fatalf("open yamux stream: %v", result.err)
				}
				stream = result.stream
			case <-time.After(time.Second):
				t.Fatal("opening in-flight yamux stream timed out")
			}
			t.Cleanup(func() { _ = stream.Close() })
			_ = stream.SetWriteDeadline(time.Now().Add(time.Second))
			writeDone := make(chan error, 1)
			go func() {
				_, writeErr := io.Copy(stream, bytes.NewReader(frame))
				writeDone <- writeErr
			}()
			select {
			case <-readComplete:
			case writeErr := <-writeDone:
				if writeErr != nil {
					t.Fatalf("write in-flight yamux envelope: %v", writeErr)
				}
				select {
				case <-readComplete:
				case <-time.After(time.Second):
					t.Fatal("server read did not reach the pre-dispatch barrier")
				}
			case <-time.After(time.Second):
				t.Fatal("server did not read the in-flight yamux envelope")
			}

			connection.Close()
			close(releaseDispatch)
			select {
			case <-dispatchReleased:
			case <-time.After(time.Second):
				t.Fatal("pre-dispatch barrier did not release")
			}
			select {
			case <-streamFinished:
			case <-time.After(time.Second):
				t.Fatal("in-flight yamux stream did not finish after close")
			}
			select {
			case <-returned:
			case <-time.After(time.Second):
				t.Fatal("yamux transport survived in-flight close")
			}
			if got := handlerCalls.Load(); got != 0 {
				t.Fatalf("in-flight handler ran %d times after implant connection close", got)
			}
		})
	}
}

func TestDNSSessionOutgoingRetentionLimitsAndClear(t *testing.T) {
	key := implantCrypto.RandomSymmetricKey()
	dnsSession := newTestDNSSession(0x123456)
	dnsSession.CipherCtx = serverCrypto.NewCipherContext(key)
	dnsSession.maxOutgoingMessages = 1
	dnsSession.maxOutgoingBytes = 1024 * 1024

	first := &sliverpb.Envelope{Type: sliverpb.MsgPing, Data: []byte("first")}
	if err := dnsSession.StageOutgoingEnvelope(first); err != nil {
		t.Fatalf("stage first envelope: %s", err)
	}
	if dnsSession.outgoingBytes == 0 {
		t.Fatal("staged envelope did not consume byte budget")
	}
	if len(dnsSession.outgoingMsgIDs) != 1 {
		t.Fatalf("staged message IDs = %d, want 1", len(dnsSession.outgoingMsgIDs))
	}
	firstID := dnsSession.outgoingMsgIDs[0]

	if err := dnsSession.StageOutgoingEnvelope(&sliverpb.Envelope{Type: sliverpb.MsgPing}); !errors.Is(err, ErrDNSOutgoingLimit) {
		t.Fatalf("second stage error = %v, want %v", err, ErrDNSOutgoingLimit)
	}
	dnsSession.ClearOutgoingEnvelope(firstID)
	if dnsSession.outgoingBytes != 0 {
		t.Fatalf("outgoing bytes after clear = %d, want 0", dnsSession.outgoingBytes)
	}
	if len(dnsSession.outgoingBuffers) != 0 || len(dnsSession.outgoingMsgIDs) != 0 {
		t.Fatal("clear retained outgoing message state")
	}
	if err := dnsSession.StageOutgoingEnvelope(&sliverpb.Envelope{Type: sliverpb.MsgPing}); err != nil {
		t.Fatalf("stage after clear: %s", err)
	}

	byteLimited := newTestDNSSession(0x654321)
	byteLimited.CipherCtx = serverCrypto.NewCipherContext(key)
	byteLimited.maxOutgoingMessages = 10
	byteLimited.maxOutgoingBytes = 1
	if err := byteLimited.StageOutgoingEnvelope(first); !errors.Is(err, ErrDNSOutgoingLimit) {
		t.Fatalf("byte-limited stage error = %v, want %v", err, ErrDNSOutgoingLimit)
	}
	if byteLimited.outgoingBytes != 0 || len(byteLimited.outgoingBuffers) != 0 {
		t.Fatal("rejected byte-limited envelope consumed retention budget")
	}
}

func TestDNSNoPollBackpressureClosesSessionAndReverseState(t *testing.T) {
	server := newTestDNSServer()
	dnsSession := newTestDNSSession(0x456789)
	dnsSession.CipherCtx = serverCrypto.NewCipherContext(implantCrypto.RandomSymmetricKey())
	dnsSession.ImplantConn = core.NewImplantConnection("dns", "test")
	dnsSession.maxOutgoingMessages = 2
	dnsSession.maxOutgoingBytes = 1024 * 1024
	server.sessions.Store(dnsSession.ID, dnsSession)

	coreSession := core.NewSession(dnsSession.ImplantConn)
	core.Sessions.Add(coreSession)
	t.Cleanup(func() {
		server.closeDNSSession(dnsSession)
		core.Sessions.Remove(coreSession.ID)
	})
	cleanupComplete := make(chan struct{})
	if !dnsSession.ImplantConn.SetCleanup(func() {
		defer close(cleanupComplete)
		core.Sessions.Remove(coreSession.ID)
	}) {
		t.Fatal("failed to install DNS cleanup")
	}
	authorizationID, tunnel, writer := addLifecycleReverseState(t, coreSession.ID)
	server.startDNSSessionSendLoop(dnsSession)

	for index := range 3 {
		envelope := &sliverpb.Envelope{Type: sliverpb.MsgPing, Data: []byte{byte(index)}}
		select {
		case dnsSession.ImplantConn.Send <- envelope:
		case <-time.After(time.Second):
			t.Fatalf("send envelope %d blocked", index)
		}
	}

	select {
	case <-dnsSession.ImplantConn.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("DNS session did not close after outgoing retention limit")
	}
	select {
	case <-cleanupComplete:
	case <-time.After(2 * time.Second):
		t.Fatal("DNS session cleanup did not complete")
	}
	if _, ok := server.sessions.Load(dnsSession.ID); ok {
		t.Fatal("DNS transport session survived outgoing retention limit")
	}
	if got := core.Sessions.Get(coreSession.ID); got != nil {
		t.Fatal("core session survived DNS outgoing retention limit")
	}
	if dnsSession.outgoingBytes != 0 || len(dnsSession.outgoingBuffers) != 0 || len(dnsSession.outgoingMsgIDs) != 0 {
		t.Fatal("DNS session retained outgoing state after close")
	}
	assertLifecycleReverseStateClosed(t, coreSession.ID, authorizationID, tunnel, writer)
}
