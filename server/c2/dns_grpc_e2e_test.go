//go:build server && go_sqlite && sliver_e2e

package c2_test

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"testing"
	"time"

	implantCrypto "github.com/bishopfox/sliver/implant/sliver/cryptography"
	implantHandlers "github.com/bishopfox/sliver/implant/sliver/handlers"
	"github.com/bishopfox/sliver/implant/sliver/transports/dnsclient"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/core"
	serverCrypto "github.com/bishopfox/sliver/server/cryptography"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/bishopfox/sliver/server/transport"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

const dnsE2EEchoMsgType = uint32(0x7fffffff)

func TestDNS_EndToEndPingRPC(t *testing.T) {
	// NOTE: If you run this test in a restricted environment where writes to
	// `~/.sliver` are blocked, set `SLIVER_ROOT_DIR` to a writable temp dir.

	t.Cleanup(func() {
		for _, session := range core.Sessions.All() {
			core.Sessions.Remove(session.ID)
		}
	})

	setupDNSKeyExImplantBuild(t)

	grpcServer, grpcListener, err := transport.LocalListener()
	if err != nil {
		t.Fatalf("start local grpc listener: %v", err)
	}
	t.Cleanup(func() {
		grpcServer.Stop()
		_ = grpcListener.Close()
	})

	rpcConn, err := dialBufConn(context.Background(), grpcListener)
	if err != nil {
		t.Fatalf("dial grpc/bufconn: %v", err)
	}
	t.Cleanup(func() { _ = rpcConn.Close() })
	rpcClient := rpcpb.NewSliverRPCClient(rpcConn)

	dnsParent := "e2e.example.com."
	dnsPort := freeUDPPort(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	dnsListener, err := rpcClient.StartDNSListener(ctx, &clientpb.DNSListenerReq{
		Domains:  []string{dnsParent},
		Canaries: false,
		Host:     "127.0.0.1",
		Port:     dnsPort,
	})
	if err != nil {
		t.Fatalf("StartDNSListener: %v", err)
	}
	t.Cleanup(func() {
		if job := core.Jobs.Get(int(dnsListener.JobID)); job != nil {
			job.JobCtrl <- true
		}
		_ = db.DeleteListener(dnsListener.JobID)
	})

	// Give the UDP listener a moment to bind before the implant starts sending queries.
	time.Sleep(50 * time.Millisecond)

	stopImplant := startTestDNSImplant(t, dnsParent, dnsPort)
	t.Cleanup(stopImplant)

	sessionID := waitForSessionID(t, 10*time.Second)

	pingReq := &sliverpb.Ping{
		Nonce: 31337,
		Request: &commonpb.Request{
			SessionID: sessionID,
			Timeout:   int64(5 * time.Second),
		},
	}
	pingResp, err := rpcClient.Ping(ctx, pingReq)
	if err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if pingResp.Nonce != pingReq.Nonce {
		t.Fatalf("ping nonce mismatch: %d != %d", pingResp.Nonce, pingReq.Nonce)
	}

	// Extra validation: send a larger request/response directly via the session to
	// force message fragmentation and reassembly over the DNS C2 transport.
	session := core.Sessions.Get(sessionID)
	if session == nil {
		t.Fatalf("session not found: %s", sessionID)
	}
	large := make([]byte, 8192)
	if _, err := rand.Read(large); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	resp, err := session.Request(dnsE2EEchoMsgType, 20*time.Second, large)
	if err != nil {
		t.Fatalf("session.Request(echo): %v", err)
	}
	if !bytesEqual(resp, large) {
		t.Fatalf("echo response mismatch: %d != %d bytes", len(resp), len(large))
	}
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func freeUDPPort(t *testing.T) uint32 {
	t.Helper()
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen udp: %v", err)
	}
	defer conn.Close()
	_, portStr, err := net.SplitHostPort(conn.LocalAddr().String())
	if err != nil {
		t.Fatalf("split udp addr: %v", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("parse udp port: %v", err)
	}
	if port <= 0 || 65535 < port {
		t.Fatalf("unexpected udp port: %d", port)
	}
	return uint32(port)
}

func setupDNSKeyExImplantBuild(t *testing.T) {
	t.Helper()

	certs.SetupCAs()
	serverAgeKeyPair := serverCrypto.AgeServerKeyPair()
	peerAgeKeyPair, err := serverCrypto.RandomAgeKeyPair()
	if err != nil {
		t.Fatalf("random age key pair: %v", err)
	}

	implantCrypto.SetSecrets(
		peerAgeKeyPair.Public,
		peerAgeKeyPair.Private,
		"",
		serverAgeKeyPair.Public,
		serverCrypto.MinisignServerPublicKey(),
	)

	digest := sha256.Sum256([]byte(peerAgeKeyPair.Public))
	publicKeyDigest := hex.EncodeToString(digest[:])

	implantBuild := &models.ImplantBuild{
		Name:                "test-" + publicKeyDigest,
		PeerPublicKey:       peerAgeKeyPair.Public,
		PeerPublicKeyDigest: publicKeyDigest,
		PeerPrivateKey:      peerAgeKeyPair.Private,
		AgeServerPublicKey:  serverAgeKeyPair.Public,
	}

	if err := db.Session().Create(implantBuild).Error; err != nil {
		t.Fatalf("create implant build: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Session().Delete(implantBuild).Error
	})
}

func startTestDNSImplant(t *testing.T, dnsParent string, dnsPort uint32) func() {
	t.Helper()

	opts := &dnsclient.DNSOptions{
		QueryTimeout:       250 * time.Millisecond,
		RetryWait:          50 * time.Millisecond,
		RetryCount:         10,
		MaxErrors:          10,
		WorkersPerResolver: 1,
		NoTXT:              false,
		ForceResolvers:     fmt.Sprintf("127.0.0.1:%d", dnsPort),
	}

	// Starting the UDP listener is asynchronous relative to the gRPC call that
	// created it. Retry session init for a short time to avoid flakes.
	var (
		client *dnsclient.SliverDNSClient
		err    error
	)
	deadline := time.Now().Add(5 * time.Second)
	for {
		client, err = dnsclient.DNSStartSession(dnsParent, opts)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("DNSStartSession: %v", err)
		}
		time.Sleep(50 * time.Millisecond)
	}

	register := &sliverpb.Register{
		Name:              "e2e-dns",
		Hostname:          "localhost",
		Uuid:              uuid.NewString(),
		Username:          "unit-test",
		Os:                runtime.GOOS,
		Arch:              runtime.GOARCH,
		Pid:               int32(os.Getpid()),
		Filename:          "sliver-e2e",
		ActiveC2:          "dns://e2e",
		Version:           "e2e",
		ReconnectInterval: 0,
		ProxyURL:          "",
		Locale:            "en_US",
	}
	regData, err := proto.Marshal(register)
	if err != nil {
		t.Fatalf("marshal register: %v", err)
	}
	if err := client.WriteEnvelope(&sliverpb.Envelope{Type: sliverpb.MsgRegister, Data: regData}); err != nil {
		t.Fatalf("send register: %v", err)
	}

	handlers := implantHandlers.GetSystemHandlers()
	abort := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-abort:
				return
			default:
			}

			envelope, err := client.ReadEnvelope()
			switch err {
			case nil:
				if envelope == nil {
					time.Sleep(25 * time.Millisecond)
					continue
				}
			case dnsclient.ErrTimeout:
				continue
			case dnsclient.ErrClosed:
				return
			default:
				// Treat transient resolver errors as retryable in this test loop.
				continue
			}

			if envelope.ID == 0 {
				continue
			}
			if envelope.Type == dnsE2EEchoMsgType {
				_ = client.WriteEnvelope(&sliverpb.Envelope{ID: envelope.ID, Data: envelope.Data})
				continue
			}

			handler, ok := handlers[envelope.Type]
			if !ok {
				_ = client.WriteEnvelope(&sliverpb.Envelope{ID: envelope.ID, UnknownMessageType: true})
				continue
			}

			handler(envelope.Data, func(data []byte, err error) {
				_ = err
				_ = client.WriteEnvelope(&sliverpb.Envelope{ID: envelope.ID, Data: data})
			})
		}
	}()

	return func() {
		close(abort)
		_ = client.CloseSession()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
		}
	}
}
