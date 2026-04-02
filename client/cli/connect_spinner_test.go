package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"google.golang.org/grpc"
)

func TestFormatConnectSpinnerMessage(t *testing.T) {
	tests := []struct {
		name   string
		target string
		status string
		want   string
	}{
		{
			name:   "target only",
			target: "127.0.0.1:31337",
			want:   "Connecting to 127.0.0.1:31337 ...",
		},
		{
			name:   "target with status",
			target: "127.0.0.1:31337",
			status: "wireguard",
			want:   "Connecting to 127.0.0.1:31337 (wireguard) ...",
		},
		{
			name:   "status without target",
			status: "grpc/mtls",
			want:   "Connecting (grpc/mtls) ...",
		},
		{
			name:   "trim whitespace",
			target: " 127.0.0.1:31337 ",
			status: " grpc/mtls over wireguard ",
			want:   "Connecting to 127.0.0.1:31337 (grpc/mtls over wireguard) ...",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := formatConnectSpinnerMessage(test.target, test.status); got != test.want {
				t.Fatalf("expected %q, got %q", test.want, got)
			}
		})
	}
}

func TestConnectWithSpinnerOutput(t *testing.T) {
	var out bytes.Buffer
	spinnerDelay := 2 * (100 * time.Millisecond)

	_, _, err := connectWithSpinner(&out, "127.0.0.1:31337", func(statusFn transport.ConnectStatusFn) (rpcpb.SliverRPCClient, *grpc.ClientConn, error) {
		statusFn("wireguard")
		time.Sleep(spinnerDelay)
		statusFn("grpc/mtls over wireguard")
		time.Sleep(spinnerDelay)
		return nil, nil, nil
	})
	if err != nil {
		t.Fatalf("connectWithSpinner returned error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Connecting to 127.0.0.1:31337 (wireguard) ...") {
		t.Fatalf("expected wireguard status in output, got %q", got)
	}
	if !strings.Contains(got, "Connecting to 127.0.0.1:31337 (grpc/mtls over wireguard) ...") {
		t.Fatalf("expected grpc over wireguard status in output, got %q", got)
	}
	frameCount := 0
	for _, frame := range []string{"|", "/", "-", "\\"} {
		if strings.Contains(got, frame+" Connecting to 127.0.0.1:31337") {
			frameCount++
		}
	}
	if frameCount < 2 {
		t.Fatalf("expected multiple spinner frames in output, got %q", got)
	}
}

func TestConnectWithSpinnerFastSuccessStillShowsMultipleFrames(t *testing.T) {
	var out bytes.Buffer

	_, _, err := connectWithSpinner(&out, "127.0.0.1:31337", func(statusFn transport.ConnectStatusFn) (rpcpb.SliverRPCClient, *grpc.ClientConn, error) {
		statusFn("grpc/mtls")
		return nil, nil, nil
	})
	if err != nil {
		t.Fatalf("connectWithSpinner returned error: %v", err)
	}

	got := out.String()
	frameCount := 0
	for _, frame := range []string{"|", "/", "-", "\\"} {
		if strings.Contains(got, frame+" Connecting to 127.0.0.1:31337 (grpc/mtls) ...") {
			frameCount++
		}
	}
	if frameCount < 2 {
		t.Fatalf("expected multiple spinner frames for fast success, got %q", got)
	}
}

func TestConnectWithSpinnerFastSuccessShowsEachStatus(t *testing.T) {
	var out bytes.Buffer

	_, _, err := connectWithSpinner(&out, "127.0.0.1:31337", func(statusFn transport.ConnectStatusFn) (rpcpb.SliverRPCClient, *grpc.ClientConn, error) {
		statusFn("wireguard")
		statusFn("grpc/mtls over wireguard")
		return nil, nil, nil
	})
	if err != nil {
		t.Fatalf("connectWithSpinner returned error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Connecting to 127.0.0.1:31337 (wireguard) ...") {
		t.Fatalf("expected fast success output to include wireguard status, got %q", got)
	}
	if !strings.Contains(got, "Connecting to 127.0.0.1:31337 (grpc/mtls over wireguard) ...") {
		t.Fatalf("expected fast success output to include grpc over wireguard status, got %q", got)
	}
	if strings.Contains(got, "Connected to 127.0.0.1:31337") {
		t.Fatalf("did not expect past-tense success output, got %q", got)
	}
}
