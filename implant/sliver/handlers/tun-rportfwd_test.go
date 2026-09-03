package handlers

import (
	"net"
	"sync"
	"testing"
	"time"

	"github.com/bishopfox/sliver/implant/sliver/rportfwd"
	"github.com/bishopfox/sliver/implant/sliver/transports"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

type rportfwdDiscardWriteCloser struct{}

func (rportfwdDiscardWriteCloser) Write(data []byte) (int, error) { return len(data), nil }
func (rportfwdDiscardWriteCloser) Close() error                   { return nil }

func TestRportFwdListenerClosesWithOwningConnectionAndAllowsFreshBind(t *testing.T) {
	isolateRportFwdListeners(t)
	bindAddress := reserveRportFwdBindAddress(t)

	firstConnection := newRportFwdTestConnection()
	first := startRportFwdTestListener(t, firstConnection, bindAddress, "first-authorization", 1)
	if got := len(rportfwd.Portfwds.List()); got != 1 {
		t.Fatalf("listener count = %d, want 1", got)
	}

	firstConnection.Cleanup()
	waitForRportFwdListenerCount(t, 0)
	waitForRportFwdBindRelease(t, bindAddress)

	secondConnection := newRportFwdTestConnection()
	second := startRportFwdTestListener(t, secondConnection, bindAddress, "second-authorization", 2)
	if second.ID == first.ID {
		t.Fatalf("fresh connection reused listener generation %d", second.ID)
	}
	if second.AuthorizationID != "second-authorization" {
		t.Fatalf("fresh listener authorization = %q", second.AuthorizationID)
	}
	secondConnection.Cleanup()
	waitForRportFwdListenerCount(t, 0)
	waitForRportFwdBindRelease(t, bindAddress)
}

func TestRportFwdListenerSurvivesRetiredTunnelWindowRotation(t *testing.T) {
	isolateRportFwdListeners(t)
	bindAddress := reserveRportFwdBindAddress(t)
	connection := newRportFwdTestConnection()
	startRportFwdTestListener(t, connection, bindAddress, "rotated-authorization", 3)

	for tunnelID := uint64(1); tunnelID <= 5_000; tunnelID++ {
		tunnel := transports.NewTunnel(tunnelID, rportfwdDiscardWriteCloser{})
		if result := connection.TryAddTunnel(tunnel); result != transports.TunnelAdded {
			tunnel.Close()
			t.Fatalf("claim tunnel %d result = %v, want %v", tunnelID, result, transports.TunnelAdded)
		}
		if !connection.CloseTunnelRemote(tunnel) {
			t.Fatalf("failed to retire tunnel %d", tunnelID)
		}
	}
	select {
	case <-connection.Done():
		t.Fatal("retired tunnel window rotation closed the owning connection")
	default:
	}
	if got := len(rportfwd.Portfwds.List()); got != 1 {
		t.Fatalf("listener count after replay-window rotation = %d, want 1", got)
	}

	connection.Cleanup()
	waitForRportFwdListenerCount(t, 0)
	waitForRportFwdBindRelease(t, bindAddress)

	freshConnection := newRportFwdTestConnection()
	startRportFwdTestListener(t, freshConnection, bindAddress, "fresh-authorization", 4)
	freshConnection.Cleanup()
	waitForRportFwdListenerCount(t, 0)
	waitForRportFwdBindRelease(t, bindAddress)
}

func TestRportFwdListenerFailedResponseAndStopCleanupRacesDoNotLeak(t *testing.T) {
	t.Run("failed start response", func(t *testing.T) {
		isolateRportFwdListeners(t)
		bindAddress := reserveRportFwdBindAddress(t)
		connection := &transports.Connection{}
		request := marshalRportFwdStartRequest(t, bindAddress, "failed-response-authorization")
		rportFwdStartListenerHandler(&sliverpb.Envelope{ID: 5, Data: request}, connection)
		waitForRportFwdListenerCount(t, 0)
		waitForRportFwdBindRelease(t, bindAddress)
		connection.Cleanup()
	})

	t.Run("explicit stop and connection close", func(t *testing.T) {
		for iteration := 0; iteration < 32; iteration++ {
			isolateRPortFwdListenersNow()
			bindAddress := reserveRportFwdBindAddress(t)
			connection := newRportFwdTestConnection()
			listener := startRportFwdTestListener(t, connection, bindAddress, "race-authorization", int64(100+iteration))
			stopData, err := proto.Marshal(&sliverpb.RportFwdStopListenerReq{ID: listener.ID})
			if err != nil {
				t.Fatalf("marshal stop request: %v", err)
			}

			start := make(chan struct{})
			var waitGroup sync.WaitGroup
			waitGroup.Add(2)
			go func() {
				defer waitGroup.Done()
				<-start
				connection.Cleanup()
			}()
			go func() {
				defer waitGroup.Done()
				<-start
				rportFwdStopListenerHandler(&sliverpb.Envelope{ID: int64(200 + iteration), Data: stopData}, connection)
			}()
			close(start)
			waitGroup.Wait()
			waitForRportFwdListenerCount(t, 0)
			waitForRportFwdBindRelease(t, bindAddress)
		}
		isolateRPortFwdListenersNow()
	})
}

func newRportFwdTestConnection() *transports.Connection {
	return &transports.Connection{Send: make(chan *sliverpb.Envelope, 4)}
}

func startRportFwdTestListener(t *testing.T, connection *transports.Connection, bindAddress string, authorizationID string, envelopeID int64) *sliverpb.RportFwdListener {
	t.Helper()
	request := marshalRportFwdStartRequest(t, bindAddress, authorizationID)
	rportFwdStartListenerHandler(&sliverpb.Envelope{ID: envelopeID, Data: request}, connection)
	select {
	case envelope := <-connection.Send:
		response := &sliverpb.RportFwdListener{}
		if err := proto.Unmarshal(envelope.Data, response); err != nil {
			t.Fatalf("decode start response: %v", err)
		}
		if response.Response == nil {
			t.Fatal("start response omitted status")
		}
		if response.Response.Err != "" {
			t.Fatalf("start listener response error: %s", response.Response.Err)
		}
		return response
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for start response")
		return nil
	}
}

func marshalRportFwdStartRequest(t *testing.T, bindAddress string, authorizationID string) []byte {
	t.Helper()
	request, err := proto.Marshal(&sliverpb.RportFwdStartListenerReq{
		BindAddress:     bindAddress,
		ForwardAddress:  "127.0.0.1:4444",
		AuthorizationID: authorizationID,
	})
	if err != nil {
		t.Fatalf("marshal start request: %v", err)
	}
	return request
}

func reserveRportFwdBindAddress(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve loopback listener: %v", err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("release reserved loopback listener: %v", err)
	}
	return address
}

func waitForRportFwdListenerCount(t *testing.T, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if len(rportfwd.Portfwds.List()) == want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("listener count = %d, want %d", len(rportfwd.Portfwds.List()), want)
}

func waitForRportFwdBindRelease(t *testing.T, bindAddress string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		listener, err := net.Listen("tcp", bindAddress)
		if err == nil {
			_ = listener.Close()
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("listener bind %s was not released", bindAddress)
}

func isolateRportFwdListeners(t *testing.T) {
	t.Helper()
	isolateRPortFwdListenersNow()
	t.Cleanup(isolateRPortFwdListenersNow)
}

func isolateRPortFwdListenersNow() {
	for _, listener := range rportfwd.Portfwds.List() {
		rportfwd.Portfwds.Remove(listener.ID)
	}
}
