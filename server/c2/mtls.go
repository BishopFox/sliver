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
	"errors"
	"fmt"
	"io"
	"net"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/core"
	serverHandlers "github.com/bishopfox/sliver/server/handlers"
	"github.com/bishopfox/sliver/server/log"
	"google.golang.org/protobuf/proto"
)

const (
	// defaultServerCert - Default certificate name if bind is "" (all interfaces)
	defaultServerCert = ""
)

var (
	mtlsLog = log.NamedLogger("c2", consts.MtlsStr)
)

// StartMutualTLSListener - Start a mutual TLS listener
func StartMutualTLSListener(bindIface string, port uint16) (net.Listener, error) {
	mtlsLog.Infof("Starting raw TCP/mTLS listener on %s:%d", bindIface, port)
	host := bindIface
	if host == "" {
		host = defaultServerCert
	}
	_, _, err := certs.GetCertificate(certs.MtlsServerCA, certs.ECCKey, host)
	if err != nil {
		certs.MtlsC2ServerGenerateECCCertificate(host)
	}
	tlsConfig := getServerTLSConfig(host)
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
		conn, err := ln.Accept()
		if err != nil {
			if errType, ok := err.(*net.OpError); ok && errType.Op == "accept" {
				break // Listener was closed by the user
			}
			mtlsLog.Errorf("Accept failed: %v", err)
			continue
		}
		go handleSliverConnection(conn)
	}
}

func handleSliverConnection(conn net.Conn) {
	mtlsLog.Infof("Accepted incoming connection: %s", conn.RemoteAddr())
	implantConn := core.NewImplantConnection(consts.MtlsStr, conn.RemoteAddr().String())

	defer func() {
		mtlsLog.Debugf("mtls connection closing")
		conn.Close()
		implantConn.Cleanup()
	}()

	done := make(chan bool)
	go func() {
		defer func() {
			done <- true
		}()
		handlers := serverHandlers.GetHandlers()
		for {
			envelope, err := socketReadEnvelope(conn)
			if err != nil {
				mtlsLog.Errorf("Socket read error %v", err)
				return
			}
			implantConn.UpdateLastMessage()
			if envelope.ID != 0 {
				implantConn.RespMutex.RLock()
				if resp, ok := implantConn.Resp[envelope.ID]; ok {
					resp <- envelope // Could deadlock, maybe want to investigate better solutions
				}
				implantConn.RespMutex.RUnlock()
			} else if handler, ok := handlers[envelope.Type]; ok {
				mtlsLog.Debugf("Received new mtls message type %d, data: %s", envelope.Type, envelope.Data)
				go func() {
					respEnvelope := handler(implantConn, envelope.Data)
					if respEnvelope != nil {
						implantConn.Send <- respEnvelope
					}
				}()
			}
		}
	}()

Loop:
	for {
		select {
		case envelope := <-implantConn.Send:
			err := socketWriteEnvelope(conn, envelope)
			if err != nil {
				mtlsLog.Errorf("Socket write failed %v", err)
				break Loop
			}
		case <-done:
			break Loop
		}
	}
	mtlsLog.Debugf("Closing implant connection %s", implantConn.ID)
}

// socketWriteEnvelope - Writes a message to the TLS socket using length prefix framing
// which is a fancy way of saying we write the length of the message then the message
// e.g. [uint32 length|message] so the receiver can delimit messages properly
func socketWriteEnvelope(connection net.Conn, envelope *sliverpb.Envelope) error {
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
func socketReadEnvelope(connection net.Conn) (*sliverpb.Envelope, error) {
	// Read the first four bytes to determine data length
	dataLengthBuf := make([]byte, 4) // Size of uint32
	n, err := io.ReadFull(connection, dataLengthBuf)

	if err != nil || n != 4 {
		mtlsLog.Errorf("Socket error (read msg-length): %v", err)
		return nil, err
	}
	dataLength := int(binary.LittleEndian.Uint32(dataLengthBuf))
	if dataLength <= 0 {
		// {{if .Config.Debug}}
		mtlsLog.Printf("[pivot] read error: %s\n", err)
		// {{end}}
		return nil, errors.New("[pivot] zero data length")
	}

	dataBuf := make([]byte, dataLength)

	n, err = io.ReadFull(connection, dataBuf)

	if err != nil || n != dataLength {
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

	mtlsCACert, _, err := certs.GetCertificateAuthority(certs.MtlsImplantCA)
	if err != nil {
		mtlsLog.Fatalf("Failed to find ca type (%s)", certs.MtlsImplantCA)
	}
	mtlsCACertPool := x509.NewCertPool()
	mtlsCACertPool.AddCert(mtlsCACert)

	certPEM, keyPEM, err := certs.GetECCCertificate(certs.MtlsServerCA, host)
	if err != nil {
		mtlsLog.Errorf("Failed to generate or fetch certificate %s", err)
		return nil
	}

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		mtlsLog.Fatalf("Error loading server certificate: %v", err)
	}

	// We're not going to randomize the JARM on this one, the traffic
	// going over mTLS needs to be secure, and the JARM is fairly
	// common Golang TLS server so it's not going to be too suspicious
	tlsConfig := &tls.Config{
		RootCAs:      mtlsCACertPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    mtlsCACertPool,
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13, // Force TLS v1.3
	}

	return tlsConfig
}
