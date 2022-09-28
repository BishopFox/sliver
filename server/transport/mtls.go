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
	"fmt"
	"net"
	"runtime/debug"

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

// StartClientListener - Start a mutual TLS listener
func StartClientListener(host string, port uint16) (*grpc.Server, net.Listener, error) {
	mtlsLog.Infof("Starting gRPC  listener on %s:%d", host, port)

	tlsConfig := getOperatorServerTLSConfig("multiplayer")

	creds := credentials.NewTLS(tlsConfig)
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		mtlsLog.Error(err)
		return nil, nil, err
	}
	options := []grpc.ServerOption{
		grpc.Creds(creds),
		grpc.MaxRecvMsgSize(ServerMaxMessageSize),
		grpc.MaxSendMsgSize(ServerMaxMessageSize),
	}
	options = append(options, initMiddleware(true)...)
	grpcServer := grpc.NewServer(options...)
	rpcpb.RegisterSliverRPCServer(grpcServer, rpc.NewServer())
	go func() {
		panicked := true
		defer func() {
			if panicked {
				mtlsLog.Errorf("stacktrace from panic: %s", string(debug.Stack()))
			}
		}()
		if err := grpcServer.Serve(ln); err != nil {
			mtlsLog.Warnf("gRPC server exited with error: %v", err)
		} else {
			panicked = false
		}
	}()
	return grpcServer, ln, nil
}

// getOperatorServerTLSConfig - Generate the TLS configuration, we do now allow the end user
// to specify any TLS paramters, we choose sensible defaults instead
func getOperatorServerTLSConfig(host string) *tls.Config {
	caCertPtr, _, err := certs.GetCertificateAuthority(certs.OperatorCA)
	if err != nil {
		mtlsLog.Fatalf("Invalid ca type (%s): %v", certs.OperatorCA, host)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(caCertPtr)

	_, _, err = certs.OperatorServerGetCertificate(host)
	if err == certs.ErrCertDoesNotExist {
		certs.OperatorServerGenerateCertificate(host)
	}

	certPEM, keyPEM, err := certs.OperatorServerGetCertificate(host)
	if err != nil {
		mtlsLog.Errorf("Failed to generate or fetch certificate %s", err)
		return nil
	}
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		mtlsLog.Fatalf("Error loading server certificate: %v", err)
	}

	tlsConfig := &tls.Config{
		RootCAs:                  caCertPool,
		ClientAuth:               tls.RequireAndVerifyClientCert,
		ClientCAs:                caCertPool,
		Certificates:             []tls.Certificate{cert},
		PreferServerCipherSuites: true,
		MinVersion:               tls.VersionTLS13,
	}
	if certs.TLSKeyLogger != nil {
		tlsConfig.KeyLogWriter = certs.TLSKeyLogger
	}

	tlsConfig.BuildNameToCertificate()
	return tlsConfig
}
