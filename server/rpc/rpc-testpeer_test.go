package rpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
)

func contextWithCommonName(commonName string) context.Context {
	cert := &x509.Certificate{Subject: pkix.Name{CommonName: commonName}}
	tlsInfo := credentials.TLSInfo{
		State: tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{cert}}},
	}
	return peer.NewContext(context.Background(), &peer.Peer{AuthInfo: tlsInfo})
}
