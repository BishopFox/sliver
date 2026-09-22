package e2e

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

func TestReverseForwardRoundTripPreservesBinaryPayload(t *testing.T) {
	server, err := startTCPEchoServer()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := server.close(); err != nil {
			t.Errorf("close echo server: %v", err)
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	payload := []byte("binary-echo\x00\x01\xfe\xff")
	if err := reverseForwardRoundTrip(ctx, server.address(), payload); err != nil {
		t.Fatal(err)
	}
}

func TestRequireReverseForwardDialRejection(t *testing.T) {
	port, err := unusedTCPPort()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := requireReverseForwardDialRejection(ctx, "127.0.0.1:"+strconv.Itoa(port)); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRportFwdListenerRequiresAuthoritativeMetadata(t *testing.T) {
	listener := &sliverpb.RportFwdListener{
		ID:              7,
		BindAddress:     "127.0.0.1:41000",
		ForwardAddress:  "127.0.0.1:42000",
		AuthorizationID: "authorization-token",
	}
	if err := validateRportFwdListener(listener, 7, listener.BindAddress, listener.ForwardAddress, listener.AuthorizationID); err != nil {
		t.Fatal(err)
	}

	listener.ForwardAddress = "127.0.0.1:43000"
	if err := validateRportFwdListener(listener, 7, "127.0.0.1:41000", "127.0.0.1:42000", "authorization-token"); err == nil {
		t.Fatal("expected mismatched forward metadata to fail validation")
	}
}
