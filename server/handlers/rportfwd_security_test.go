package handlers

import (
	"context"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/core/rtunnels"
	"google.golang.org/protobuf/proto"
)

func TestReverseTunnelAuthorizationControlsDialDestination(t *testing.T) {
	const (
		operatorTarget = "127.0.0.1:4444"
		tunnelID       = uint64(0x1001)
	)
	registry, dialer, broker := newRecordingReverseForwardBroker(t)
	connection, session := addBufferedTestSession(t)
	authorizationID := beginActiveAuthorization(t, registry, session.ID, operatorTarget, 1)

	sendReverseTunnelRequest(t, connection, broker, &sliverpb.TunnelData{
		TunnelID:      tunnelID,
		CreateReverse: true,
		Rportfwd: &sliverpb.RPortfwd{
			AuthorizationID: authorizationID.String(),
			Host:            "169.254.169.254",
			Port:            80,
		},
	})

	assertDialAddresses(t, dialer, operatorTarget)
	tunnel := rtunnels.GetRTunnel(tunnelID)
	if tunnel == nil {
		t.Fatalf("authorized request did not create reverse tunnel %d", tunnelID)
	}
	if got := tunnel.AuthorizationID(); got != authorizationID {
		t.Fatalf("tunnel authorization = %q, want %q", got, authorizationID)
	}
	t.Cleanup(func() {
		closeTestReverseTunnel(tunnelID)
	})
}

func TestReverseTunnelAuthorizationRejectsInvalidCapabilities(t *testing.T) {
	tests := []struct {
		name      string
		prepareID func(t *testing.T, registry *rtunnels.Registry, ownerSession string) rtunnels.AuthorizationID
		attacker  bool
	}{
		{
			name: "unknown authorization",
			prepareID: func(t *testing.T, registry *rtunnels.Registry, ownerSession string) rtunnels.AuthorizationID {
				return rtunnels.AuthorizationID("unknown-authorization")
			},
		},
		{
			name: "wrong session",
			prepareID: func(t *testing.T, registry *rtunnels.Registry, ownerSession string) rtunnels.AuthorizationID {
				return beginActiveAuthorization(t, registry, ownerSession, "127.0.0.1:4444", 2)
			},
			attacker: true,
		},
		{
			name: "revoked authorization",
			prepareID: func(t *testing.T, registry *rtunnels.Registry, ownerSession string) rtunnels.AuthorizationID {
				authorizationID := beginActiveAuthorization(t, registry, ownerSession, "127.0.0.1:4444", 3)
				if !registry.Revoke(ownerSession, authorizationID) {
					t.Fatal("Revoke() = false, want true")
				}
				return authorizationID
			},
		},
	}

	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			registry, dialer, broker := newRecordingReverseForwardBroker(t)
			ownerConnection, ownerSession := addBufferedTestSession(t)
			connection := ownerConnection
			if test.attacker {
				attackerConnection, _ := addBufferedTestSession(t)
				connection = attackerConnection
			}
			authorizationID := test.prepareID(t, registry, ownerSession.ID)
			tunnelID := uint64(0x2000 + index)

			sendReverseTunnelRequest(t, connection, broker, &sliverpb.TunnelData{
				TunnelID:      tunnelID,
				CreateReverse: true,
				Rportfwd: &sliverpb.RPortfwd{
					AuthorizationID: authorizationID.String(),
					Host:            "169.254.169.254",
					Port:            80,
				},
			})

			assertDialAddresses(t, dialer)
			if tunnel := rtunnels.GetRTunnel(tunnelID); tunnel != nil {
				closeTestReverseTunnel(tunnelID)
				t.Fatalf("rejected request created reverse tunnel %d", tunnelID)
			}
		})
	}
}

func TestLegacyReverseTunnelRequiresCanonicalOperatorDestination(t *testing.T) {
	t.Run("canonical match dials stored destination", func(t *testing.T) {
		const tunnelID = uint64(0x3001)
		registry, dialer, broker := newRecordingReverseForwardBroker(t)
		connection, session := addBufferedTestSession(t)
		beginActiveAuthorization(t, registry, session.ID, "Example.COM.:443", 4)

		sendReverseTunnelRequest(t, connection, broker, &sliverpb.TunnelData{
			TunnelID:      tunnelID,
			CreateReverse: true,
			Rportfwd: &sliverpb.RPortfwd{
				Host: "EXAMPLE.com",
				Port: 443,
			},
		})

		assertDialAddresses(t, dialer, "example.com:443")
		t.Cleanup(func() {
			closeTestReverseTunnel(tunnelID)
		})
	})

	t.Run("mismatch is rejected", func(t *testing.T) {
		const tunnelID = uint64(0x3002)
		registry, dialer, broker := newRecordingReverseForwardBroker(t)
		connection, session := addBufferedTestSession(t)
		beginActiveAuthorization(t, registry, session.ID, "127.0.0.1:4444", 5)

		sendReverseTunnelRequest(t, connection, broker, &sliverpb.TunnelData{
			TunnelID:      tunnelID,
			CreateReverse: true,
			Rportfwd: &sliverpb.RPortfwd{
				Host: "169.254.169.254",
				Port: 80,
			},
		})

		assertDialAddresses(t, dialer)
		if tunnel := rtunnels.GetRTunnel(tunnelID); tunnel != nil {
			closeTestReverseTunnel(tunnelID)
			t.Fatalf("legacy address mismatch created reverse tunnel %d", tunnelID)
		}
	})
}

func TestNegotiatedAuthorizationRejectsLegacyDowngrade(t *testing.T) {
	const tunnelID = uint64(0x3100)
	registry, dialer, broker := newRecordingReverseForwardBroker(t)
	connection, session := addBufferedTestSession(t)
	authorizationID, err := registry.Begin(session.ID, "127.0.0.1:4444", 0)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}
	if err := registry.ActivateProtocol(session.ID, authorizationID, 6, true); err != nil {
		t.Fatalf("ActivateProtocol() error = %v", err)
	}

	sendReverseTunnelRequest(t, connection, broker, &sliverpb.TunnelData{
		TunnelID:      tunnelID,
		CreateReverse: true,
		Rportfwd:      &sliverpb.RPortfwd{Host: "127.0.0.1", Port: 4444},
	})

	assertDialAddresses(t, dialer)
	if tunnel := rtunnels.GetRTunnel(tunnelID); tunnel != nil {
		closeTestReverseTunnel(tunnelID)
		t.Fatalf("capability-required listener accepted a legacy tunnel open")
	}
}

func TestTunnelWriterUnblocksWhenAuthorizationCloses(t *testing.T) {
	connection := core.NewImplantConnection("test", "n/a")
	tunnel := rtunnels.NewAuthorizedRTunnel(0x3200, "session", rtunnels.AuthorizationID("authorization"), &countingHandlerWriteCloser{})
	result := make(chan error, 1)
	go func() {
		_, err := (tunnelWriter{tun: tunnel, conn: connection}).Write([]byte("blocked"))
		result <- err
	}()

	tunnel.Close()
	select {
	case err := <-result:
		if !errors.Is(err, errReverseTunnelClosed) {
			t.Fatalf("tunnelWriter.Write() error = %v, want %v", err, errReverseTunnelClosed)
		}
	case <-time.After(time.Second):
		t.Fatal("closing authorization did not unblock the implant send")
	}
}

func TestReverseTunnelCreationRejectsMalformedMessages(t *testing.T) {
	connection, _ := addTestSession(t)

	t.Run("invalid protobuf", func(t *testing.T) {
		assertNoPanic(t, func() {
			tunnelDataHandler(connection, []byte{0xff})
		})
	})

	t.Run("nil tunnel data", func(t *testing.T) {
		assertNoPanic(t, func() {
			createReverseTunnelHandler(connection, nil)
		})
	})

	t.Run("nil reverse metadata", func(t *testing.T) {
		const tunnelID = uint64(0x8badf00d)
		data, err := proto.Marshal(&sliverpb.TunnelData{
			TunnelID:      tunnelID,
			CreateReverse: true,
		})
		if err != nil {
			t.Fatalf("marshal tunnel data: %v", err)
		}

		assertNoPanic(t, func() {
			tunnelDataHandler(connection, data)
		})
		if tunnel := rtunnels.GetRTunnel(tunnelID); tunnel != nil {
			closeTestReverseTunnel(tunnelID)
			t.Fatalf("malformed request created reverse tunnel %d", tunnelID)
		}
	})

	select {
	case envelope := <-connection.Send:
		t.Fatalf("malformed request produced outbound envelope type %d", envelope.Type)
	default:
	}
}

func TestHandlersUseReverseForwardBrokerBoundary(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read handlers directory: %v", err)
	}

	files := token.NewFileSet()
	forbiddenCalls := map[string]bool{
		"Dial":           true,
		"DialContext":    true,
		"DialTimeout":    true,
		"Acquire":        true,
		"AcquireLegacy":  true,
		"Lookup":         true,
		"LookupListener": true,
		"List":           true,
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		parsed, err := parser.ParseFile(files, filepath.Clean(name), nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			if selector, ok := node.(*ast.SelectorExpr); ok && selector.Sel.Name == "DefaultRegistry" {
				position := files.Position(selector.Pos())
				t.Errorf("direct DefaultRegistry access is forbidden in handlers at %s; use the reverse-forward broker", position)
				return true
			}

			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if !forbiddenCalls[selector.Sel.Name] {
				return true
			}

			position := files.Position(call.Pos())
			t.Errorf("direct %s call is forbidden in handlers at %s; use the reverse-forward broker", selector.Sel.Name, position)
			return true
		})
	}
}

type recordingReverseForwardDialer struct {
	mutex  sync.Mutex
	calls  []string
	peers  []net.Conn
	drains sync.WaitGroup
}

func (dialer *recordingReverseForwardDialer) DialContext(_ context.Context, network string, address string) (net.Conn, error) {
	dialer.mutex.Lock()
	defer dialer.mutex.Unlock()
	dialer.calls = append(dialer.calls, network+" "+address)
	connection, peer := net.Pipe()
	dialer.peers = append(dialer.peers, peer)
	dialer.drains.Add(1)
	go func() {
		defer dialer.drains.Done()
		const maxTestRelayBytes = 1 << 20
		_, _ = io.CopyN(io.Discard, peer, maxTestRelayBytes)
	}()
	return connection, nil
}

func (dialer *recordingReverseForwardDialer) addresses() []string {
	dialer.mutex.Lock()
	defer dialer.mutex.Unlock()
	addresses := make([]string, 0, len(dialer.calls))
	for _, call := range dialer.calls {
		_, address, _ := strings.Cut(call, " ")
		addresses = append(addresses, address)
	}
	return addresses
}

func (dialer *recordingReverseForwardDialer) close() {
	dialer.mutex.Lock()
	peers := dialer.peers
	dialer.peers = nil
	dialer.mutex.Unlock()
	for _, peer := range peers {
		_ = peer.Close()
	}
	dialer.drains.Wait()
}

func newRecordingReverseForwardBroker(t *testing.T) (*rtunnels.Registry, *recordingReverseForwardDialer, *rtunnels.Broker) {
	t.Helper()
	registry := rtunnels.NewRegistry()
	dialer := &recordingReverseForwardDialer{}
	broker := rtunnels.NewBroker(registry, dialer, time.Second)
	t.Cleanup(dialer.close)
	return registry, dialer, broker
}

func addBufferedTestSession(t *testing.T) (*core.ImplantConnection, *core.Session) {
	t.Helper()
	connection, session := addTestSession(t)
	connection.Send = make(chan *sliverpb.Envelope, 8)
	return connection, session
}

func beginActiveAuthorization(t *testing.T, registry *rtunnels.Registry, sessionID string, address string, listenerID uint32) rtunnels.AuthorizationID {
	t.Helper()
	authorizationID, err := registry.Begin(sessionID, address, 0)
	if err != nil {
		t.Fatalf("Begin(%q) error = %v", address, err)
	}
	if err := registry.Activate(sessionID, authorizationID, listenerID); err != nil {
		t.Fatalf("Activate() error = %v", err)
	}
	return authorizationID
}

func sendReverseTunnelRequest(t *testing.T, connection *core.ImplantConnection, broker *rtunnels.Broker, request *sliverpb.TunnelData) {
	t.Helper()
	assertNoPanic(t, func() {
		createReverseTunnelHandlerWithBroker(connection, request, broker)
	})
}

func assertDialAddresses(t *testing.T, dialer *recordingReverseForwardDialer, want ...string) {
	t.Helper()
	got := dialer.addresses()
	if len(got) != len(want) {
		t.Fatalf("dial addresses = %v, want %v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("dial addresses = %v, want %v", got, want)
		}
	}
}

func closeTestReverseTunnel(tunnelID uint64) {
	tunnel := rtunnels.GetRTunnel(tunnelID)
	if tunnel != nil && rtunnels.RemoveRTunnelIf(tunnelID, tunnel) {
		tunnel.Close()
	}
}

type countingHandlerWriteCloser struct{}

func (*countingHandlerWriteCloser) Write(buffer []byte) (int, error) {
	return len(buffer), nil
}

func (*countingHandlerWriteCloser) Close() error {
	return nil
}
