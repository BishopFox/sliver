package console

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	clientassets "github.com/bishopfox/sliver/client/assets"
	clienttransport "github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	servertransport "github.com/bishopfox/sliver/server/transport"
	"gorm.io/gorm"
)

func TestNewOperatorConfigWithWireGuardConnectsToWrappedMultiplayer(t *testing.T) {
	certs.SetupCAs()
	certs.SetupWGKeys()
	certs.SetupMultiplayerWGKeys()
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
	if config.WG.ServerPubKey == "" {
		t.Fatal("expected operator config to include wireguard server public key")
	}
	if config.WG.ClientPrivateKey == "" {
		t.Fatal("expected operator config to include wireguard client private key")
	}
	if config.WG.ClientPubKey == "" {
		t.Fatal("expected operator config to include wireguard client public key")
	}
	if config.WG.ClientIP == "" {
		t.Fatal("expected operator config to include wireguard client tunnel IP")
	}
	if config.WG.ServerIP != certs.MultiplayerWireGuardServerIP {
		t.Fatalf("expected wireguard server IP %q, got %q", certs.MultiplayerWireGuardServerIP, config.WG.ServerIP)
	}
	if !db.IsMultiplayerWireGuardIP(config.WG.ClientIP) {
		t.Fatalf("expected operator client IP %q to be in the multiplayer WireGuard network", config.WG.ClientIP)
	}
	if _, _, err := certs.GenerateWGKeys(true, config.WG.ClientIP); err == nil || !strings.Contains(err.Error(), db.C2WireGuardIPCIDR) {
		t.Fatalf("expected C2 WireGuard peer generation to reject multiplayer IP %q, got %v", config.WG.ClientIP, err)
	}

	_, c2ServerPubKey, err := certs.GetWGServerKeys()
	if err != nil {
		t.Fatalf("load c2 wireguard server keys: %v", err)
	}
	_, multiplayerServerPubKey, err := certs.GetMultiplayerWGServerKeys()
	if err != nil {
		t.Fatalf("load multiplayer wireguard server keys: %v", err)
	}
	if c2ServerPubKey == multiplayerServerPubKey {
		t.Fatal("expected multiplayer WireGuard server keypair to be distinct from the C2 WireGuard server keypair")
	}
	if config.WG.ServerPubKey != multiplayerServerPubKey {
		t.Fatalf("expected operator config to use multiplayer wireguard server public key %q, got %q", multiplayerServerPubKey, config.WG.ServerPubKey)
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

func TestOperatorCLIWireGuardConfigConnectsToWrappedMultiplayer(t *testing.T) {
	certs.SetupCAs()
	certs.SetupWGKeys()
	certs.SetupMultiplayerWGKeys()
	clienttransport.SetMultiplayerConnectMode(clienttransport.MultiplayerConnectAuto)

	port := freeUDPPort(t)
	grpcServer, ln, err := servertransport.StartWGWrappedMtlsClientListener("127.0.0.1", uint16(port))
	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skipf("wireguard listener bind not permitted in this environment: %v", err)
		}
		t.Fatalf("start wrapped multiplayer listener: %v", err)
	}
	defer grpcServer.Stop()
	defer ln.Close()

	operatorName := uniqueKickOperatorName(t)
	t.Cleanup(func() {
		_ = removeOperator(operatorName)
		_ = revokeOperatorClientCertificate(operatorName)
		closeOperatorStreams(operatorName)
	})

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
	savePath := filepath.Join(t.TempDir(), operatorName+".cfg")

	cmd := exec.Command(
		"go", "run", "-tags=server,go_sqlite", "./server", "operator",
		"--name", operatorName,
		"--lhost", "127.0.0.1",
		"--lport", strconv.Itoa(port),
		"--permissions", "all",
		"--save", savePath,
	)
	cmd.Dir = repoRoot
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generate operator config through CLI: %v\n%s", err, output)
	}

	configJSON, err := os.ReadFile(savePath)
	if err != nil {
		t.Fatalf("read generated operator config: %v", err)
	}
	config := &clientassets.ClientConfig{}
	if err := json.Unmarshal(configJSON, config); err != nil {
		t.Fatalf("parse operator config: %v", err)
	}
	if config.WG == nil {
		t.Fatal("expected wireguard config block to be present")
	}

	rpcClient, conn, err := clienttransport.MTLSConnect(config)
	if err != nil {
		t.Fatalf("connect operator through wireguard wrapper after CLI generation: %v", err)
	}
	defer clienttransport.CloseGRPCConnection(conn)

	if _, err := rpcClient.GetVersion(context.Background(), &commonpb.Empty{}); err != nil {
		t.Fatalf("GetVersion over wrapped multiplayer failed: %v", err)
	}
}

func TestKickOperatorReleasesWireGuardTunnelIPReservation(t *testing.T) {
	certs.SetupCAs()
	certs.SetupMultiplayerWGKeys()

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
	if config.WG == nil || config.WG.ClientIP == "" {
		t.Fatal("expected operator config to include wireguard client IP")
	}

	if err := db.ReserveWGIP(config.WG.ClientIP, models.WGIPOwnerTypeOperator, operatorName+"-duplicate"); !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("expected duplicate tunnel IP error while operator exists, got %v", err)
	}

	if err := kickOperator(operatorName); err != nil {
		t.Fatalf("kick operator: %v", err)
	}

	if err := db.ReserveWGIP(config.WG.ClientIP, models.WGIPOwnerTypeOperator, operatorName+"-after-kick"); err != nil {
		t.Fatalf("expected released tunnel IP to be reusable after kick, got %v", err)
	}
	t.Cleanup(func() {
		_ = db.ReleaseWGIP(config.WG.ClientIP)
	})
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
