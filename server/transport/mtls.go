package transport

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"runtime/debug"
	"strings"

	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/log"
	"github.com/bishopfox/sliver/server/rpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	kb = 1024
	mb = kb * 1024
	gb = mb * 1024

	// ServerMaxMessageSize - Server-side max GRPC message size
	ServerMaxMessageSize = 2 * gb
)

var (
	mtlsLog = log.NamedLogger("transport", "mtls")
)

// StartMtlsClientListener - Start a mutual TLS listener
func StartMtlsClientListener(host string, port uint16) (*grpc.Server, net.Listener, error) {
	mtlsLog.Infof("Starting gRPC/mtls  listener on %s:%d", host, port)
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		mtlsLog.Error(err)
		return nil, nil, err
	}

	grpcServer, err := StartMtlsClientServer(ln)
	if err != nil {
		ln.Close()
		return nil, nil, err
	}
	return grpcServer, ln, nil
}

// StartMtlsClientServer serves the authenticated multiplayer gRPC server on an
// existing listener. This is primarily useful for tests that need the full mTLS
// + auth stack without opening a real TCP socket.
func StartMtlsClientServer(ln net.Listener) (*grpc.Server, error) {
	if ln == nil {
		return nil, errors.New("listener is required")
	}

	tlsConfig := getOperatorServerTLSConfig("multiplayer")
	if tlsConfig == nil {
		return nil, errors.New("failed to create operator TLS config")
	}

	creds := credentials.NewTLS(tlsConfig)
	options := []grpc.ServerOption{
		grpc.Creds(creds),
		grpc.MaxRecvMsgSize(ServerMaxMessageSize),
		grpc.MaxSendMsgSize(ServerMaxMessageSize),
	}
	options = append(options, grpcKeepaliveOptions()...)
	options = append(options, initMiddleware(true)...)
	grpcServer := grpc.NewServer(options...)
	rpcpb.RegisterSliverRPCServer(grpcServer, rpc.NewServer())
	go func() {
		defer func() {
			if r := recover(); r != nil {
				mtlsLog.Errorf("gRPC server panic: %v\n%s", r, string(debug.Stack()))
			}
		}()
		if err := grpcServer.Serve(ln); err != nil && !isExpectedGRPCServerExit(err) {
			mtlsLog.Warnf("gRPC server exited with error: %v", err)
		}
	}()
	return grpcServer, nil
}

func isExpectedGRPCServerExit(err error) bool {
	if err == nil {
		return true
	}
	if errors.Is(err, grpc.ErrServerStopped) || errors.Is(err, net.ErrClosed) {
		return true
	}

	// The gVisor-backed listener used by the WireGuard multiplayer transport can
	// surface this accept error during normal shutdown on Windows.
	errString := err.Error()
	return strings.Contains(errString, "use of closed network connection") ||
		strings.Contains(errString, "endpoint is in invalid state")
}

// getOperatorServerTLSConfig - Generate the TLS configuration, we do now allow the end user
// to specify any TLS parameters, we choose sensible defaults instead
func getOperatorServerTLSConfig(host string) *tls.Config {
	caCertPtr, _, err := certs.GetCertificateAuthority(certs.OperatorCA)
	if err != nil {
		mtlsLog.Fatalf("Invalid ca type (%s): %v", certs.OperatorCA, host)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(caCertPtr)

	certPEM, keyPEM, err := ensureCurrentOperatorServerCertificate(host, caCertPtr)
	if err != nil {
		mtlsLog.Errorf("Failed to generate or fetch certificate %s", err)
		return nil
	}
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		mtlsLog.Fatalf("Error loading server certificate: %v", err)
	}

	tlsConfig := &tls.Config{
		RootCAs:      caCertPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caCertPool,
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
		VerifyConnection: func(state tls.ConnectionState) error {
			return certs.ValidateOperatorClientCertificate(state.PeerCertificates)
		},
	}

	return tlsConfig
}

func ensureCurrentOperatorServerCertificate(host string, caCert *x509.Certificate) ([]byte, []byte, error) {
	certPEM, keyPEM, err := certs.OperatorServerGetCertificate(host)
	if err != nil && !errors.Is(err, certs.ErrCertDoesNotExist) {
		return nil, nil, err
	}
	if errors.Is(err, certs.ErrCertDoesNotExist) || !operatorServerCertMatchesCA(certPEM, caCert) {
		if _, _, genErr := certs.OperatorServerGenerateCertificate(host); genErr != nil {
			return nil, nil, genErr
		}
		return certs.OperatorServerGetCertificate(host)
	}
	return certPEM, keyPEM, nil
}

func operatorServerCertMatchesCA(certPEM []byte, caCert *x509.Certificate) bool {
	if len(certPEM) == 0 || caCert == nil {
		return false
	}
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return false
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return false
	}
	roots := x509.NewCertPool()
	roots.AddCert(caCert)
	_, err = cert.Verify(x509.VerifyOptions{Roots: roots})
	return err == nil
}
