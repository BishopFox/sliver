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
	"github.com/bishopfox/sliver/server/certs"
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
	ID          uint32
	Session     *core.Session     // The session at the other end
	sshConn     ssh.Conn          // SSH Connection, that we will mux
	sshConfig   *ssh.ServerConfig // Encryption details.
	fingerprint string            // Implant key fingerprint

	// Connection management
	requests <-chan *ssh.Request   // Keep alive, close, etc.
	inbound  <-chan ssh.NewChannel // Inbound mux requests
	mutex    *sync.RWMutex         // Concurrency management.

	// Keep alives, maximum buffers depending on latency.
}

// Init - Sets up the Comm system (SSH session) around a RPC tunnel connection.
func Init(sess *core.Session) (err error) {

	// New Comm SSH multiplexer
	comm := &Comm{
		ID:      newID(),
		Session: sess,
		mutex:   &sync.RWMutex{},
	}

	// Pepare the SSH conn/security, and set keepalive policies/handlers.
	err = comm.setupServerAuth()
	if err != nil {
		rLog.Errorf("failed to setup SSH client connection: %s", err.Error())
		return err
	}

	// Create a new tunnel (net.Conn): this asks the remote implant to create the other end.
	conn, err := newTunnelTo(sess)
	if err != nil {
		return fmt.Errorf("failed to create tunnel: %s", err.Error())
	}

	// Start server connection.
	rLog.Infof("Starting SSH server connection...")
	comm.sshConn, comm.inbound, comm.requests, err = ssh.NewServerConn(conn, comm.sshConfig)
	if err != nil {
		rLog.Errorf("failed to initiate SSH client connection: %s", err.Error())
		return
	}
	rLog.Infof("Done")

	// Send additional requests for keepalive policies.
	go comm.serveRequests() // Serve pings & requests with keepalive policies.
	comm.checkLatency()     // Get latency for this tunnel.

	Comms.Add(comm) // Everything is up and running, register the mux.
	go comm.serve() // Handle requests (pings) and incoming connections in the background.

	return
}

// SetupAuth - The Comm prepares SSH code and security details.
func (comm *Comm) setupServerAuth() (err error) {

	// Get the implant's private key for fingerprinting its public key,
	_, implantKey, _ := certs.GetECCCertificate(certs.ImplantCA, comm.Session.Name)
	implantSigner, _ := ssh.ParsePrivateKey(implantKey)
	iKeyBytes := sha256.Sum256(implantSigner.PublicKey().Marshal())
	comm.fingerprint = base64.StdEncoding.EncodeToString(iKeyBytes[:])
	rLog.Infof("Waiting key with fingerprint: %s", comm.fingerprint)

	// Private key of the Server, for authenticating us and encryption.
	_, serverCAKey, _ := certs.GetCertificateAuthorityPEM(certs.C2ServerCA)
	signer, err := ssh.ParsePrivateKey(serverCAKey)
	if err != nil {
		rLog.Errorf("SSH failed to parse Private Key: %s", err.Error())
		return
	}

	// Config
	comm.sshConfig = &ssh.ServerConfig{
		MaxAuthTries:      3,
		PublicKeyCallback: comm.verifyImplant,
	}
	comm.sshConfig.AddHostKey(signer)

	return
}

// verifyImplant - Check the implant's host key fingerprint.
func (comm *Comm) verifyImplant(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {

	expect := comm.fingerprint
	if expect == "" {
		return nil, errors.New("No server key fingerprint")
	}

	// calculate the SHA256 hash of an SSH public key
	bytes := sha256.Sum256(key.Marshal())
	got := base64.StdEncoding.EncodeToString(bytes[:])

	_, err := base64.StdEncoding.DecodeString(expect)
	if _, ok := err.(base64.CorruptInputError); ok {
		return nil, fmt.Errorf("MD5 fingerprint (%s), update to SHA256 fingerprint: %s", expect, got)
	} else if err != nil {
		return nil, fmt.Errorf("Error decoding fingerprint: %w", err)
	}
	if got != expect {
		return nil, fmt.Errorf("Invalid fingerprint (%s)", got)
	}

	rLog.Infof("Fingerprint %s", got)

	perms := &ssh.Permissions{
		Extensions: map[string]string{"session": ""},
	}

	return perms, nil

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

// dial - Take a stream coming from the server/clients and forward it to the implant.
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
