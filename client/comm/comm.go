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
	"net"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"

	"github.com/bishopfox/sliver/client/log"
	"github.com/bishopfox/sliver/protobuf/commpb"
)

// Comm - The network subsystem of the client console. Equivalent to an implant Comm.
// This object is tied to the server's Comm system.
type Comm struct {
	// Core
	SessionID     uint32            // Any multiplexer's physical connection is tied to an implant.
	RemoteAddress string            // The multiplexer may be tied to a pivoted implant.
	ssh           ssh.Conn          // SSH Connection, that we will mux
	config        *ssh.ClientConfig // We are the talking to the C2 server.
	fingerprint   string            // A key fingerprint to authenticate the server/pivot.
	Log           *logrus.Logger    // Client logger, passed around to subcomponents

	// Connection management
	requests <-chan *ssh.Request   // Keep alive
	inbound  <-chan ssh.NewChannel // Inbound mux requests
	mutex    *sync.RWMutex         // Concurrency management.
	pending  int                   // Number of actively processed connections.
}

// InitClient - Sets up and start a SSH connection (multiplexer) around a logical connection/stream, or a Session RPC.
func InitClient(conn net.Conn, key []byte, fingerprint string) (err error) {
	// Return if we don't have what we need.
	if conn == nil {
		return errors.New("net.Conn is nil, cannot init Comm")
	}

	// New SSH multiplexer Comm
	comm := &Comm{
		fingerprint: fingerprint,
		mutex:       &sync.RWMutex{}}

	// Register the client logger. The comm field will be overriden.
	comm.Log = log.NewClientLogger("comm")

	// Pepare the SSH conn/security, and set keepalive policies/handlers.
	err = comm.setupAuthClient(key, "")
	if err != nil {
		return err
	}

	// We are the client of the C2 server here.
	comm.ssh, comm.inbound, comm.requests, err = ssh.NewClientConn(conn, "", comm.config)
	if err != nil {
		return
	}

	// Serve pings & requests with keepalive policies.
	go comm.serveRequests()

	// Unique Comm instance for this console client.
	ClientComm = comm

	// Handle incoming connections in the background.
	go comm.serve()

	return
}

// setupAuthClient - The Comm prepares SSH code and security details for a client connection.
func (comm *Comm) setupAuthClient(key []byte, name string) (err error) {

	// Private key is used to authenticate to the server/pivot.
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return
	}
	comm.config = &ssh.ClientConfig{
		User:            name,                                     // TODO: Add name from config.
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)}, // Authenticate as an operator
		HostKeyCallback: comm.verifyServer,                        // Checks the key given by the server
	}

	return
}

// verifyServer - Check the server's host key fingerprint. We have an exampled compiled in above.
func (comm *Comm) verifyServer(hostname string, remote net.Addr, key ssh.PublicKey) error {
	expect := comm.fingerprint
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
	comm.Log.Debugf("Server SSH fingerprint: %s", got)

	return nil
}

// serve - concurrently serves all incoming (server <- implant) connections.
func (comm *Comm) serve() {
	defer func() {
		comm.ssh.Close()
	}()

	// For each incoming stream request, analyse connection information and dispatch.
	// We don't check the Channel type, because only reverse port forwarders can yield
	// streams through channel requests.
	for ch := range comm.inbound {

		// Parse incoming connection request details
		info := &commpb.Conn{}
		err := proto.Unmarshal(ch.ExtraData(), info)

		// Reject any stream without an ID
		if info.ID == "" && err != nil {
			return
		}

		// Get the (reverse) port forwarder that will handle the connection.
		forwarder := Forwarders.Get(info.ID)
		if forwarder == nil {
			ch.Reject(ssh.Prohibited, "No matching forwarder")
			return
		}

		// Go handle it in the background
		go forwarder.handleReverse(ch)
	}
}

// serverRequests - Handles all requests coming from the implant comm. (latency checks, close requests, etc.)
func (comm *Comm) serveRequests() {

	for req := range comm.requests {

		// Port forwarder close requests - When an implant has disconnected
		// the server-side forwarder sends us a request to close associated listeners.
		if req.Type == commpb.Request_PortfwdStop.String() {

			info := &commpb.Handler{}
			proto.Unmarshal(req.Payload, info)

			// Get the forwarder and close it
			forwarder := Forwarders.Get(info.ID)
			if forwarder != nil {
				err := forwarder.Close(true)
				if err == nil {
					comm.Log.Errorf("%s %s forwarder closed (session disconnect)\n",
						forwarder.Info().Type.String(), forwarder.Info().Transport.String())
				}
			}

			// If ping, respond keepalive
			if req.Type == "keepalive" {
				req.Reply(true, []byte{})
			}
		}
	}
}
