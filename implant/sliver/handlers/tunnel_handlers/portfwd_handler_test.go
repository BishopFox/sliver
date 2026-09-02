package tunnel_handlers

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/bishopfox/sliver/implant/sliver/transports"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

type portfwdDialFunc func(context.Context, string, string) (net.Conn, error)

func (dial portfwdDialFunc) DialContext(ctx context.Context, network string, address string) (net.Conn, error) {
	return dial(ctx, network, address)
}

func TestPortfwdRemoteAddress(t *testing.T) {
	tests := []struct {
		name string
		host string
		port uint32
		want string
	}{
		{name: "hostname", host: "target.example", port: 443, want: "target.example:443"},
		{name: "IPv4", host: "192.0.2.10", port: 8080, want: "192.0.2.10:8080"},
		{name: "IPv6", host: "2001:db8::10", port: 8443, want: "[2001:db8::10]:8443"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := portfwdRemoteAddress(test.host, test.port); got != test.want {
				t.Fatalf("portfwdRemoteAddress(%q, %d) = %q, want %q", test.host, test.port, got, test.want)
			}
		})
	}
}

func TestPortfwdCloseCancelsBlockedDial(t *testing.T) {
	const tunnelID = uint64(0x7011)
	connection := &transports.Connection{Send: make(chan *sliverpb.Envelope, 2)}
	t.Cleanup(connection.Cleanup)
	started := make(chan struct{})
	dialer := portfwdDialFunc(func(ctx context.Context, _, _ string) (net.Conn, error) {
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	})
	done := runPortfwdHandler(t, connection, tunnelID, dialer, time.Hour)
	waitPortfwdTestSignal(t, started, "destination dial")

	handleTunnelClose(&sliverpb.TunnelData{TunnelID: tunnelID, Closed: true}, connection, time.Second)
	waitPortfwdTestSignal(t, done, "canceled port-forward handler")
	if active := connection.Tunnel(tunnelID); active != nil {
		t.Fatalf("canceled destination dial published tunnel %p", active)
	}
	if replacement, result := connection.BeginTunnel(tunnelID, time.Hour); result != transports.TunnelAddDuplicate || replacement != nil {
		t.Fatalf("closed setup ID replacement = %p, %v; want nil, %v", replacement, result, transports.TunnelAddDuplicate)
	}
}

func TestPortfwdCloseOvertakesLateDialAndClosesSocket(t *testing.T) {
	const tunnelID = uint64(0x7012)
	connection := &transports.Connection{Send: make(chan *sliverpb.Envelope, 2)}
	t.Cleanup(connection.Cleanup)
	implantSide, targetSide := net.Pipe()
	t.Cleanup(func() { _ = targetSide.Close() })
	started := make(chan struct{})
	canceled := make(chan struct{})
	release := make(chan struct{})
	dialer := portfwdDialFunc(func(ctx context.Context, _, _ string) (net.Conn, error) {
		close(started)
		go func() {
			<-ctx.Done()
			close(canceled)
		}()
		<-release // Simulate a dialer that returns a socket after cancellation.
		return implantSide, nil
	})
	done := runPortfwdHandler(t, connection, tunnelID, dialer, time.Hour)
	waitPortfwdTestSignal(t, started, "late destination dial")
	handleTunnelClose(&sliverpb.TunnelData{TunnelID: tunnelID, Closed: true}, connection, time.Second)
	waitPortfwdTestSignal(t, canceled, "pending dial cancellation")
	if err := targetSide.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set target-side read deadline before late socket closes: %v", err)
	}
	close(release)
	waitPortfwdTestSignal(t, done, "late port-forward handler")

	if active := connection.Tunnel(tunnelID); active != nil {
		t.Fatalf("late destination dial published tunnel %p", active)
	}
	buffer := make([]byte, 1)
	if _, err := targetSide.Read(buffer); !errors.Is(err, io.EOF) {
		t.Fatalf("late destination socket read error = %v, want %v", err, io.EOF)
	}
}

func TestPortfwdOwnerDisconnectCancelsBlockedDial(t *testing.T) {
	const tunnelID = uint64(0x7013)
	connection := &transports.Connection{Send: make(chan *sliverpb.Envelope, 2)}
	started := make(chan struct{})
	dialer := portfwdDialFunc(func(ctx context.Context, _, _ string) (net.Conn, error) {
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	})
	done := runPortfwdHandler(t, connection, tunnelID, dialer, time.Hour)
	waitPortfwdTestSignal(t, started, "owner-scoped destination dial")
	connection.Cleanup()
	waitPortfwdTestSignal(t, done, "owner-disconnected port-forward handler")
	if active := connection.Tunnel(tunnelID); active != nil {
		t.Fatalf("owner-disconnected destination dial published tunnel %p", active)
	}
}

func TestPortfwdDestinationDialHasFiniteDeadline(t *testing.T) {
	const tunnelID = uint64(0x7014)
	connection := &transports.Connection{Send: make(chan *sliverpb.Envelope, 2)}
	t.Cleanup(connection.Cleanup)
	started := make(chan struct{})
	dialer := portfwdDialFunc(func(ctx context.Context, _, _ string) (net.Conn, error) {
		close(started)
		if _, ok := ctx.Deadline(); !ok {
			return nil, errors.New("destination dial context has no deadline")
		}
		<-ctx.Done()
		return nil, ctx.Err()
	})
	done := runPortfwdHandler(t, connection, tunnelID, dialer, 20*time.Millisecond)
	waitPortfwdTestSignal(t, started, "deadline-bound destination dial")
	waitPortfwdTestSignal(t, done, "deadline-bound port-forward handler")
	if active := connection.Tunnel(tunnelID); active != nil {
		t.Fatalf("timed-out destination dial published tunnel %p", active)
	}
}

func runPortfwdHandler(t *testing.T, connection *transports.Connection, tunnelID uint64, dialer portfwdContextDialer, timeout time.Duration) <-chan struct{} {
	t.Helper()
	data, err := proto.Marshal(&sliverpb.PortfwdReq{
		Request:  &commonpb.Request{SessionID: "portfwd-handler-test"},
		Host:     "127.0.0.1",
		Port:     3389,
		Protocol: sliverpb.PortFwdProtoTCP,
		TunnelID: tunnelID,
	})
	if err != nil {
		t.Fatalf("marshal port-forward request: %v", err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		handlePortfwdReq(&sliverpb.Envelope{ID: int64(tunnelID), Data: data}, connection, dialer, timeout)
	}()
	return done
}

func waitPortfwdTestSignal(t *testing.T, signal <-chan struct{}, description string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", description)
	}
}
