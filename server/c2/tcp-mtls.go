package c2

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
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/golang/protobuf/proto"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/core"
	serverHandlers "github.com/bishopfox/sliver/server/handlers"
	"github.com/bishopfox/sliver/server/log"
)

const (
	// defaultServerCert - Default certificate name if bind is "" (all interfaces)
	defaultServerCert = ""

	readBufSize = 1024
)

var (
	mtlsLog = log.NamedLogger("c2", "mtls")
)

// StartMutualTLSListener - Start a mutual TLS listener, on the server only (no implant routes used).
func StartMutualTLSListener(bindIface string, port uint16) (net.Listener, error) {
	StartPivotListener()

	host := bindIface
	if host == "" {
		host = defaultServerCert
	}
	_, _, err := certs.GetCertificate(certs.C2ServerCA, certs.ECCKey, host)
	if err != nil {
		certs.C2ServerGenerateECCCertificate(host)
	}

	// Load the TLS configuration for a server position.
	tlsConfig := newCredentialsTLS().ServerConfig(host)

	mtlsLog.Infof("Starting raw TCP/mTLS listener on %s:%d", bindIface, port)

	ln, err := tls.Listen("tcp", fmt.Sprintf("%s:%d", bindIface, port), tlsConfig)
	if err != nil {
		mtlsLog.Error(err)
		return nil, err
	}
	go acceptSliverConnections(ln)
	return ln, nil
}

func acceptSliverConnections(ln net.Listener) {
	for {
		// Accept a normal TCP connection
		conn, err := ln.Accept()
		if err != nil {
			if errType, ok := err.(*net.OpError); ok && errType.Op == "accept" {
				mtlsLog.Errorf("Accept failed: %v", err)
				break
			}
			continue
		}

		// For some reason when closing the listener
		// from the Comm system returns a nil connection...
		if conn == nil {
			mtlsLog.Errorf("Accepted a nil conn: %v", err)
			break
		}

		go handleSliverConnection(conn)
	}
}

func handleSliverConnection(conn net.Conn) {
	// mtlsLog.Infof("Accepted incoming connection: %s", conn.RemoteAddr())

	session := &core.Session{
		Transport:     "mtls",
		Send:          make(chan *sliverpb.Envelope),
		RespMutex:     &sync.RWMutex{},
		Resp:          map[uint64]chan *sliverpb.Envelope{},
		RemoteAddress: conn.RemoteAddr().String(),
	}
	session.UpdateCheckin()

	defer func() {
		mtlsLog.Debugf("Cleaning up for %s", session.Name)
		core.Sessions.Remove(session.ID)
		conn.Close()
	}()

	done := make(chan bool)

	go func() {
		defer func() {
			done <- true
		}()
		handlers := serverHandlers.GetSessionHandlers()
		for {
			envelope, err := socketReadEnvelope(conn)
			if err != nil {
				mtlsLog.Errorf("Socket read error %v", err)
				return
			}
			session.UpdateCheckin()
			if envelope.ID != 0 {
				session.RespMutex.RLock()
				if resp, ok := session.Resp[envelope.ID]; ok {
					resp <- envelope // Could deadlock, maybe want to investigate better solutions
				}
				session.RespMutex.RUnlock()
			} else if handler, ok := handlers[envelope.Type]; ok {
				go handler.(func(*core.Session, []byte))(session, envelope.Data)
			}
		}
	}()

Loop:
	for {
		select {
		case envelope := <-session.Send:
			err := socketWriteEnvelope(conn, envelope)
			if err != nil {
				mtlsLog.Errorf("Socket write failed %v", err)
				break Loop
			}
		case <-done:
			break Loop
		}
	}
	mtlsLog.Infof("Closing connection to session %s", session.Name)
}

// socketWriteEnvelope - Writes a message to the TLS socket using length prefix framing
// which is a fancy way of saying we write the length of the message then the message
// e.g. [uint32 length|message] so the receiver can delimit messages properly
func socketWriteEnvelope(connection io.ReadWriteCloser, envelope *sliverpb.Envelope) error {
	data, err := proto.Marshal(envelope)
	if err != nil {
		mtlsLog.Errorf("Envelope marshaling error: %v", err)
		return err
	}
	dataLengthBuf := new(bytes.Buffer)
	binary.Write(dataLengthBuf, binary.LittleEndian, uint32(len(data)))
	connection.Write(dataLengthBuf.Bytes())
	connection.Write(data)
	return nil
}

// socketReadEnvelope - Reads a message from the TLS connection using length prefix framing
// returns messageType, message, and error
func socketReadEnvelope(connection io.ReadWriteCloser) (*sliverpb.Envelope, error) {

	// Read the first four bytes to determine data length
	dataLengthBuf := make([]byte, 4) // Size of uint32
	_, err := connection.Read(dataLengthBuf)
	if err != nil {
		mtlsLog.Errorf("Socket error (read msg-length): %v", err)
		return nil, err
	}
	dataLength := int(binary.LittleEndian.Uint32(dataLengthBuf))

	// Read the length of the data, keep in mind each call to .Read() may not
	// fill the entire buffer length that we specify, so instead we use two buffers
	// readBuf is the result of each .Read() operation, which is then concatinated
	// onto dataBuf which contains all of data read so far and we keep calling
	// .Read() until the running total is equal to the length of the message that
	// we're expecting or we get an error.
	readBuf := make([]byte, readBufSize)
	dataBuf := make([]byte, 0)
	totalRead := 0
	for {
		n, err := connection.Read(readBuf)
		dataBuf = append(dataBuf, readBuf[:n]...)
		totalRead += n
		if totalRead == dataLength {
			break
		}
		if err != nil {
			mtlsLog.Errorf("Read error: %s", err)
			break
		}
	}

	if err != nil {
		mtlsLog.Errorf("Socket error (read data): %v", err)
		return nil, err
	}
	// Unmarshal the protobuf envelope
	envelope := &sliverpb.Envelope{}
	err = proto.Unmarshal(dataBuf, envelope)
	if err != nil {
		mtlsLog.Errorf("Un-marshaling envelope error: %v", err)
		return nil, err
	}
	return envelope, nil
}

// tlsConfig - A wrapper around several elements needed to produce a TLS config for either
// a server or a client, depending on the direction of the connection to the implant.
type tlsConfig struct {
	ca   *x509.CertPool
	cert tls.Certificate
	key  []byte
}

// newCredentialsTLS - Generates a new custom tlsConfig loaded with the Slivers Certificate Authority.
// It may thus load and export any TLS configuration for talking with an implant, bind or reverse.
func newCredentialsTLS() (creds *tlsConfig) {

	// The Certificate Authority is needed by all TLS configs, whether server or client.
	sliverCACert, _, err := certs.GetCertificateAuthority(certs.ImplantCA)
	if err != nil {
		mtlsLog.Fatalf("Failed to find ca type (%s)", certs.ImplantCA)
	}
	sliverCACertPool := x509.NewCertPool()
	sliverCACertPool.AddCert(sliverCACert)

	creds = &tlsConfig{
		ca: sliverCACertPool,
	}

	return creds
}

// ClientConfig - TLS config used when we dial an implant over Mutual TLS.
// This makes use of a custom function for skipping (only) hostname validation,
// because the tlsConfig verifies the peer only against its own Certificate Authority.
func (t *tlsConfig) ClientConfig(host string) (c *tls.Config) {

	// The host is the address of the host on which the implant is listening.
	certPEM, keyPEM, err := certs.GetCertificate(certs.C2ServerCA, certs.ECCKey, host)
	if err != nil {
		mtlsLog.Errorf("Failed to generate or fetch certificate %s", err)
		return nil
	}

	t.cert, err = tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		mtlsLog.Fatalf("Error loading server certificate: %v", err)
	}

	// Client config with custom certificate validation routine
	c = &tls.Config{
		Certificates:          []tls.Certificate{t.cert},
		RootCAs:               t.ca,
		InsecureSkipVerify:    true, // Don't worry I sorta know what I'm doing
		VerifyPeerCertificate: t.rootOnlyVerifyCertificate,
	}
	c.BuildNameToCertificate()

	return c
}

// ServerConfig - TLS config used when we listen for incoming Mutual TLS implant connections.
func (t *tlsConfig) ServerConfig(host string) (c *tls.Config) {

	// The host is either the server host, or a host on which an implant is listening.
	// If the host is one of an implant's, the server acts as if it ways this host.
	certPEM, keyPEM, err := certs.GetCertificate(certs.C2ServerCA, certs.ECCKey, host)
	if err != nil {
		mtlsLog.Errorf("Failed to generate or fetch certificate %s", err)
		return nil
	}

	t.cert, err = tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		mtlsLog.Fatalf("Error loading server certificate: %v", err)
	}

	// Server configuration
	c = &tls.Config{
		RootCAs:                  t.ca,
		ClientAuth:               tls.RequireAndVerifyClientCert,
		ClientCAs:                t.ca,
		Certificates:             []tls.Certificate{t.cert},
		CipherSuites:             []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384},
		PreferServerCipherSuites: true,
		MinVersion:               tls.VersionTLS12,
	}
	c.BuildNameToCertificate()

	return
}

// rootOnlyVerifyCertificate - Go doesn't provide a method for only skipping hostname validation so
// we have to disable all of the fucking certificate validation and re-implement everything.
// https://github.com/golang/go/issues/21971
func (t *tlsConfig) rootOnlyVerifyCertificate(rawCerts [][]byte, _ [][]*x509.Certificate) error {

	cert, err := x509.ParseCertificate(rawCerts[0]) // We should only get one cert
	if err != nil {
		return err
	}

	// Basically we only care if the certificate was signed by our authority
	// Go selects sensible defaults for time and EKU, basically we're only
	// skipping the hostname check, I think?
	options := x509.VerifyOptions{
		Roots: t.ca,
	}
	if _, err := cert.Verify(options); err != nil {
		return err
	}

	return nil
}
