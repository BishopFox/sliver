package console

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	clientassets "github.com/bishopfox/sliver/client/assets"
	clienttransport "github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	servertransport "github.com/bishopfox/sliver/server/transport"
	"google.golang.org/grpc"
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

func TestNewOperatorConfigWithWireGuardConnectsToWrappedMultiplayerRepeatedly(t *testing.T) {
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

	grpcServer, ln, err := servertransport.StartWGWrappedMtlsClientListener("127.0.0.1", uint16(port))
	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skipf("wireguard listener bind not permitted in this environment: %v", err)
		}
		t.Fatalf("start wrapped multiplayer listener: %v", err)
	}
	defer grpcServer.Stop()
	defer ln.Close()

	for attempt := 1; attempt <= 5; attempt++ {
		rpcClient, conn, err := clienttransport.MTLSConnect(config)
		if err != nil {
			t.Fatalf("attempt %d: connect operator through wireguard wrapper: %v", attempt, err)
		}
		if _, err := rpcClient.GetVersion(context.Background(), &commonpb.Empty{}); err != nil {
			_ = clienttransport.CloseGRPCConnection(conn)
			t.Fatalf("attempt %d: GetVersion over wrapped multiplayer failed: %v", attempt, err)
		}
		if _, err := rpcClient.GetOperators(context.Background(), &commonpb.Empty{}); err != nil {
			_ = clienttransport.CloseGRPCConnection(conn)
			t.Fatalf("attempt %d: GetOperators over wrapped multiplayer failed: %v", attempt, err)
		}
		if err := clienttransport.CloseGRPCConnection(conn); err != nil {
			t.Fatalf("attempt %d: close wrapped multiplayer connection: %v", attempt, err)
		}
	}
}

func TestWrappedMultiplayerWireGuardListenerRestartsCleanly(t *testing.T) {
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

	for attempt := 1; attempt <= 3; attempt++ {
		grpcServer, ln, err := servertransport.StartWGWrappedMtlsClientListener("127.0.0.1", uint16(port))
		if err != nil {
			if strings.Contains(err.Error(), "operation not permitted") {
				t.Skipf("wireguard listener bind not permitted in this environment: %v", err)
			}
			t.Fatalf("attempt %d: start wrapped multiplayer listener: %v", attempt, err)
		}

		rpcClient, conn, err := clienttransport.MTLSConnect(config)
		if err != nil {
			grpcServer.Stop()
			_ = ln.Close()
			t.Fatalf("attempt %d: connect operator through restarted wireguard wrapper: %v", attempt, err)
		}
		if _, err := rpcClient.GetVersion(context.Background(), &commonpb.Empty{}); err != nil {
			_ = clienttransport.CloseGRPCConnection(conn)
			grpcServer.Stop()
			_ = ln.Close()
			t.Fatalf("attempt %d: GetVersion over restarted wrapped multiplayer failed: %v", attempt, err)
		}
		if err := clienttransport.CloseGRPCConnection(conn); err != nil {
			grpcServer.Stop()
			_ = ln.Close()
			t.Fatalf("attempt %d: close wrapped multiplayer connection: %v", attempt, err)
		}

		grpcServer.Stop()
		if err := ln.Close(); err != nil {
			t.Fatalf("attempt %d: close wrapped multiplayer listener: %v", attempt, err)
		}
	}
}

func TestWrappedMultiplayerWireGuardSupportsUnaryRPCsWithDedicatedCommandConnection(t *testing.T) {
	t.Run("events-only", func(t *testing.T) {
		runWrappedMultiplayerWireGuardUnaryWithBackgroundStreams(t, true, false)
	})
	t.Run("tunnel-only", func(t *testing.T) {
		runWrappedMultiplayerWireGuardUnaryWithBackgroundStreams(t, false, true)
	})
	t.Run("events-and-tunnel", func(t *testing.T) {
		runWrappedMultiplayerWireGuardUnaryWithBackgroundStreams(t, true, true)
	})
}

func runWrappedMultiplayerWireGuardUnaryWithBackgroundStreams(t *testing.T, useEvents bool, useTunnel bool) {
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

	grpcServer, ln, err := servertransport.StartWGWrappedMtlsClientListener("127.0.0.1", uint16(port))
	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skipf("wireguard listener bind not permitted in this environment: %v", err)
		}
		t.Fatalf("start wrapped multiplayer listener: %v", err)
	}
	defer grpcServer.Stop()
	defer ln.Close()

	var streamClient rpcpb.SliverRPCClient
	var streamConn *grpc.ClientConn
	if useEvents || useTunnel {
		streamClient, streamConn, err = clienttransport.MTLSConnect(config)
		if err != nil {
			t.Fatalf("connect dedicated background operator through wireguard wrapper: %v", err)
		}
		defer clienttransport.CloseGRPCConnection(streamConn)
	}

	streamCtx, streamCancel := context.WithCancel(context.Background())
	defer streamCancel()

	if useEvents {
		events, err := streamClient.Events(streamCtx, &commonpb.Empty{})
		if err != nil {
			t.Fatalf("open events stream: %v", err)
		}
		go drainWGTestStream(t, "events", func() error {
			_, err := events.Recv()
			return err
		})
	}

	if useTunnel {
		tunnelData, err := streamClient.TunnelData(streamCtx)
		if err != nil {
			t.Fatalf("open tunnel data stream: %v", err)
		}
		go drainWGTestStream(t, "tunnel-data", func() error {
			_, err := tunnelData.Recv()
			return err
		})
	}

	// Let the background streams settle before issuing the unary call. The
	// interactive console opens these streams first and only later runs
	// unary RPC-backed commands like "operators".
	if useEvents || useTunnel {
		time.Sleep(2 * time.Second)
	}

	rpcClient, conn, err := clienttransport.MTLSConnect(config)
	if err != nil {
		t.Fatalf("connect operator command channel through wireguard wrapper: %v", err)
	}
	defer clienttransport.CloseGRPCConnection(conn)

	callCtx, callCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer callCancel()
	if _, err := rpcClient.GetOperators(callCtx, &commonpb.Empty{}); err != nil {
		t.Fatalf("GetOperators with dedicated command connection over wrapped multiplayer failed (events=%t tunnel=%t): %v", useEvents, useTunnel, err)
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

func drainWGTestStream(t *testing.T, name string, recv func() error) {
	t.Helper()

	err := recv()
	if err == nil || errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
		return
	}
	if strings.Contains(err.Error(), "context canceled") {
		return
	}
	t.Errorf("%s stream recv failed: %v", name, err)
}
