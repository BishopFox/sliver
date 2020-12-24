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

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/sliver/comm"
	"github.com/bishopfox/sliver/sliver/transports"
	"github.com/golang/protobuf/proto"
)

var routeHandlers = map[uint32]RouteHandler{
	sliverpb.MsgAddRouteReq: addRouteHandler,
	sliverpb.MsgRmRouteReq:  removeRouteHandler,
}

// GetSystemRouteHandlers - Returns a map of route handlers
func GetSystemRouteHandlers() map[uint32]RouteHandler {
	return routeHandlers
}

// ---------------- Route Handlers ----------------

func addRouteHandler(envelope *sliverpb.Envelope, connection *transports.Connection) {

	// {{if .Config.Debug}}
	log.Printf("Entered route handler")
	// {{end}}

	// Request / Response
	addRouteReq := &sliverpb.AddRouteReq{}
	err := proto.Unmarshal(envelope.Data, addRouteReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}
	addRoute := &sliverpb.AddRoute{Response: &commonpb.Response{}}

	// Add the route and map it to a comm multiplexer if needed
	comm.Routes.Add(addRouteReq.Route)

	addRoute.Success = true
	data, _ := proto.Marshal(addRoute)
	connection.Send <- &sliverpb.Envelope{
		ID:   envelope.GetID(),
		Data: data,
	}
	// {{if .Config.Debug}}
	log.Printf("Returned route response")
	// {{end}}
}

func removeRouteHandler(envelope *sliverpb.Envelope, connection *transports.Connection) {

	// Request / Response
	// rmRouteReq := &sliverpb.RmRouteReq{}
	// err := proto.Unmarshal(envelope.Data, rmRouteReq)
	// if err != nil {
	//         // {{if .Config.Debug}}
	//         log.Printf("error decoding message: %v", err)
	//         // {{end}}
	//         return
	// }
	// rmRoute := &sliverpb.RmRoute{Response: &commonpb.Response{}}
	//
	// // Remove handler from router
	// route.Routes.Server.Off(bon.Route(rmRouteReq.Route.ID))
	//
	// // {{if .Config.Debug}}
	// log.Printf("Removed route (ID: %d)", rmRouteReq.Route.ID)
	// // {{end}}
	//
	// rmRoute.Success = true
	//
	// data, _ := proto.Marshal(rmRoute)
	// connection.Send <- &sliverpb.Envelope{
	//         ID:   envelope.GetID(),
	//         Data: data,
	// }
}

func startHandler(envelope *sliverpb.Envelope, connection *transports.Connection) {

}
