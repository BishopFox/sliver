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
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
)

const (
	kb = 1024
	mb = kb * 1024
	gb = mb * 1024

	// ClientMaxReceiveMessageSize - Max gRPC message size ~2Gb
	ClientMaxReceiveMessageSize = (2 * gb) - 1 // 2Gb - 1 byte

	defaultTimeout = time.Duration(10 * time.Second)
)

type TokenAuth struct {
	token string
}

// Return value is mapped to request headers.
func (t TokenAuth) GetRequestMetadata(ctx context.Context, in ...string) (map[string]string, error) {
	return map[string]string{
		"Authorization": "Bearer " + t.token,
	}, nil
}

func (TokenAuth) RequireTransportSecurity() bool {
	return true
}

// MTLSConnect - Connect to the sliver server
func MTLSConnect(config *assets.ClientConfig) (rpcpb.SliverRPCClient, *grpc.ClientConn, error) {
	tlsConfig, err := getTLSConfig(config.CACertificate, config.Certificate, config.PrivateKey)
	if err != nil {
		return nil, nil, err
	}
	transportCreds := credentials.NewTLS(tlsConfig)
	callCreds := credentials.PerRPCCredentials(TokenAuth{token: config.Token})
	options := []grpc.DialOption{
		grpc.WithTransportCredentials(transportCreds),
		grpc.WithPerRPCCredentials(callCreds),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(ClientMaxReceiveMessageSize)),
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	connection, err := grpc.DialContext(ctx, fmt.Sprintf("%s:%d", config.LHost, config.LPort), options...)
	if err != nil {
		return nil, nil, err
	}
	return rpcpb.NewSliverRPCClient(connection), connection, nil
}

func getTLSConfig(caCertificate string, certificate string, privateKey string) (*tls.Config, error) {

	certPEM, err := tls.X509KeyPair([]byte(certificate), []byte(privateKey))
	if err != nil {
		log.Printf("Cannot parse client certificate: %v", err)
		return nil, err
	}

	// Load CA cert
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM([]byte(caCertificate))

	// Setup config with custom certificate validation routine
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{certPEM},
		RootCAs:            caCertPool,
		InsecureSkipVerify: true, // Don't worry I sorta know what I'm doing
		VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			return RootOnlyVerifyCertificate(caCertificate, rawCerts)
		},
	}
	return tlsConfig, nil
}

// RootOnlyVerifyCertificate - Go doesn't provide a method for only skipping hostname validation so
// we have to disable all of the certificate validation and re-implement everything.
// https://github.com/golang/go/issues/21971
func RootOnlyVerifyCertificate(caCertificate string, rawCerts [][]byte) error {
	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM([]byte(caCertificate))
	if !ok {
		log.Printf("Failed to parse root certificate")
		os.Exit(3)
	}

	cert, err := x509.ParseCertificate(rawCerts[0]) // We should only get one cert
	if err != nil {
		log.Printf("Failed to parse certificate: " + err.Error())
		return err
	}

	// Basically we only care if the certificate was signed by our authority
	// Go selects sensible defaults for time and EKU, basically we're only
	// skipping the hostname check, I think?
	options := x509.VerifyOptions{
		Roots: roots,
	}
	if options.Roots == nil {
		panic("no root certificate")
	}
	if _, err := cert.Verify(options); err != nil {
		log.Printf("Failed to verify certificate: " + err.Error())
		return err
	}

	return nil
}
