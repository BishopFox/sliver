package e2e

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

func TestTunnelFullDuplexEchoOnConn(t *testing.T) {
	echoServer, err := startTCPEchoServer()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := echoServer.close(); err != nil {
			t.Errorf("close echo server: %v", err)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	connection, err := (&net.Dialer{}).DialContext(ctx, "tcp4", echoServer.address())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = connection.Close() }()
	if _, err := tunnelFullDuplexEchoOnConn(ctx, connection, "unit-full-duplex", 2*1024*1024, 1); err != nil {
		t.Fatal(err)
	}
}

func TestTunnelFullDuplexRejectsInvalidBounds(t *testing.T) {
	left, right := net.Pipe()
	defer func() { _ = left.Close() }()
	defer func() { _ = right.Close() }()
	if _, err := tunnelFullDuplexEchoOnConn(context.Background(), left, "invalid", 0, 1); err == nil {
		t.Fatal("zero payload was accepted")
	}
	if _, err := tunnelFullDuplexEchoOnConn(context.Background(), left, "invalid", 1, 0); err == nil {
		t.Fatal("zero throughput floor was accepted")
	}
	if err := writeTunnelChunks(left, []byte("x"), []int{0}); err == nil || errors.Is(err, net.ErrClosed) {
		t.Fatalf("invalid chunk error = %v", err)
	}
}
