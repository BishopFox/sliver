package rportfwd

import (
	"testing"

	"github.com/bishopfox/sliver/implant/sliver/transports"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

func TestTunnelWriterIncludesAuthorizationIDOnCreateReverse(t *testing.T) {
	const (
		authorizationID = "server-issued-authorization"
		tunnelID        = uint64(42)
	)

	connection := &transports.Connection{Send: make(chan *sliverpb.Envelope, 2)}
	tunnel := transports.NewTunnel(tunnelID, nil)
	if !connection.AddTunnel(tunnel) {
		t.Fatal("failed to publish test tunnel")
	}
	t.Cleanup(func() { connection.CloseTunnelRemote(tunnel) })
	writer := tunnelWriter{
		tun:             tunnel,
		conn:            connection,
		host:            "implant-supplied.example",
		port:            8443,
		protocol:        sliverpb.PortFwdProtoTCP,
		tunnelID:        tunnelID,
		authorizationID: authorizationID,
	}

	if _, err := writer.Write([]byte("first")); err != nil {
		t.Fatalf("first Write() error = %v", err)
	}
	first := decodeTunnelData(t, <-connection.Send)
	if !first.CreateReverse {
		t.Fatal("first tunnel frame did not request reverse tunnel creation")
	}
	if first.Rportfwd == nil {
		t.Fatal("first tunnel frame has no reverse port forward metadata")
	}
	if got := first.Rportfwd.AuthorizationID; got != authorizationID {
		t.Fatalf("AuthorizationID = %q, want %q", got, authorizationID)
	}
	if first.Rportfwd.Host != "" || first.Rportfwd.Port != 0 { //nolint:staticcheck // Verify deprecated fields are absent from authorized traffic.
		t.Fatalf("authorized create frame included legacy destination %q:%d", first.Rportfwd.Host, first.Rportfwd.Port) //nolint:staticcheck // Test diagnostics for deprecated compatibility fields.
	}

	if _, err := writer.Write([]byte("second")); err != nil {
		t.Fatalf("second Write() error = %v", err)
	}
	second := decodeTunnelData(t, <-connection.Send)
	if second.CreateReverse {
		t.Fatal("subsequent tunnel frame unexpectedly requested reverse tunnel creation")
	}
}

func TestTunnelWriterIncludesLegacyAddressWithoutAuthorization(t *testing.T) {
	const tunnelID = uint64(43)
	connection := &transports.Connection{Send: make(chan *sliverpb.Envelope, 1)}
	tunnel := transports.NewTunnel(tunnelID, nil)
	if !connection.AddTunnel(tunnel) {
		t.Fatal("failed to publish test tunnel")
	}
	t.Cleanup(func() { connection.CloseTunnelRemote(tunnel) })
	writer := tunnelWriter{
		tun:      tunnel,
		conn:     connection,
		host:     "legacy.example",
		port:     9443,
		protocol: sliverpb.PortFwdProtoTCP,
		tunnelID: tunnelID,
	}

	if _, err := writer.Write([]byte("legacy")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	first := decodeTunnelData(t, <-connection.Send)
	if first.Rportfwd == nil {
		t.Fatal("first tunnel frame has no reverse port forward metadata")
	}
	if got := first.Rportfwd.AuthorizationID; got != "" {
		t.Fatalf("AuthorizationID = %q, want empty", got)
	}
	if first.Rportfwd.Host != "legacy.example" || first.Rportfwd.Port != 9443 { //nolint:staticcheck // Verify deprecated compatibility fields remain available.
		t.Fatalf("legacy destination = %q:%d, want legacy.example:9443", first.Rportfwd.Host, first.Rportfwd.Port) //nolint:staticcheck // Test diagnostics for deprecated compatibility fields.
	}
}

func decodeTunnelData(t *testing.T, envelope *sliverpb.Envelope) *sliverpb.TunnelData {
	t.Helper()
	if envelope.Type != sliverpb.MsgTunnelData {
		t.Fatalf("envelope type = %d, want %d", envelope.Type, sliverpb.MsgTunnelData)
	}
	tunnelData := &sliverpb.TunnelData{}
	if err := proto.Unmarshal(envelope.Data, tunnelData); err != nil {
		t.Fatalf("decode tunnel data: %v", err)
	}
	return tunnelData
}
