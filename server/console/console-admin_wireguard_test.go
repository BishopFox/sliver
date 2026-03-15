package console

import (
	"context"
	"encoding/json"
	"net"
	"strings"
	"testing"

	clientassets "github.com/bishopfox/sliver/client/assets"
	clienttransport "github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/certs"
	servertransport "github.com/bishopfox/sliver/server/transport"
)

func TestNewOperatorConfigWithWireGuardConnectsToWrappedMultiplayer(t *testing.T) {
	certs.SetupCAs()
	certs.SetupWGKeys()
	clienttransport.SetMultiplayerConnectMode(clienttransport.MultiplayerConnectAuto)

	operatorName := uniqueKickOperatorName(t)
	t.Cleanup(func() {
		_ = removeOperator(operatorName)
		_ = revokeOperatorClientCertificate(operatorName)
		closeOperatorStreams(operatorName)
	})

	port := freeUDPPort(t)
	configJSON, err := NewOperatorConfig(operatorName, "127.0.0.1", uint16(port), []string{"all"}, true)
	if err != nil {
		t.Fatalf("generate wireguard operator config: %v", err)
	}

	config := &clientassets.ClientConfig{}
	if err := json.Unmarshal(configJSON, config); err != nil {
		t.Fatalf("parse operator config: %v", err)
	}
	if config.WG == nil {
		t.Fatal("expected wireguard config block to be present")
	}

	operators, err := operatorRecordsByName(operatorName)
	if err != nil {
		t.Fatalf("load operator row: %v", err)
	}
	if len(operators) != 1 {
		t.Fatalf("expected one operator row, got %d", len(operators))
	}
	if operators[0].WGPubKey != config.WG.ClientPubKey {
		t.Fatalf("expected stored wireguard public key %q, got %q", config.WG.ClientPubKey, operators[0].WGPubKey)
	}
	if operators[0].WGTunIP != config.WG.ClientIP {
		t.Fatalf("expected stored wireguard tunnel IP %q, got %q", config.WG.ClientIP, operators[0].WGTunIP)
	}

	grpcServer, ln, err := servertransport.StartWGWrappedMtlsClientListener("127.0.0.1", uint16(port))
	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skipf("wireguard listener bind not permitted in this environment: %v", err)
		}
		t.Fatalf("start wrapped multiplayer listener: %v", err)
	}
	defer grpcServer.Stop()
	defer ln.Close()

	rpcClient, conn, err := clienttransport.MTLSConnect(config)
	if err != nil {
		t.Fatalf("connect operator through wireguard wrapper: %v", err)
	}
	defer clienttransport.CloseGRPCConnection(conn)

	if _, err := rpcClient.GetVersion(context.Background(), &commonpb.Empty{}); err != nil {
		t.Fatalf("GetVersion over wrapped multiplayer failed: %v", err)
	}
}

func freeUDPPort(t *testing.T) int {
	t.Helper()

	ln, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Skipf("udp binds are not permitted in this environment: %v", err)
	}
	defer ln.Close()
	return ln.LocalAddr().(*net.UDPAddr).Port
}
