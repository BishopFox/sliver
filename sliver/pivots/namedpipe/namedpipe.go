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
	readBufSize = 1024
)

// PivotWriteEnvelope - Writes a protobuf envolope to a generic connection
func PivotWriteEnvelope(conn *PipeConn, envelope *pb.Envelope) error {
	data, err := proto.Marshal(envelope)
	if err != nil {
		// {{if .Debug}}
		log.Print("Envelope marshaling error: ", err)
		// {{end}}
		return err
	}
	dataLengthBuf := new(bytes.Buffer)
	binary.Write(dataLengthBuf, binary.LittleEndian, uint32(len(data)))
	conn.Write(dataLengthBuf.Bytes())
	conn.Write(data)
	return nil
}

// PivotReadEnvelope - Reads a protobuf envolope from a generic connection
func PivotReadEnvelope(conn *PipeConn) (*pb.Envelope, error) {
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
	log.Printf("Found an evelope of %d bytes\n", dataLength)
	// {{end}}
	readBuf := make([]byte, readBufSize)
	dataBuf := make([]byte, 0)
	totalRead := 0
	for {
		n, err := conn.Read(readBuf)
		// {{if .Debug}}
		log.Printf("Read %d bytes with %d bytes total\n", n, n+totalRead)
		// {{end}}
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
	log.Printf("namedPipeReadEnvelope %d\n", envelope.GetType())
	// {{end}}
	return envelope, nil
}
