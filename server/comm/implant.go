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
	"fmt"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"golang.org/x/crypto/ssh"

	"github.com/bishopfox/sliver/protobuf/commpb"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/core"
)

// InitSession - Sets up the Comm system (SSH session) around a RPC tunnel connection.
func InitSession(sess *core.Session) (err error) {

	// New Comm SSH multiplexer
	comm := &Comm{
		ID:      newID(),
		session: sess,
		mutex:   &sync.RWMutex{},
	}

	// Pepare the SSH conn/security, and set keepalive policies/handlers.
	err = comm.setupServerAuth()
	if err != nil {
		rLog.Errorf("failed to setup SSH client connection: %s", err.Error())
		return err
	}

	// Create a new tunnel (net.Conn): this asks the remote implant to create the other end.
	comm.tunnel, err = newTunnelTo(sess)
	if err != nil {
		return fmt.Errorf("failed to create tunnel: %s", err.Error())
	}

	// Some C2 protocols may require a bit of time before starting to send data over the tunnel.
	time.Sleep(500 * time.Millisecond)

	// Start server connection.
	rLog.Infof("Starting SSH server connection...")
	comm.sshConn, comm.inbound, comm.requests, err = ssh.NewServerConn(comm.tunnel, comm.sshConfig)
	if err != nil {
		rLog.Errorf("failed to initiate SSH client connection: %s", err.Error())
		return
	}

	// Everything is up and running, register the mux.
	Comms.Add(comm)
	rLog.Infof("Comm started")
	comm.checkLatency() // Get latency for this tunnel.

	go comm.serve() // Handle requests (pings) and incoming connections in the background.

	return
}

// SetupAuth - The Comm prepares SSH code and security details.
func (comm *Comm) setupServerAuth() (err error) {

	// Get the implant's private key for fingerprinting its public key,
	_, implantKey, _ := certs.GetECCCertificate(certs.ImplantCA, comm.session.Name)
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
		PublicKeyCallback: comm.verifyPeer,
	}
	comm.sshConfig.AddHostKey(signer)

	return
}

// serve - concurrently serves all incoming (server <- implant) connections.
func (comm *Comm) serve() {

	// Serve pings & requests with keepalive policies.
	go comm.serveImplantRequests()

	// Incoming connections from reverse handlers.
	for in := range comm.inbound {
		go comm.handleReverseConnection(in)
	}
}

// handleReverseConnection - Handle, validate and pipe a single logical connection coming from an implant.
func (comm *Comm) handleReverseConnection(ch ssh.NewChannel) error {

	// We don't check the Channel type, because this stream destination
	// is directly check against either listeners or port forwarders IDs.
	info := &commpb.Conn{}
	err := proto.Unmarshal(ch.ExtraData(), info)

	if info.ID == "" && err != nil {
		rLog.Errorf("rejected connection: (bad info payload: %s)", string(ch.ExtraData()))
		return err
	}

	// If this connection is mapped to a Comm abstracted listener,
	// (used by Sliver C2 , or even by a library) let the listener
	// process and "package" the connection
	listener := listeners.Get(info.ID)
	if listener != nil {
		return listener.handleReverse(info, ch)
	}

	// If this connection is mapped to a port forwader,
	// let it push the conn back to its client console.
	forwarder := portForwarders.Get(info.ID)
	if forwarder != nil {
		return forwarder.handle(info, ch)
	}

	// Otherwise this connection is illegal, as none of the clients
	// nor the server have a corresponding handler ID
	ch.Reject(ssh.Prohibited, "")
	return fmt.Errorf("Could not handle incoming channel: no listeners or port forwarders")
}

// serveImplantRequests - Handles all requests coming from the
// implant comm. (latency checks, close requests, etc.)
func (comm *Comm) serveImplantRequests() {

	for req := range comm.requests {

		// Ping
		if req.Type == "keepalive" {
			err := req.Reply(true, []byte{})
			if err != nil {
				rLog.Errorf("Error replying to request")
			}
		}
	}
}

// ShutdownImplant - The implant Comm closes all active connections
// (bind & reverse) going through it. the tunnel and the SSH connection.
// The function is called (defered) in connection loops of the C2 package.
// Also notifies associated client port forwarders to close themselves.
func (comm *Comm) ShutdownImplant() (err error) {

	rLog.Infof("Closing Comm (ID: %d)...", comm.ID)

	// If there are any active connections going through this Comm, close them.
	if len(comm.active) > 0 {
		for _, conn := range comm.active {
			err = conn.Close()
			if err != nil {
				rLog.Errorf("Error closing Comm connection: %s", err.Error())
			}
		}
	}

	// If there are any listeners matching this Comm, close them.
	for _, ln := range listeners.active {
		if ln.comms().ID == comm.ID {
			err := ln.Close()
			if err != nil {
				rLog.Errorf("Error closing listener on Comm close: %s", err.Error())
			}
		}
	}

	// If there are any portfwders matching this Comm,
	// send a request to their client to shut them down.
	for _, forwarder := range portForwarders.active {
		_, implant := forwarder.comms()
		if implant.ID == comm.ID {
			err := forwarder.notifyClose()
			if err != nil {
				rLog.Errorf("Error closing port forwarder on Comm close: %s", err.Error())
			}
		}
	}

	// SSH
	err = comm.sshConn.Close()
	if err != nil {
		rLog.Errorf("Error closing SSH connection: %s", err.Error())
	}

	// Comm custom RPC tunnel
	if comm.tunnel != nil {
		err = comm.tunnel.Close()
		if err != nil {
			rLog.Errorf("Error closing Comm tunnel : %s", err.Error())
		}
	}

	// Remove this multiplexer from our active sockets.
	Comms.Remove(comm.ID)

	return
}
