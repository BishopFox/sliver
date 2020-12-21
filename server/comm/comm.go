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
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"golang.org/x/crypto/ssh"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	"github.com/bishopfox/sliver/server/log"
)

var (
	rLog = log.NamedLogger("route", "routes")
)

// Comm - Wrapper around a net.Conn, adding SSH infrastructure for encryption and tunneling.
// This object is only used when the implant connection is directly tied to the server (not through a pivot).
// Therefore, a Comm may serve multiple network Routes concurrently.
type Comm struct {
	// Core
	ID        uint32
	SessionID uint32            // Any multiplexer's physical connection is tied to an implant.
	sshConn   ssh.Conn          // SSH Connection, that we will mux
	sshConfig *ssh.ServerConfig // Encryption details.

	// Connection management
	requests <-chan *ssh.Request   // Keep alive, close, etc.
	inbound  <-chan ssh.NewChannel // Inbound mux requests
	mutex    *sync.RWMutex         // Concurrency management.

	// Keep alives, maximum buffers depending on latency.
}

// NewComm - Creates a SSH-based connection multiplexer.
func NewComm() (mux *Comm) {
	mux = &Comm{
		ID:        newID(),
		sshConfig: &ssh.ServerConfig{},
		mutex:     &sync.RWMutex{},
	}
	return
}

// Init - Sets up the Comm system (SSH session) around a physical connection or a RPC tunnel connection.
// The function may return (optionally) the stream over which the session will register its RPC handlers.
func (comm *Comm) Init(conn net.Conn, sess *core.Session, key []byte) (sessionStream io.ReadWriteCloser, err error) {

	// If no conn given, we setup a conn based on a tunnel.
	if conn == nil {
		// Create a tunnel, which also asks the remote implant to create the tunnel's other end.
		conn, err = newTunnel(sess)
		if err != nil {
			return nil, fmt.Errorf("failed to create tunnel: %s", err.Error())
		}

		// Map this Comm to the session
		comm.SessionID = sess.ID
	}

	// Pepare the SSH conn/security, and set keepalive policies/handlers.
	err = comm.Setup(key)
	if err != nil {
		rLog.Errorf("failed to setup SSH client connection: %s", err.Error())
		return nil, err
	}

	// Start server connection.
	rLog.Infof("Starting SSH server connection...")
	comm.sshConn, comm.inbound, comm.requests, err = ssh.NewServerConn(conn, comm.sshConfig)
	if err != nil {
		rLog.Errorf("failed to initiate SSH client connection: %s", err.Error())
		return
	}
	rLog.Infof("Done")

	// Serve pings & requests with keepalive policies.
	go comm.serveRequests()

	// Send additional requests for keepalive policies.

	// Get latency for this tunnel.
	comm.checkLatency()

	// Get a stream for the session, if not through a tunnel.
	if sess == nil {
		// Get the first stream for registering the Session
		sessionStream, err = comm.initSessionStream()
		if err != nil {
			return
		}
	}

	// Everything is up and running, register the mux.
	Comms.Add(comm)

	// Handle requests (pings) and incoming connections in the background.
	go comm.serve()

	return
}

// Setup - The Comm prepares SSH code and security details.
func (comm *Comm) Setup(key []byte) (err error) {

	// Encryption & authentication
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		rLog.Errorf("SSH failed to parse Private Key: %s", err.Error())
		return
	}
	comm.sshConfig.AddHostKey(signer)
	comm.sshConfig.NoClientAuth = true

	// Keep-alives and other

	return
}

// serve - concurrently serves all incoming (server <- implant and outgoing (server -> implant) connections.
func (comm *Comm) serve() {
	defer func() {
		rLog.Infof("Closing SSH server connection (Comm ID: %d)...", comm.ID)
		err := comm.sshConn.Close()
		if err != nil {
			rLog.Errorf("Error closing SSH connection: %s", err.Error())
		}
		// Remove this multiplexer from our active sockets.
		Comms.Remove(comm.ID)
	}()

	// For each incoming stream request, process concurrently.
	for in := range comm.inbound {
		go comm.handleReverse(in)
	}
}

// handleReverse - Handle, validate and pipe a single logical connection coming from an implant.
func (comm *Comm) handleReverse(ch ssh.NewChannel) {

	// Parse incoming connection request details:
	info := &sliverpb.ConnectionInfo{}
	err := proto.Unmarshal(ch.ExtraData(), info)

	if info.ID == "" && err != nil {
		rLog.Errorf("rejected connection: (bad info payload: %s)", string(ch.ExtraData()))
		return
	}

	// The listener that will take on the stream.
	ln := listeners.Get(info.ID)
	if ln == nil {
		err := ch.Reject(ssh.Prohibited, "NOID")
		if err != nil {
			rLog.Errorf("Error: rejecting stream: %s", err)
		}
		rLog.Errorf("rejected stream: could not find associated abstract handler (ID: %s)", info.ID)
		return
	}

	// Accept the stream and make it a conn.
	sshChan, reqs, err := ch.Accept()
	if err != nil {
		rLog.Errorf("failed to accept stream (%s)", string(ch.ExtraData()))
		return
	}
	go ssh.DiscardRequests(reqs)

	// Populate the connection with comm properties
	conn := newConn(info, io.ReadWriteCloser(sshChan))

	// Add conn stats.
	ln.pending <- conn // We push the stream to its listener (non-blocking)
	rLog.Infof("Processed inbound stream: %s <-- %s", conn.lAddr.String(), conn.rAddr.String())
}

// dial - Take a stream coming from the server/clients and forward it to an implant node.
func (comm *Comm) dial(info *sliverpb.ConnectionInfo) (conn net.Conn, err error) {
	data, _ := proto.Marshal(info)

	// Create muxed channel and pipe.
	dst, reqs, err := comm.sshConn.OpenChannel("route", data)
	if err != nil {
		rLog.Errorf("Failed to open channel: %s", err.Error())
		return nil, fmt.Errorf("Connection failed: %s", err.Error())
	}
	go ssh.DiscardRequests(reqs)

	// Populate the connection with comm properties
	conn = newConn(info, io.ReadWriteCloser(dst))

	return
}

// initSessionStream - The multiplexer has just been created and set up, and we need to register
// the implant session for this physical connection. We create a dedicated stream and return it.
func (comm *Comm) initSessionStream() (stream io.ReadWriteCloser, err error) {
	if comm.sshConn == nil {
		rLog.Errorf("Tried to open channel on nil SSH connection")
		return
	}

	info := &sliverpb.ConnectionInfo{
		ID: "REGISTRATION",
	}
	data, _ := proto.Marshal(info)

	dst, reqs, err := comm.sshConn.OpenChannel("session", data)
	if err != nil {
		rLog.Errorf("Failed to open Session RPC channel: %s", err.Error())
		return
	}
	go ssh.DiscardRequests(reqs)

	rLog.Infof("Opened Session stream")
	return dst, nil
}

// serverRequests - Handles all requests coming from the implant comm. (latency checks, close requests, etc.)
func (comm *Comm) serveRequests() {

	for req := range comm.requests {

		// If ping, respond keepalive
		if req.Type == "keepalive" {
			err := req.Reply(true, []byte{})
			if err != nil {
				rLog.Errorf("Error replying to request")
			}
		}
	}
}

// checkLatency - get latency for this tunnel.
func (comm *Comm) checkLatency() {
	t0 := time.Now()
	_, _, err := comm.sshConn.SendRequest("latency", true, []byte{})
	if err != nil {
		rLog.Errorf("Could not check latency: %s", err.Error())
		return
	}
	rLog.Infof("Latency: %s", time.Since(t0))
}

// newID- Returns an incremental nonce as an id
func newID() uint32 {
	newID := transportID + 1
	transportID++
	return newID
}

var transportID = uint32(0)

func transport(rw1, rw2 io.ReadWriter) error {
	errc := make(chan error, 1)
	go func() {
		errc <- copyBuffer(rw1, rw2)
	}()

	go func() {
		errc <- copyBuffer(rw2, rw1)
	}()

	err := <-errc
	if err != nil && err == io.EOF {
		err = nil
	}
	return err
}

func copyBuffer(dst io.Writer, src io.Reader) error {
	buf := lPool.Get().([]byte)
	defer lPool.Put(buf)

	_, err := io.CopyBuffer(dst, src, buf)
	return err
}

var (
	sPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, smallBufferSize)
		},
	}
	mPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, mediumBufferSize)
		},
	}
	lPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, largeBufferSize)
		},
	}
)

var (
	tinyBufferSize   = 512
	smallBufferSize  = 2 * 1024  // 2KB small buffer
	mediumBufferSize = 8 * 1024  // 8KB medium buffer
	largeBufferSize  = 32 * 1024 // 32KB large buffer
)
