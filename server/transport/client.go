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
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	consts "github.com/bishopfox/sliver/client/constants"
	clientpb "github.com/bishopfox/sliver/protobuf/client"
	sliverpb "github.com/bishopfox/sliver/protobuf/sliver"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/log"
	"github.com/bishopfox/sliver/server/rpc"

	"github.com/golang/protobuf/proto"
	"github.com/sirupsen/logrus"
)

var (
	clientLog = log.NamedLogger("transport", "client")
)

const (
	defaultHostname = ""
	readBufSize     = 1024
)

var (
	once = &sync.Once{}
)

// StartClientListener - Start a mutual TLS listener
func StartClientListener(bindIface string, port uint16) (net.Listener, error) {
	clientLog.Infof("Starting Raw TCP/TLS listener on %s:%d", bindIface, port)
	tlsConfig := getOperatorServerTLSConfig(bindIface)
	ln, err := tls.Listen("tcp", fmt.Sprintf("%s:%d", bindIface, port), tlsConfig)
	if err != nil {
		clientLog.Error(err)
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
			clientLog.Errorf("Accept failed: %v", err)
			continue
		}
		go handleClientConnection(conn)
	}
}

func logState(tlsConn *tls.Conn) {
	clientLog.Debug(">>>>>>>>>>>>>>>> TLS State <<<<<<<<<<<<<<<<")
	state := tlsConn.ConnectionState()
	clientLog.Debugf("Version: %x", state.Version)
	clientLog.Debugf("HandshakeComplete: %t", state.HandshakeComplete)
	clientLog.Debugf("DidResume: %t", state.DidResume)
	clientLog.Debugf("CipherSuite: %x", state.CipherSuite)
	clientLog.Debugf("NegotiatedProtocol: %s", state.NegotiatedProtocol)
	clientLog.Debugf("NegotiatedProtocolIsMutual: %t", state.NegotiatedProtocolIsMutual)
	clientLog.Debug("Certificate chain:")
	for i, cert := range state.PeerCertificates {
		subject := cert.Subject
		issuer := cert.Issuer
		clientLog.Debugf(" %d s:/C=%v/ST=%v/L=%v/O=%v/OU=%v/CN=%s", i, subject.Country, subject.Province, subject.Locality, subject.Organization, subject.OrganizationalUnit, subject.CommonName)
		clientLog.Debugf("   i:/C=%v/ST=%v/L=%v/O=%v/OU=%v/CN=%s", issuer.Country, issuer.Province, issuer.Locality, issuer.Organization, issuer.OrganizationalUnit, issuer.CommonName)
	}
	clientLog.Debug(">>>>>>>>>>>>>>>> State End <<<<<<<<<<<<<<<<")
}

func handleClientConnection(conn net.Conn) {
	defer conn.Close()
	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		return
	}
	tlsConn.Read([]byte{}) // Unless you read 0 bytes the TLS handshake will not complete
	logState(tlsConn)
	certs := tlsConn.ConnectionState().PeerCertificates
	if len(certs) < 1 {
		return
	}
	operator := certs[0].Subject.CommonName // Get operator name from cert CN
	clientLog.Infof("Accepted incoming client connection: %s (%s)", conn.RemoteAddr(), operator)

	log.AuditLogger.WithFields(logrus.Fields{
		"pkg":      "transport",
		"operator": operator,
	}).Info("connected")

	client := core.GetClient(certs[0])
	core.Clients.AddClient(client)

	core.EventBroker.Publish(core.Event{
		EventType: consts.JoinedEvent,
		Client:    client,
	})

	cleanup := func() {
		clientLog.Infof("Closing connection to client (%s)", client.Operator)
		log.AuditLogger.WithFields(logrus.Fields{
			"pkg":      "transport",
			"operator": client.Operator,
		}).Info("disconnected")
		core.Clients.RemoveClient(client.ID)
		conn.Close()
		core.EventBroker.Publish(core.Event{
			EventType: consts.LeftEvent,
			Client:    client,
		})
	}

	go func() {
		defer once.Do(cleanup)
		rpcHandlers := rpc.GetRPCHandlers()
		tunHandlers := rpc.GetTunnelHandlers()
		for {
			envelope, err := socketReadEnvelope(conn)
			if err != nil {
				clientLog.Errorf("Socket read error %v", err)
				return
			}
			// RPC
			if rpcHandler, ok := (*rpcHandlers)[envelope.Type]; ok {
				timeout := time.Duration(envelope.Timeout)
				go rpcHandler(envelope.Data, timeout, func(data []byte, err error) {
					errStr := ""
					if err != nil {
						errStr = fmt.Sprintf("%v", err)
					}
					client.Send <- &sliverpb.Envelope{
						ID:   envelope.ID,
						Data: data,
						Err:  errStr,
					}
				})
				log.AuditLogger.WithFields(logrus.Fields{
					"pkg":           "transport",
					"operator":      client.Operator,
					"envelope_type": envelope.Type,
				}).Info("rpc command")
			} else if tunHandler, ok := (*tunHandlers)[envelope.Type]; ok {
				go tunHandler(client, envelope.Data, func(data []byte, err error) {
					errStr := ""
					if err != nil {
						errStr = fmt.Sprintf("%v", err)
					}
					client.Send <- &sliverpb.Envelope{
						ID:   envelope.ID,
						Data: data,
						Err:  errStr,
					}
				})
			} else {
				client.Send <- &sliverpb.Envelope{
					ID:   envelope.ID,
					Data: []byte{},
					Err:  "Unknown rpc command",
				}
			}
		}
	}()

	events := core.EventBroker.Subscribe()
	defer core.EventBroker.Unsubscribe(events)
	go socketEventLoop(conn, events)

	defer once.Do(cleanup)
	for envelope := range client.Send {
		err := socketWriteEnvelope(conn, envelope)
		if err != nil {
			clientLog.Errorf("Socket error %v", err)
			return
		}
	}
}

func socketEventLoop(conn net.Conn, events chan core.Event) {
	for event := range events {
		pbEvent := &clientpb.Event{
			EventType: event.EventType,
			Data:      event.Data,
		}

		if event.Job != nil {
			pbEvent.Job = event.Job.ToProtobuf()
		}
		if event.Client != nil {
			pbEvent.Client = event.Client.ToProtobuf()
		}
		if event.Sliver != nil {
			pbEvent.Sliver = event.Sliver.ToProtobuf()
		}
		if event.Err != nil {
			pbEvent.Err = fmt.Sprintf("%v", event.Err)
		}

		data, _ := proto.Marshal(pbEvent)
		envelope := &sliverpb.Envelope{
			Type: clientpb.MsgEvent,
			Data: data,
		}
		err := socketWriteEnvelope(conn, envelope)
		if err != nil {
			clientLog.Errorf("Socket write failed %v", err)
			return
		}
	}
}

// socketWriteEnvelope - Writes a message to the TLS socket using length prefix framing
// which is a fancy way of saying we write the length of the message then the message
// e.g. [uint32 length|message] so the reciever can delimit messages properly
func socketWriteEnvelope(connection net.Conn, envelope *sliverpb.Envelope) error {
	data, err := proto.Marshal(envelope)
	if err != nil {
		clientLog.Errorf("Envelope marshaling error: %v", err)
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
	_, err := connection.Read(dataLengthBuf)
	if err != nil {
		clientLog.Errorf("Socket error (read msg-length): %v", err)
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
			clientLog.Errorf("Read error: %s", err)
			break
		}
	}

	if err != nil {
		clientLog.Errorf("Socket error (read data): %v", err)
		return nil, err
	}
	// Unmarshal the protobuf envelope
	envelope := &sliverpb.Envelope{}
	err = proto.Unmarshal(dataBuf, envelope)
	if err != nil {
		clientLog.Errorf("unmarshaling envelope error: %v", err)
		return nil, err
	}
	return envelope, nil
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
