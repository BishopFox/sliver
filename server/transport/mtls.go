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

	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/log"
	"github.com/bishopfox/sliver/server/rpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	clientLog = log.NamedLogger("transport", "client")
)

const (
	defaultHostname = ""
)

// StartClientListener - Start a mutual TLS listener
func StartClientListener(host string, port uint16) (*grpc.Server, *net.Listener, error) {
	clientLog.Infof("Starting gRPC  listener on %s:%d", host, port)
	tlsConfig := getOperatorServerTLSConfig(host)
	creds := credentials.NewTLS(tlsConfig)

	ln, err := tls.Listen("tcp", fmt.Sprintf("%s:%d", host, port), tlsConfig)
	if err != nil {
		clientLog.Error(err)
		return nil, nil, err
	}

	options := []grpc.ServerOption{grpc.Creds(creds)}
	grpcServer := grpc.NewServer(options...)
	rpcpb.RegisterSliverRPCServer(grpcServer, rpc.NewServer())
	go grpcServer.Serve(ln)
	return grpcServer, &ln, nil
}

// getOperatorServerTLSConfig - Generate the TLS configuration, we do now allow the end user
// to specify any TLS paramters, we choose sensible defaults instead
func getOperatorServerTLSConfig(host string) *tls.Config {

	caCertPtr, _, err := certs.GetCertificateAuthority(certs.OperatorCA)
	if err != nil {
		clientLog.Fatalf("Invalid ca type (%s): %v", certs.OperatorCA, host)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(caCertPtr)

	_, _, err = certs.OperatorServerGetCertificate(host)
	if err == certs.ErrCertDoesNotExist {
		certs.OperatorServerGenerateCertificate(host)
	}

	certPEM, keyPEM, err := certs.OperatorServerGetCertificate(host)
	if err != nil {
		clientLog.Errorf("Failed to generate or fetch certificate %s", err)
		return nil
	}
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		clientLog.Fatalf("Error loading server certificate: %v", err)
	}

	tlsConfig := &tls.Config{
		RootCAs:                  caCertPool,
		ClientAuth:               tls.RequireAndVerifyClientCert,
		ClientCAs:                caCertPool,
		Certificates:             []tls.Certificate{cert},
		CipherSuites:             []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384},
		PreferServerCipherSuites: true,
		MinVersion:               tls.VersionTLS12,
	}
	tlsConfig.BuildNameToCertificate()
	return tlsConfig
}
