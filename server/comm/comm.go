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
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

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
	ID      uint32
	session *core.Session // The session at the other end
	mutex   *sync.RWMutex // Concurrency management.

	// Duplex connection SSH
	tunnel      *tunnel           // Tunnel (duplex connection on top of implant RPC loop)
	sshConn     ssh.Conn          // SSH Connection, that we will mux
	sshConfig   *ssh.ServerConfig // Encryption details.
	fingerprint string            // Implant key fingerprint

	// Connection management
	requests <-chan *ssh.Request   // Reverse handlers open/close, latency, etc.
	inbound  <-chan ssh.NewChannel // Inbound connections (bind handlers)
	active   []io.ReadWriteCloser  // All connections (bind/reverse) goind through

	// Keep alives, maximum buffers depending on latency.
}

// verifyPeer - Check the other end's key fingerprint (implant, or client console)
func (comm *Comm) verifyPeer(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {

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
