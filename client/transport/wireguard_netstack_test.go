package transport

import (
	"net/netip"
	"testing"
)

func TestRandomTransportTCPBindPortUsesEphemeralRange(t *testing.T) {
	for attempt := 0; attempt < 16; attempt++ {
		port, err := randomTransportTCPBindPort()
		if err != nil {
			t.Fatalf("choose random tcp bind port: %v", err)
		}
		if port < transportTCPBindPortMin {
			t.Fatalf("expected bind port >= %d, got %d", transportTCPBindPortMin, port)
		}
		if 65535 < port {
			t.Fatalf("expected bind port <= 65535, got %d", port)
		}
	}
}

func TestRandomTCPBindAddrUsesPrimaryTransportAddress(t *testing.T) {
	netstack := &transportNet{
		primaryAddr: netip.MustParseAddr("100.65.0.2"),
	}

	localAddr, err := netstack.randomTCPBindAddr()
	if err != nil {
		t.Fatalf("random tcp bind addr: %v", err)
	}
	if localAddr.NIC != 1 {
		t.Fatalf("expected local bind NIC 1, got %d", localAddr.NIC)
	}
	got, ok := netip.AddrFromSlice(localAddr.Addr.AsSlice())
	if !ok {
		t.Fatal("expected local bind address to be parseable")
	}
	if got != netstack.primaryAddr {
		t.Fatalf("expected local bind IP %s, got %s", netstack.primaryAddr, got)
	}
	if localAddr.Port < transportTCPBindPortMin {
		t.Fatalf("expected local bind port >= %d, got %d", transportTCPBindPortMin, localAddr.Port)
	}
}
