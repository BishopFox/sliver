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
	"errors"
	"fmt"

	"golang.org/x/crypto/ssh"

	"github.com/bishopfox/sliver/protobuf/commpb"
	"github.com/golang/protobuf/proto"
)

type directForwarderTCP struct {
	info    *commpb.Handler // Info shared between Client, Server & Implant.
	client  *Comm           // Used for reverse portfwds
	implant *Comm           // Used for direct & reverse portfwds
}

func newDirectForwarderTCP(info *commpb.Handler, client *Comm, implant *Comm) *directForwarderTCP {
	directTCP := &directForwarderTCP{
		info:    info,
		client:  client,
		implant: implant,
	}
	return directTCP
}

func (f *directForwarderTCP) Info() *commpb.Handler {
	return f.info
}

func (f *directForwarderTCP) comms() (client, implant *Comm) {
	return f.client, f.implant
}

// start - Implements forwarder start(). Not needed for direct TCP, because
// the implant does not need to add any mapping or start any listener.
func (f *directForwarderTCP) start() (err error) {
	return
}

func (f *directForwarderTCP) handle(info *commpb.Conn, ch ssh.NewChannel) (err error) {

	// Create channel with implant
	dst, reqs, err := f.implant.sshConn.OpenChannel(commpb.Request_RouteConn.String(), ch.ExtraData())
	if err != nil {
		ch.Reject(ssh.ConnectionFailed, err.Error())
		return fmt.Errorf("Connection failed: %s", err.Error())
	}
	go ssh.DiscardRequests(reqs)

	// Accept from client.
	src, sReqs, err := ch.Accept()
	if err != nil {
		rLog.Errorf("failed to accept stream (%s)", string(ch.ExtraData()))
		ch.Reject(ssh.ConnectionFailed, err.Error())
		return err
	}
	go ssh.DiscardRequests(sReqs)

	// Pipe. Blocks until EOF, or any other error
	transportConn(src, dst)

	// Close connections once we're done, with a delay left so our
	// custom RPC tunnel has time to transmit the remaining data.
	closeConnections(src, dst)

	rLog.Warnf("Closed connections (%s:%d -> %s:%d): EOF",
		info.LHost, info.LPort, info.RHost, info.RPort)
	return nil
}

// close - There is no listener or mapping active on the implant, just delete on server.
func (f *directForwarderTCP) close() (err error) {

	// Remove from the forwarders map
	portForwarders.Remove(f.info.ID)

	return
}

// notifyClose - The implant has disconnected: we request
// the client of its forwarder to close it gracefully.
func (f *directForwarderTCP) notifyClose() (err error) {

	// We just want the client to acknowledge the request, so no response.
	data, _ := proto.Marshal(f.info)
	ok, _, err := f.client.sshConn.SendRequest(commpb.Request_PortfwdStop.String(), false, data)
	if err != nil {
		return err
	} else if !ok {
		return errors.New("Comm error: cannot request client peer to close")
	}

	return nil
}
