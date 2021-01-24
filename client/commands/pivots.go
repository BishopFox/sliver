package commands

import (
	"context"
	"fmt"

	cctx "github.com/bishopfox/sliver/client/context"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

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

// TCPPivot - Start a TCP pivot listener
type TCPPivot struct {
	Options struct {
		LHost   string `long:"lhost" short:"l" description:"interface address to bind listener to" default:"localhost"`
		LPort   int    `long:"lport" short:"p" description:"listener TCP listen port" default:"1789"`
		Timeout int    `long:"timeout" short:"t" description:"command timeout in seconds" default:"60"`
	} `group:"mTLS listener options"`
}

// Execute - Start a TCP pivot listener
func (tp *TCPPivot) Execute(args []string) (err error) {
	session := cctx.Context.Sliver.Session
	if session == nil {
		return
	}

	server := tp.Options.LHost
	lport := uint16(tp.Options.LPort)
	address := fmt.Sprintf("%s:%d", server, lport)

	_, err = transport.RPC.TCPListener(context.Background(), &sliverpb.TCPPivotReq{
		Address: address,
		Request: ContextRequest(session),
	})

	if err != nil {
		fmt.Printf(util.Error+"%s\n", err)
		return nil
	}

	fmt.Printf(util.Info+"Listening on tcp://%s \n", address)
	return
}

// NamedPipePivot - Start a Named pipe pivot listener
type NamedPipePivot struct {
	Options struct {
		Name string `long:"name" short:"n" description:"name of the pipe" required:"yes"`
	} `group:"named pipe options"`
}

// Execute - Start a named pipe pivot listener
func (tp *NamedPipePivot) Execute(args []string) (err error) {
	session := cctx.Context.Sliver.Session
	if session == nil {
		return
	}

	pipeName := tp.Options.Name
	_, err = transport.RPC.NamedPipes(context.Background(), &sliverpb.NamedPipesReq{
		PipeName: pipeName,
		Request:  ContextRequest(session),
	})

	if err != nil {
		fmt.Printf(util.Error+"%s\n", err)
		return nil
	}

	fmt.Printf(util.Info+"Listening on %s", "\\\\.\\pipe\\"+pipeName+" \n")
	return
}
