package pivots

import (
	"bytes"
	"encoding/binary"
	"net"

	// {{if .Debug}}
	"log"
	// {{end}}

	pb "github.com/bishopfox/sliver/protobuf/sliver"
	"github.com/golang/protobuf/proto"
)

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

const (
	readBufSize  = 256
	writeBufSize = 256
)

// PivotWriteEnvelope - Writes a protobuf envolope to a generic connection
func PivotWriteEnvelope(conn *net.Conn, envelope *pb.Envelope) error {

	// {{if .Debug}}
	log.Printf("IN pivots.PivotWriteEnvelope\n")
	// {{end}}

	data, err := proto.Marshal(envelope)
	if err != nil {
		// {{if .Debug}}
		log.Print("Envelope marshaling error: ", err)
		// {{end}}
		return err
	}
	dataLengthBuf := new(bytes.Buffer)
	binary.Write(dataLengthBuf, binary.LittleEndian, uint32(len(data)))
	_, err = (*conn).Write(dataLengthBuf.Bytes())
	if err != nil {
		// {{if .Debug}}
		log.Printf("pivots.PivotWriteEnvelope error %v and %d\n", err, dataLengthBuf)
		// {{end}}
	}
	totalWritten := 0
	for totalWritten < len(data)-writeBufSize {
		n, err2 := (*conn).Write(data[totalWritten : totalWritten+writeBufSize])
		totalWritten += n
		if err2 != nil {
			// {{if .Debug}}
			log.Printf("pivots.PivotWriteEnvelope error %v\n", err)
			// {{end}}
		}
		// {{if .Debug}}
		log.Printf("pivots.PivotWriteEnvelope WRITE LOOP totalWritten=%d n=%d TOTAL=%d\n", totalWritten, n, len(data))
		// {{end}}
	}
	if totalWritten < len(data) {
		missing := len(data) - totalWritten
		_, err := (*conn).Write(data[totalWritten : totalWritten+missing])
		if err != nil {
			// {{if .Debug}}
			log.Printf("pivots.PivotWriteEnvelope error %v\n", err)
			// {{end}}
		}
	}

	// {{if .Debug}}
	log.Printf("OUT pivots.PivotWriteEnvelope\n")
	// {{end}}

	return nil
}

// PivotReadEnvelope - Reads a protobuf envolope from a generic connection
func PivotReadEnvelope(conn *net.Conn) (*pb.Envelope, error) {

	// {{if .Debug}}
	log.Printf("IN pivots.PivotReadEnvelope\n")
	// {{end}}

	dataLengthBuf := make([]byte, 4)
	_, err := (*conn).Read(dataLengthBuf)
	if err != nil {
		// {{if .Debug}}
		log.Printf("Named Pipe error (read msg-length): %v\n", err)
		// {{end}}
		return nil, err
	}
	dataLength := int(binary.LittleEndian.Uint32(dataLengthBuf))
	// {{if .Debug}}
	log.Printf("pivots.PivotReadEnvelope found envolope of %d bytes\n", dataLength)
	// {{end}}
	readBuf := make([]byte, readBufSize)
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
			// {{if .Debug}}
			log.Printf("Read error: %s\n", err)
			// {{end}}
			break
		}
	}
	envelope := &pb.Envelope{}
	err = proto.Unmarshal(dataBuf, envelope)
	if err != nil {
		// {{if .Debug}}
		log.Printf("Unmarshaling envelope error: %v", err)
		// {{end}}
		return &pb.Envelope{}, err
	}

	// {{if .Debug}}
	log.Printf("OUT pivots.PivotReadEnvelope\n")
	// {{end}}

	return envelope, nil
}
