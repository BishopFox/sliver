package transport

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"sliver/server/assets"
	"sliver/server/certs"
)

const (
	defaultServerCert = "clients"
)

// StartClientListener - Start a mutual TLS listener
func StartClientListener(bindIface string, port uint16) (net.Listener, error) {
	log.Printf("Starting Raw TCP/TLS listener on %s:%d", bindIface, port)
	hostCert := bindIface
	if hostCert == "" {
		hostCert = defaultServerCert
	}
	tlsConfig := getServerTLSConfig(certs.ClientsCertDir, hostCert)
	ln, err := tls.Listen("tcp", fmt.Sprintf("%s:%d", bindIface, port), tlsConfig)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	go acceptClientConnections(ln)
	return ln, nil
}

func acceptClientConnections(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			if errType, ok := err.(*net.OpError); ok && errType.Op == "accept" {
				break
			}
			log.Printf("Accept failed: %v", err)
			continue
		}
		go handleClientConnection(conn)
	}
}

func handleClientConnection(conn net.Conn) {
	log.Printf("Accepted incoming connection: %s", conn.RemoteAddr())

}

// getServerTLSConfig - Generate the TLS configuration, we do now allow the end user
// to specify any TLS paramters, we choose sensible defaults instead
func getServerTLSConfig(caType string, host string) *tls.Config {

	rootDir := assets.GetRootAppDir()

	caCertPtr, _, err := certs.GetCertificateAuthority(rootDir, caType)
	if err != nil {
		log.Fatalf("Invalid ca type (%s): %v", caType, host)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(caCertPtr)

	certPEM, keyPEM, _ := certs.GetServerCertificatePEM(rootDir, caType, host, true)
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		log.Fatalf("Error loading server certificate: %v", err)
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
