package e2e

import (
	"bytes"
	"context"
	"testing"
	"time"
)

func TestPortForwardPayloadCorpusIsDeterministic(t *testing.T) {
	first := portForwardPayloadCases()
	second := portForwardPayloadCases()
	if len(first) != len(second) || len(first) == 0 {
		t.Fatalf("payload corpus lengths = %d and %d", len(first), len(second))
	}
	for index := range first {
		if first[index].name != second[index].name {
			t.Fatalf("case %d name = %q, want %q", index, first[index].name, second[index].name)
		}
		if !bytes.Equal(first[index].payload, second[index].payload) {
			t.Fatalf("case %d payload is not reproducible", index)
		}
		if len(first[index].payload) == 0 || len(first[index].chunks) == 0 {
			t.Fatalf("case %d is empty: %+v", index, first[index])
		}
		for _, chunk := range first[index].chunks {
			if chunk <= 0 {
				t.Fatalf("case %d contains invalid chunk size %d", index, chunk)
			}
		}
	}
}

func TestTunnelFixturesCarryBinaryAndHTTPDirectly(t *testing.T) {
	echoServer, err := startTCPEchoServer()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := echoServer.close(); err != nil {
			t.Errorf("close echo fixture: %v", err)
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	payload := deterministicTunnelPayload("direct-fixture-binary", 4108)
	if err := tunnelEchoRoundTrip(ctx, echoServer.address(), payload, []int{1, 4095, 12}); err != nil {
		cancel()
		t.Fatal(err)
	}
	cancel()

	httpFixture, err := startDeterministicHTTPServer("direct-fixture-http")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := httpFixture.close(); err != nil {
			t.Errorf("close HTTP fixture: %v", err)
		}
	})
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := requestDeterministicHTTP(ctx, httpFixture.address(), httpFixture); err != nil {
		t.Fatal(err)
	}
}
