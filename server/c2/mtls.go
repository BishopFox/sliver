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
	"bufio"
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/core"
	serverHandlers "github.com/bishopfox/sliver/server/handlers"
	"github.com/bishopfox/sliver/server/log"
	"github.com/hashicorp/yamux"
	"google.golang.org/protobuf/proto"
)

const (
	// defaultServerCert - Default certificate name if bind is "" (all interfaces)
	defaultServerCert = ""

	// ServerMaxMessageSize - Server-side max GRPC message size
	ServerMaxMessageSize = (2 * 1024 * 1024 * 1024) - 1

	mtlsYamuxPreface = "SLIVER/YAMUX/1\n"

	mtlsYamuxMaxConcurrentStreams = 128
	mtlsYamuxMaxConcurrentSends   = 64
)

var (
	mtlsLog = log.NamedLogger("c2", consts.MtlsStr)

	mtlsYamuxPrefaceBytes = []byte(mtlsYamuxPreface)
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

	br := bufio.NewReader(conn)
	bufferedConn := &mtlsBufferedConn{Conn: conn, r: br}

	preface, err := br.Peek(len(mtlsYamuxPrefaceBytes))
	if err == nil && bytes.Equal(preface, mtlsYamuxPrefaceBytes) {
		if _, err := br.Discard(len(mtlsYamuxPrefaceBytes)); err != nil {
			mtlsLog.Errorf("Failed to discard yamux preface: %v", err)
			return
		}
		handleSliverConnectionYamux(bufferedConn, implantConn)
		return
	}
	handleSliverConnectionLegacy(bufferedConn, implantConn)
}

type mtlsBufferedConn struct {
	net.Conn
	r *bufio.Reader
}

func (c *mtlsBufferedConn) Read(p []byte) (int, error) {
	return c.r.Read(p)
}

func handleSliverConnectionLegacy(conn net.Conn, implantConn *core.ImplantConnection) {
	done := make(chan struct{})
	var doneOnce sync.Once
	closeDone := func() {
		doneOnce.Do(func() {
			close(done)
		})
	}

	go func() {
		defer closeDone()
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
				resp, ok := implantConn.Resp[envelope.ID]
				implantConn.RespMutex.RUnlock()
				if ok {
					resp <- envelope
				}
			} else if handler, ok := handlers[envelope.Type]; ok {
				mtlsLog.Debugf("Received new mtls message type %d, data: %s", envelope.Type, envelope.Data)
				go func(envelope *sliverpb.Envelope) {
					respEnvelope := handler(implantConn, envelope.Data)
					if respEnvelope != nil {
						implantConn.Send <- respEnvelope
					}
				}(envelope)
			}
		}
	}()

	for {
		select {
		case envelope := <-implantConn.Send:
			if err := socketWriteEnvelope(conn, envelope); err != nil {
				mtlsLog.Errorf("Socket write failed %v", err)
				closeDone()
				return
			}
		case <-done:
			return
		}
	}
}

func handleSliverConnectionYamux(conn net.Conn, implantConn *core.ImplantConnection) {
	session, err := yamux.Server(conn, nil)
	if err != nil {
		mtlsLog.Errorf("Failed to initialize yamux session: %v", err)
		return
	}
	defer session.Close()

	done := make(chan struct{})
	var doneOnce sync.Once
	closeDone := func() {
		doneOnce.Do(func() {
			close(done)
			session.Close()
		})
	}

	streamSem := make(chan struct{}, mtlsYamuxMaxConcurrentStreams)
	sendSem := make(chan struct{}, mtlsYamuxMaxConcurrentSends)
	handlers := serverHandlers.GetHandlers()

	go func() {
		defer closeDone()
		for {
			stream, err := session.Accept()
			if err != nil {
				if !errors.Is(err, io.EOF) {
					mtlsLog.Errorf("yamux accept error: %v", err)
				}
				return
			}

			select {
			case streamSem <- struct{}{}:
			case <-done:
				stream.Close()
				return
			}

			go func(stream net.Conn) {
				defer func() {
					<-streamSem
				}()
				defer stream.Close()

				envelope, err := socketReadEnvelope(stream)
				if err != nil {
					mtlsLog.Errorf("Stream read error %v", err)
					return
				}
				implantConn.UpdateLastMessage()

				if envelope.ID != 0 {
					implantConn.RespMutex.RLock()
					resp, ok := implantConn.Resp[envelope.ID]
					implantConn.RespMutex.RUnlock()
					if ok {
						resp <- envelope
					}
					return
				}

				if handler, ok := handlers[envelope.Type]; ok {
					mtlsLog.Debugf("Received new mtls message type %d, data: %s", envelope.Type, envelope.Data)
					go func(envelope *sliverpb.Envelope) {
						respEnvelope := handler(implantConn, envelope.Data)
						if respEnvelope != nil {
							implantConn.Send <- respEnvelope
						}
					}(envelope)
				}
			}(stream)
		}
	}()

	go func() {
		defer closeDone()
		for {
			select {
			case envelope := <-implantConn.Send:
				select {
				case sendSem <- struct{}{}:
				case <-done:
					return
				}

				go func(envelope *sliverpb.Envelope) {
					defer func() {
						<-sendSem
					}()

					stream, err := session.Open()
					if err != nil {
						mtlsLog.Errorf("yamux open stream error: %v", err)
						closeDone()
						return
					}
					defer stream.Close()

					if err := socketWriteEnvelope(stream, envelope); err != nil {
						mtlsLog.Errorf("Stream write failed %v", err)
						closeDone()
						return
					}
				}(envelope)

			case <-done:
				return
			}
		}
	}()

	<-done
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
	if err := binary.Write(dataLengthBuf, binary.LittleEndian, uint32(len(data))); err != nil {
		mtlsLog.Errorf("Envelope marshaling error: %v", err)
		return err
	}
	if _, err := connection.Write(dataLengthBuf.Bytes()); err != nil {
		return err
	}
	if _, err := connection.Write(data); err != nil {
		return err
	}
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
	if dataLength <= 0 || ServerMaxMessageSize < dataLength {
		// {{if .Config.Debug}}
		mtlsLog.Printf("[pivot] read error: %s\n", err)
		// {{end}}
		return nil, errors.New("[pivot] invalid data length")
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
	if certs.TLSKeyLogger != nil {
		tlsConfig.KeyLogWriter = certs.TLSKeyLogger
	}
	return tlsConfig
}
