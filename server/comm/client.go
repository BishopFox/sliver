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
	"net"
	"sync"

	"github.com/golang/protobuf/proto"
	"golang.org/x/crypto/ssh"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/commpb"
	"github.com/bishopfox/sliver/server/certs"
)

// InitClient - The Comm system wires up a Client Console so that the latter can
// make use of the infrastructure for port forwards (direct/reverse) and proxies.
func InitClient(conn net.Conn, operator string) (comm *Comm, err error) {

	// New Comm SSH multiplexer
	comm = &Comm{
		ID:    newID(),
		mutex: &sync.RWMutex{},
	}

	// Pepare the SSH conn/security, and set keepalive policies/handlers.
	err = comm.setupClientAuth(operator)
	if err != nil {
		rLog.Errorf("failed to setup SSH client connection: %s", err.Error())
		return nil, err
	}

	// Start server connection.
	rLog.Infof("Starting Comm Client SSH server connection...")
	comm.sshConn, comm.inbound, comm.requests, err = ssh.NewServerConn(conn, comm.sshConfig)
	if err != nil {
		rLog.Errorf("failed to initiate SSH client connection: %s", err.Error())
		return
	}
	rLog.Infof("Done setting Comm Client")

	return
}

// SetupClientAuth - The Comm prepares SSH code and security details.
func (comm *Comm) setupClientAuth(operator string) (err error) {

	// The operator might be nil, when the client is the server binary itself,
	// in which case we load the Operators Certificate Authority private key.
	var key []byte
	if operator == "" {
		_, key, _ = certs.GetCertificateAuthorityPEM(certs.OperatorCA)
	} else {
		_, key, _ = certs.GetECCCertificate(certs.OperatorCA, "client."+operator)
	}

	// We need a key anyway, server or client.
	if len(key) == 0 {
		return errors.New("SSH Comm: no key found during Comm client auth setup")
	}

	// Get the operator private key for fingerprinting its public key.
	operatorSigner, _ := ssh.ParsePrivateKey(key)
	iKeyBytes := sha256.Sum256(operatorSigner.PublicKey().Marshal())
	comm.fingerprint = base64.StdEncoding.EncodeToString(iKeyBytes[:])
	rLog.Infof("Waiting Client key with fingerprint: %s", comm.fingerprint)

	// Private key of the Server, for authenticating us and encryption.
	_, serverCAKey, _ := certs.GetCertificateAuthorityPEM(certs.OperatorCA)
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

// ServeClient - concurrently serves all incoming (client -> server) and outgoing (client <- server) connections.
// This function is exported because when the client calls the gRPC stream function, we need it to block (and work).
func (comm *Comm) ServeClient() {

	// Serve asynchronous requests: port forwarders to be opened, closed,
	// waiting for a stream, or latency checks and keepalives.
	go func() {
		for req := range comm.requests {
			go comm.serveClientRequest(req)
		}
	}()

	// Stream requests coming from the client:
	// direct portForwarders or proxies' connections..
	for ch := range comm.inbound {
		go comm.serveClientConnection(ch)
	}
}

// serveClientConnection - A connection is coming from the client console, so we forward it to an implant.
func (comm *Comm) serveClientConnection(ch ssh.NewChannel) error {

	// Connection metadata
	info := &commpb.Conn{}
	err := proto.Unmarshal(ch.ExtraData(), info)
	if info.ID == "" && err != nil {
		rLog.Errorf("rejected connection: (bad info payload: %s)", string(ch.ExtraData()))
		return err
	}

	// Route Connections - Any connection coming from the client NOT mapped to a port forwarder
	// Generally these are proxy-handled connections, which need to be resolved by the routes first.
	if ch.ChannelType() == commpb.Request_RouteConn.String() {

		// Transport Protocols
		switch info.Transport {
		case commpb.Transport_TCP:
			err = dialClientTCP(info, ch)
			if err != nil {
				rLog.Errorf("[Client Comm] %s", err)
			}
		case commpb.Transport_UDP:
			err = dialClientUDP(info, ch)
			if err != nil {
				rLog.Errorf("[Client Comm] %s", err)
			}
		}
	}

	// Port forward Requests - Those are direct portfwds calls , which require a stream immediately:
	// TCP forwarders need it because they have a pending connection to route, and UDP forwarders
	// who need to set up their dedicated stream now.
	if ch.ChannelType() == commpb.Request_PortfwdStream.String() {

		// Find the corresponding port forwarder (transport/direction agnostic interface)
		forwarder := portForwarders.Get(info.ID)
		if forwarder == nil {
			ch.Reject(ssh.Prohibited, "No matching port forwarder for connection")
		}

		// The forwarder forwards the connection throught the implant Comm.
		err = forwarder.handle(info, ch)
		if err != nil {
			rLog.Warnf("Error transporting connections (%s:%d -> %s:%d): %v",
				info.LHost, info.LPort, info.RHost, info.RPort, err)
		}
	}

	return nil
}

// serverRequests - Handles all requests coming from the implant comm. (latency checks, close requests, etc.)
func (comm *Comm) serveClientRequest(req *ssh.Request) {

	rLog.Infof("Comm %d: Received request", comm.ID)

	// Start a port forwarding. Note that the request is just a commpb.Handler
	// wrapped into a portfwd_open request: most of the workings of a handler
	// and a portfwd are similar in the Comm system.
	if req.Type == commpb.Request_PortfwdStart.String() {

		// Request / Response
		openReq := &commpb.PortfwdOpenReq{}
		open := &commpb.PortfwdOpen{Response: &commonpb.Response{}}
		proto.Unmarshal(req.Payload, openReq)

		// Create the forwarder: this passes the request along to the appropriate
		// implant if needed, which starts any component and returns success or not.
		// The hidden value is the forwarder itself, but we don't need it as it is
		// already registered, working and therefore closable from the client.
		_, err := newForwarder(comm, openReq.Handler, openReq.Request.SessionID)
		if err != nil {
			open.Success = false
			open.Response.Err = err.Error()
		} else {
			open.Success = true
		}

		// Send back reply to the server Comm
		data, _ := proto.Marshal(open)
		req.Reply(true, data)
	}

	// Close a port forwarding
	if req.Type == commpb.Request_PortfwdStop.String() {

		// Request / Response
		closeReq := &commpb.PortfwdCloseReq{}
		clos := &commpb.PortfwdClose{Response: &commonpb.Response{}}
		proto.Unmarshal(req.Payload, closeReq)

		// Get the corresponding forwarder.
		forwarder := portForwarders.Get(closeReq.Handler.ID)
		if forwarder == nil {
			clos.Success = false
			clos.Response.Err = "No port forwarder ID matched the Portfwd Close request"
			data, _ := proto.Marshal(clos)
			req.Reply(true, data)
			return
		}

		// The forwarder sends a request to the implant, if it's a reverse
		// one that listens remotely. Also removes from forwarders map.
		// This function does not always close the connections initiated by
		// the forwarder, check implementations (TCP/UDP) for details.
		err := forwarder.close()
		if err != nil {
			clos.Success = false
			clos.Response.Err = err.Error()
		} else {
			clos.Success = true
		}

		// Send back reply to the server Comm
		data, _ := proto.Marshal(clos)
		req.Reply(true, data)
	}
}

// ShutdownClient - The implant Comm closes all active connections (bind & reverse)
// going through it. the tunnel and the SSH connection.
func (comm *Comm) ShutdownClient() (err error) {

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

	// If there are any portfwders matching this Comm, either put them on wait
	// (new streams will be yielded by a different Comm instance) or close them.
	for _, forwarder := range portForwarders.active {
		client, _ := forwarder.comms()
		if client.ID == comm.ID {
			err := forwarder.close()
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

	// Comm tunnel
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
