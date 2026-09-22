package rpc

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type portfwdRPCTestStream struct {
	rpcpb.SliverRPC_TunnelDataServer
}

func TestValidatePortfwdTarget(t *testing.T) {
	valid := []*sliverpb.PortfwdReq{
		{Host: "target.example", Port: 443, Protocol: sliverpb.PortFwdProtoTCP},
		{Host: "192.0.2.10", Port: 3389, Protocol: sliverpb.PortFwdProtoTCP},
		{Host: "2001:db8::10", Port: 8443, Protocol: sliverpb.PortFwdProtoTCP},
	}
	for _, request := range valid {
		if err := validatePortfwdTarget(request); err != nil {
			t.Errorf("validatePortfwdTarget(%+v): %v", request, err)
		}
	}

	invalid := []struct {
		name       string
		request    *sliverpb.PortfwdReq
		wantDetail string
	}{
		{name: "nil request", wantDetail: "request is required"},
		{name: "empty host", request: &sliverpb.PortfwdReq{Port: 80, Protocol: sliverpb.PortFwdProtoTCP}, wantDetail: "host"},
		{name: "whitespace host", request: &sliverpb.PortfwdReq{Host: "target example", Port: 80, Protocol: sliverpb.PortFwdProtoTCP}, wantDetail: "host"},
		{name: "control host", request: &sliverpb.PortfwdReq{Host: "target.example\n", Port: 80, Protocol: sliverpb.PortFwdProtoTCP}, wantDetail: "host"},
		{name: "zero port", request: &sliverpb.PortfwdReq{Host: "target.example", Protocol: sliverpb.PortFwdProtoTCP}, wantDetail: "port"},
		{name: "out of range port", request: &sliverpb.PortfwdReq{Host: "target.example", Port: 65536, Protocol: sliverpb.PortFwdProtoTCP}, wantDetail: "port"},
		{name: "missing protocol", request: &sliverpb.PortfwdReq{Host: "target.example", Port: 80}, wantDetail: "protocol"},
		{name: "UDP", request: &sliverpb.PortfwdReq{Host: "target.example", Port: 53, Protocol: sliverpb.PortFwdProtoUDP}, wantDetail: "protocol"},
	}
	for _, test := range invalid {
		t.Run(test.name, func(t *testing.T) {
			err := validatePortfwdTarget(test.request)
			if err == nil || !strings.Contains(err.Error(), test.wantDetail) {
				t.Fatalf("validatePortfwdTarget(%+v) error = %v, want detail %q", test.request, err, test.wantDetail)
			}
		})
	}
}

func TestPortfwdRejectsInvalidTargetBeforeTunnelBind(t *testing.T) {
	session := newPortfwdRPCTestSession(t)
	tunnel := newPortfwdRPCTestTunnel(t, session.ID)
	tests := []struct {
		name     string
		host     string
		port     uint32
		protocol int32
	}{
		{name: "empty host", port: 80, protocol: sliverpb.PortFwdProtoTCP},
		{name: "zero port", host: "target.example", protocol: sliverpb.PortFwdProtoTCP},
		{name: "out of range port", host: "target.example", port: 65536, protocol: sliverpb.PortFwdProtoTCP},
		{name: "unsupported protocol", host: "target.example", port: 53, protocol: sliverpb.PortFwdProtoUDP},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
			defer cancel()
			_, err := (&Server{}).Portfwd(ctx, &sliverpb.PortfwdReq{
				Request:  &commonpb.Request{SessionID: session.ID},
				Host:     test.host,
				Port:     test.port,
				Protocol: test.protocol,
				TunnelID: tunnel.ID,
			})
			if status.Code(err) != codes.InvalidArgument {
				t.Fatalf("Portfwd invalid target error = %v, want %s", err, codes.InvalidArgument)
			}
		})
	}

	if got := core.Tunnels.Get(tunnel.ID); got != tunnel {
		t.Fatalf("invalid target changed tunnel registration: got=%p want=%p", got, tunnel)
	}
	select {
	case envelope := <-session.Connection.Send:
		t.Fatalf("invalid target reached implant connection: %+v", envelope)
	default:
	}
}

func TestPortfwdRejectsTunnelOwnedByAnotherSession(t *testing.T) {
	owner := newPortfwdRPCTestSession(t)
	other := newPortfwdRPCTestSession(t)
	tunnel := newPortfwdRPCTestTunnel(t, owner.ID)

	_, err := (&Server{}).Portfwd(context.Background(), &sliverpb.PortfwdReq{
		Request:  &commonpb.Request{SessionID: other.ID},
		Host:     "127.0.0.1",
		Port:     8080,
		Protocol: sliverpb.PortFwdProtoTCP,
		TunnelID: tunnel.ID,
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("Portfwd ownership error = %v, want %s", err, codes.InvalidArgument)
	}
}

func TestPortfwdRejectsReplacementSessionGeneration(t *testing.T) {
	creatingSession := newPortfwdRPCTestSession(t)
	tunnel := newPortfwdRPCTestTunnel(t, creatingSession.ID)
	stream := &portfwdRPCTestStream{}
	if !tunnel.BindClient(stream) || !tunnel.MarkClientBound(stream) {
		t.Fatal("failed to bind test tunnel")
	}
	replacement := replaceTunnelStreamTestSession(t, creatingSession)

	_, err := (&Server{}).Portfwd(context.Background(), &sliverpb.PortfwdReq{
		Request:  &commonpb.Request{SessionID: replacement.ID},
		Host:     "127.0.0.1",
		Port:     8080,
		Protocol: sliverpb.PortFwdProtoTCP,
		TunnelID: tunnel.ID,
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("Portfwd replacement-generation error = %v, want %s", err, codes.InvalidArgument)
	}
	select {
	case envelope := <-creatingSession.Connection.Send:
		t.Fatalf("rejected port-forward setup reached creating connection: %+v", envelope)
	default:
	}
	assertReplacementConnectionUntouched(t, replacement)
}

func TestPortfwdWaitsForClientTunnelBind(t *testing.T) {
	session := newPortfwdRPCTestSession(t)
	tunnel := newPortfwdRPCTestTunnel(t, session.ID)
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, err := (&Server{}).Portfwd(ctx, &sliverpb.PortfwdReq{
			Request:  &commonpb.Request{SessionID: session.ID},
			Host:     "127.0.0.1",
			Port:     8080,
			Protocol: sliverpb.PortFwdProtoTCP,
			TunnelID: tunnel.ID,
		})
		result <- err
	}()

	select {
	case err := <-result:
		t.Fatalf("Portfwd returned before the tunnel was client-bound: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Portfwd canceled wait error = %v, want %v", err, context.Canceled)
		}
	case <-time.After(time.Second):
		t.Fatal("Portfwd did not stop waiting after context cancellation")
	}
}

func TestPortfwdBoundWaitRejectsClosedTunnel(t *testing.T) {
	session := newPortfwdRPCTestSession(t)
	tunnel := newPortfwdRPCTestTunnel(t, session.ID)
	result := make(chan error, 1)
	go func() {
		_, err := (&Server{}).Portfwd(context.Background(), &sliverpb.PortfwdReq{
			Request:  &commonpb.Request{SessionID: session.ID},
			Host:     "127.0.0.1",
			Port:     8080,
			Protocol: sliverpb.PortFwdProtoTCP,
			TunnelID: tunnel.ID,
		})
		result <- err
	}()

	select {
	case err := <-result:
		t.Fatalf("Portfwd returned before tunnel close: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	core.Tunnels.CloseIf(tunnel)
	select {
	case err := <-result:
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("Portfwd closed-tunnel error = %v, want %s", err, codes.InvalidArgument)
		}
	case <-time.After(time.Second):
		t.Fatal("Portfwd did not stop waiting after tunnel close")
	}
}

func TestPortfwdImplantRequestIsCanceledByCallerOrTunnel(t *testing.T) {
	tests := []struct {
		name         string
		cancelCaller bool
		closeTunnel  bool
	}{
		{name: "caller cancellation", cancelCaller: true},
		{name: "tunnel close", closeTunnel: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			session := newPortfwdRPCTestSession(t)
			tunnel := newPortfwdRPCTestTunnel(t, session.ID)
			stream := &portfwdRPCTestStream{}
			if !tunnel.BindClient(stream) || !tunnel.MarkClientBound(stream) {
				t.Fatal("failed to bind test tunnel")
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			result := make(chan error, 1)
			go func() {
				_, err := (&Server{}).Portfwd(ctx, &sliverpb.PortfwdReq{
					Request:  &commonpb.Request{SessionID: session.ID},
					Host:     "127.0.0.1",
					Port:     3389,
					Protocol: sliverpb.PortFwdProtoTCP,
					TunnelID: tunnel.ID,
				})
				result <- err
			}()

			select {
			case <-session.Connection.Send:
			case <-time.After(time.Second):
				t.Fatal("port-forward request was not sent to the implant")
			}
			if test.cancelCaller {
				cancel()
			}
			if test.closeTunnel {
				core.Tunnels.CloseIf(tunnel)
			}
			select {
			case err := <-result:
				if status.Code(err) != codes.Canceled {
					t.Fatalf("Portfwd cancellation error = %v, want %s", err, codes.Canceled)
				}
			case <-time.After(time.Second):
				t.Fatal("Portfwd request remained blocked after cancellation")
			}
			session.Connection.RespMutex.RLock()
			waiters := len(session.Connection.Resp)
			session.Connection.RespMutex.RUnlock()
			if waiters != 0 {
				t.Fatalf("response waiter count = %d, want 0", waiters)
			}
		})
	}
}

func newPortfwdRPCTestSession(t *testing.T) *core.Session {
	t.Helper()
	session := core.NewSession(core.NewImplantConnection("mtls", "portfwd-rpc-test"))
	core.Sessions.Add(session)
	t.Cleanup(func() { core.Sessions.Remove(session.ID) })
	return session
}

func newPortfwdRPCTestTunnel(t *testing.T, sessionID string) *core.Tunnel {
	t.Helper()
	tunnel, err := core.Tunnels.Create(sessionID)
	if err != nil {
		t.Fatalf("create test tunnel: %v", err)
	}
	t.Cleanup(func() { core.Tunnels.CloseIf(tunnel) })
	return tunnel
}
