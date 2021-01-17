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

	"github.com/golang/protobuf/proto"
	"golang.org/x/crypto/ssh"

	"github.com/bishopfox/sliver/protobuf/commpb"
)

type directForwarderUDP struct {
	info    *commpb.Handler // Info shared between Client, Server & Implant.
	client  *Comm           // Used for reverse portfwds
	implant *Comm           // Used for direct & reverse portfwds
}

func newDirectForwarderUDP(info *commpb.Handler, client *Comm, implant *Comm) *directForwarderUDP {
	directUDP := &directForwarderUDP{
		info:    info,
		client:  client,
		implant: implant,
	}
	return directUDP
}

func (f *directForwarderUDP) Info() *commpb.Handler {
	return f.info
}

func (f *directForwarderUDP) comms() (client, implant *Comm) {
	return f.client, f.implant
}

// handle - UDP forwarders often open new streams for passing packets: we open a stream
// with the implant conn and we pipe the output, however short it might be.
func (f *directForwarderUDP) handle(info *commpb.Conn, ch ssh.NewChannel) (err error) {

	// Create channel with implant
	data, _ := proto.Marshal(info)
	dst, reqs, err := f.implant.sshConn.OpenChannel(commpb.Request_RouteConn.String(), data)
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

	// Pipe. Unlike TCP piping, we keep copying data
	// between both conns even when receiving EOF errors.
	transportPacketConn(src, dst)
	if err != nil {
		rLog.Warnf("Closed UDP connection (%s:%d -> %s:%d): %s",
			info.LHost, info.LPort, info.RHost, info.RPort, err)
		return err
	}

	// Close connections once we're done, with a delay left so our
	// custom RPC tunnel has time to transmit the remaining data.
	closeConnections(src, dst)

	rLog.Warnf("Closed UDP connection (%s:%d -> %s:%d): EOF",
		info.LHost, info.LPort, info.RHost, info.RPort)
	return nil
}

// start - Asks the impant to immediately "dial" the UDP destination address
// and to return a stream, that we pass back to the Client. The portForwarder of
// the console is already registered and ready to handle the stream.
func (f *directForwarderUDP) start() (err error) {
	return nil
}

// close -
func (f *directForwarderUDP) close() (err error) {

	// Remove from the forwarders map
	portForwarders.Remove(f.info.ID)

	return
}

// notifyClose - The implant has disconnected: we request
// the client of its forwarder to close it gracefully.
func (f *directForwarderUDP) notifyClose() (err error) {

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
