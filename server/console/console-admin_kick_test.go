package console

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	clientassets "github.com/bishopfox/sliver/client/assets"
	clienttransport "github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/core"
	servertransport "github.com/bishopfox/sliver/server/transport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestKickOperatorClosesActiveEventStreams(t *testing.T) {
	certs.SetupCAs()

	listener := bufconn.Listen(2 * 1024 * 1024)
	grpcServer, err := servertransport.StartMtlsClientServer(listener)
	if err != nil {
		t.Fatalf("start mTLS client server: %v", err)
	}
	defer grpcServer.Stop()
	defer listener.Close()

	operatorName := uniqueKickOperatorName(t)
	t.Cleanup(func() {
		_ = removeOperator(operatorName)
		_ = revokeOperatorClientCertificate(operatorName)
		closeOperatorStreams(operatorName)
	})

	config := mustNewOperatorAssetsConfig(t, operatorName)
	rpcClient, conn, err := mustMTLSBufconnClient(t, listener, config)
	if err != nil {
		t.Fatalf("connect operator client: %v", err)
	}
	defer conn.Close()

	stream, err := rpcClient.Events(context.Background(), &commonpb.Empty{})
	if err != nil {
		t.Fatalf("start events stream: %v", err)
	}

	waitForCondition(t, 3*time.Second, func() bool {
		return operatorActive(operatorName)
	}, "operator to appear in the active client registry")

	recvErr := make(chan error, 1)
	go func() {
		_, err := stream.Recv()
		recvErr <- err
	}()

	if err := kickOperator(operatorName); err != nil {
		t.Fatalf("kick operator: %v", err)
	}

	select {
	case err := <-recvErr:
		if err == nil {
			t.Fatal("expected kicked operator event stream to close")
		}
		if errors.Is(err, io.EOF) {
			break
		}
		code := status.Code(err)
		if code != codes.Canceled && code != codes.Unavailable && code != codes.Unknown {
			t.Fatalf("expected stream cancellation after kick, got %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for kicked operator event stream to close")
	}

	waitForCondition(t, 3*time.Second, func() bool {
		return !operatorActive(operatorName)
	}, "operator to leave the active client registry")

	_, err = rpcClient.GetVersion(context.Background(), &commonpb.Empty{})
	if err == nil {
		t.Fatal("expected kicked operator to be denied on subsequent RPCs")
	}
	code := status.Code(err)
	if code != codes.Unauthenticated && code != codes.Unavailable {
		t.Fatalf("expected unauthenticated or unavailable after kick, got %v", err)
	}

	exists, err := operatorExists(operatorName)
	if err != nil {
		t.Fatalf("lookup operator after kick: %v", err)
	}
	if exists {
		t.Fatal("expected operator record to be removed after kick")
	}

	_, _, err = certs.OperatorClientGetCertificate(operatorName)
	if !errors.Is(err, certs.ErrCertDoesNotExist) {
		t.Fatalf("expected operator certificate to be removed after kick, got %v", err)
	}
}

func mustNewOperatorAssetsConfig(t *testing.T, operatorName string) *clientassets.ClientConfig {
	t.Helper()

	configJSON, err := NewOperatorConfig(operatorName, "bufnet", 31337, []string{"all"}, false)
	if err != nil {
		t.Fatalf("generate operator config: %v", err)
	}

	config := &clientassets.ClientConfig{}
	if err := json.Unmarshal(configJSON, config); err != nil {
		t.Fatalf("parse operator config: %v", err)
	}
	return config
}

func mustMTLSBufconnClient(t *testing.T, listener *bufconn.Listener, config *clientassets.ClientConfig) (rpcpb.SliverRPCClient, *grpc.ClientConn, error) {
	t.Helper()

	dialCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(dialCtx, "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(credentials.NewTLS(mustClientTLSConfig(t, config))),
		grpc.WithPerRPCCredentials(staticTokenAuth(config.Token)),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(clienttransport.ClientMaxReceiveMessageSize)),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, nil, err
	}
	return rpcpb.NewSliverRPCClient(conn), conn, nil
}

func mustClientTLSConfig(t *testing.T, config *clientassets.ClientConfig) *tls.Config {
	t.Helper()

	clientCert, err := tls.X509KeyPair([]byte(config.Certificate), []byte(config.PrivateKey))
	if err != nil {
		t.Fatalf("parse client certificate: %v", err)
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM([]byte(config.CACertificate))

	return &tls.Config{
		Certificates:       []tls.Certificate{clientCert},
		RootCAs:            caCertPool,
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS13,
		VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			return clienttransport.RootOnlyVerifyCertificate(config.CACertificate, rawCerts)
		},
	}
}

func operatorActive(name string) bool {
	for _, active := range core.Clients.ActiveOperators() {
		if active == name {
			return true
		}
	}
	return false
}

func waitForCondition(t *testing.T, timeout time.Duration, predicate func() bool, description string) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if predicate() {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", description)
}

func uniqueKickOperatorName(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("kick-operator-%d", time.Now().UnixNano())
}

type staticTokenAuth string

func (t staticTokenAuth) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{
		"Authorization": "Bearer " + string(t),
	}, nil
}

func (staticTokenAuth) RequireTransportSecurity() bool {
	return true
}
