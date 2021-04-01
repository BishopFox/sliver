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

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/commpb"
)

type reverseForwarderTCP struct {
	info    *commpb.Handler // Info shared between Client, Server & Implant.
	client  *Comm           // Used for reverse portfwds
	implant *Comm           // Used for direct & reverse portfwds
}

func newReverseForwarderTCP(info *commpb.Handler, client *Comm, implant *Comm) *reverseForwarderTCP {
	reverseTCP := &reverseForwarderTCP{
		info:    info,
		client:  client,
		implant: implant,
	}
	return reverseTCP
}

func (f *reverseForwarderTCP) Info() *commpb.Handler {
	return f.info
}

func (f *reverseForwarderTCP) comms() (client, implant *Comm) {
	return f.client, f.implant
}

// start - Implements forwarder start(). Not needed for direct TCP, because
// the implant does not need to add any mapping or start any listener.
func (f *reverseForwarderTCP) start() (err error) {

	// Request/Response
	req := &commpb.HandlerStartReq{
		Handler: f.info,
		Request: &commonpb.Request{SessionID: f.implant.session.ID},
	}
	data, _ := proto.Marshal(req)
	var res = &commpb.HandlerStart{}

	_, resp, err := f.implant.sshConn.SendRequest(commpb.Request_HandlerOpen.String(), true, data)
	if err != nil {
		return fmt.Errorf("Comm error: %s", err.Error())
	}
	// Else decode response
	proto.Unmarshal(resp, res)

	// The implant might give us a specific error
	if !res.Success {
		return fmt.Errorf("Portfwd error: %s", res.Response.Err)
	}

	return
}

func (f *reverseForwarderTCP) handle(info *commpb.Conn, ch ssh.NewChannel) (err error) {

	dst, reqs, err := f.client.sshConn.OpenChannel(commpb.Request_PortfwdStream.String(), ch.ExtraData())
	if err != nil {
		rLog.Errorf("Failed to open channel: %s", err.Error())
		ch.Reject(ssh.ConnectionFailed, err.Error())
		return err
	}
	go ssh.DiscardRequests(reqs)

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

	rLog.Warnf("Closed connections (%s:%d <- %s:%d): EOF",
		info.RHost, info.RPort, info.LHost, info.LPort)
	return nil
}

func (f *reverseForwarderTCP) close() (err error) {

	// If the implant comm is nil, it is most probably that the request
	// was sent by the client after receiving the notifyClose() function below.
	if f.implant == nil {
		portForwarders.Remove(f.info.ID)
		return nil
	}

	// Request the implant to stop its listener
	lnReq := &commpb.HandlerCloseReq{
		Handler: f.info,
		Request: &commonpb.Request{SessionID: f.implant.session.ID},
	}
	lnRes := &commpb.HandlerClose{Response: &commonpb.Response{}}
	err = remoteHandlerRequest(f.implant.session, lnReq, lnRes)
	if err != nil {
		rLog.Errorf("Listener (ID: %s) failed to close its remote peer (RPC error): %s",
			f.info.ID, err.Error())
	} else if !lnRes.Success && lnRes.Response.Err != "" {
		rLog.Errorf("Listener (ID: %s) failed to close its remote peer: %s",
			f.info.ID, lnRes.Response.Err)
	}

	// Remove from the forwarders map
	portForwarders.Remove(f.info.ID)

	return nil
}

// notifyClose - The implant has disconnected: we request
// the client of its forwarder to close it gracefully.
func (f *reverseForwarderTCP) notifyClose() (err error) {

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
