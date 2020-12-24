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

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/comm"
	"github.com/bishopfox/sliver/server/core"
	serverHandlers "github.com/bishopfox/sliver/server/handlers"
	"github.com/bishopfox/sliver/server/log"

	"github.com/golang/protobuf/proto"
)

const (
	// defaultServerCert - Default certificate name if bind is "" (all interfaces)
	defaultServerCert = ""

	readBufSize = 1024
)

var (
	mtlsLog = log.NamedLogger("c2", "mtls")
)

// StartMutualTLSListenerComm - Start a mutual TLS listener working with the Comm system.
func StartMutualTLSListenerComm(bindIface string, port uint16) (ln net.Listener, err error) {
	StartPivotListener()

	host := bindIface
	if host == "" {
		host = defaultServerCert
	}
	_, _, err = certs.GetCertificate(certs.C2ServerCA, certs.ECCKey, host)
	if err != nil {
		certs.C2ServerGenerateECCCertificate(host)
	}
	tlsConfig := getServerTLSConfig(host)

	mtlsLog.Infof("Starting routed TCP/mTLS listener on %s:%d", bindIface, port)

	// Get a TCP listner from the comm system.
	ln, err = comm.ListenTCP("tcp", fmt.Sprintf("%s:%d", bindIface, port))
	if err != nil {
		mtlsLog.Error(err)
		return nil, err
	}

	go acceptSliverConnectionsRoute(ln, tlsConfig)

	return ln, nil
}

func acceptSliverConnectionsRoute(ln net.Listener, config *tls.Config) {
	for {
		// Accept a normal TCP connection
		conn, err := ln.Accept()
		if err != nil {
			if errType, ok := err.(*net.OpError); ok && errType.Op == "accept" {
				break
			}
			mtlsLog.Errorf("Accept failed: %v", err)
			continue
		}

		// Upgrade to Mutual TLS
		tlsConn := tls.Client(conn, config)
		if tlsConn == nil {
			mtlsLog.Errorf("Upgrade to Mutual TLS failed: %s", err.Error())
		}

		go handleSliverConnection(tlsConn)
	}
}

// StartMutualTLSListener - Start a mutual TLS listener
func StartMutualTLSListener(bindIface string, port uint16) (net.Listener, error) {
	StartPivotListener()
	mtlsLog.Infof("Starting raw TCP/mTLS listener on %s:%d", bindIface, port)
	host := bindIface
	if host == "" {
		host = defaultServerCert
	}
	_, _, err := certs.GetCertificate(certs.C2ServerCA, certs.ECCKey, host)
	if err != nil {
		certs.C2ServerGenerateECCCertificate(host)
	}
	tlsConfig := getServerTLSConfig(host)
	ln, err := tls.Listen("tcp", fmt.Sprintf("%s:%d", bindIface, port), tlsConfig)
	if err != nil {
		mtlsLog.Error(err)
		return nil, err
	}
	go acceptSliverConnections(ln, tlsConfig)
	return ln, nil
}

func acceptSliverConnections(ln net.Listener) {
	for {
		// Accept a normal TCP connection
		conn, err := ln.Accept()
		if err != nil {
			if errType, ok := err.(*net.OpError); ok && errType.Op == "accept" {
				break
			}
			mtlsLog.Errorf("Accept failed: %v", err)
			continue
		}

		go handleSliverConnection(conn)
	}
}

func handleSliverConnection(conn io.ReadWriteCloser) {
	// mtlsLog.Infof("Accepted incoming connection: %s", conn.RemoteAddr())

	session := &core.Session{
		Transport: "mtls",
		// RemoteAddress: fmt.Sprintf("%s", conn.RemoteAddr()),
		Send:      make(chan *sliverpb.Envelope),
		RespMutex: &sync.RWMutex{},
		Resp:      map[uint64]chan *sliverpb.Envelope{},
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

// getServerTLSConfig - Generate the TLS configuration, we do now allow the end user
// to specify any TLS paramters, we choose sensible defaults instead
func getServerTLSConfig(host string) *tls.Config {

	sliverCACert, _, err := certs.GetCertificateAuthority(certs.ImplantCA)
	if err != nil {
		mtlsLog.Fatalf("Failed to find ca type (%s)", certs.ImplantCA)
	}
	sliverCACertPool := x509.NewCertPool()
	sliverCACertPool.AddCert(sliverCACert)

	certPEM, keyPEM, err := certs.GetCertificate(certs.C2ServerCA, certs.ECCKey, host)
	if err != nil {
		mtlsLog.Errorf("Failed to generate or fetch certificate %s", err)
		return nil
	}

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		mtlsLog.Fatalf("Error loading server certificate: %v", err)
	}

	tlsConfig := &tls.Config{
		RootCAs:                  sliverCACertPool,
		ClientAuth:               tls.RequireAndVerifyClientCert,
		ClientCAs:                sliverCACertPool,
		Certificates:             []tls.Certificate{cert},
		CipherSuites:             []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384},
		PreferServerCipherSuites: true,
		MinVersion:               tls.VersionTLS12,
	}
	tlsConfig.BuildNameToCertificate()
	return tlsConfig
}
