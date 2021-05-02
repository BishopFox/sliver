package sliver

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
	"strings"

	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/client/spin"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// Execute - Execute a program on the remote system
type Execute struct {
	Positional struct {
		Args []string `description:"command arguments" required:"1"`
	} `positional-args:"yes" required:"yes"`
	Options struct {
		Silent bool `long:"silent" short:"s" description:"don't print the command output"`
		Token  bool `long:"token" short:"T" description:"execute command with current token (Windows only)"`
	} `group:"execute options"`
}

// Execute - Execute a program on the remote system
func (e *Execute) Execute(args []string) (err error) {

	cmdPath := e.Positional.Args[0]
	var cArgs []string
	if len(e.Positional.Args) > 1 {
		cArgs = e.Positional.Args[1:]
	}
	output := e.Options.Silent
	ctrl := make(chan bool)
	var exec *sliverpb.Execute
	msg := fmt.Sprintf("Executing %s %s...", cmdPath, strings.Join(cArgs, " "))
	go spin.Until(msg, ctrl)

	if e.Options.Token {
		exec, err = transport.RPC.ExecuteToken(context.Background(), &sliverpb.ExecuteTokenReq{
			Request: core.ActiveSessionRequest(),
			Path:    cmdPath,
			Args:    args,
			Output:  !output,
		})
	} else {
		exec, err = transport.RPC.Execute(context.Background(), &sliverpb.ExecuteReq{
			Request: core.ActiveSessionRequest(),
			Path:    cmdPath,
			Args:    args,
			Output:  !output,
		})
	}

	ctrl <- true
	<-ctrl
	if err != nil {
		fmt.Printf(util.Error+"%s", err)
	} else if !output {
		if exec.Status != 0 {
			fmt.Printf(util.Error+"Exited with status %d!\n", exec.Status)
			if exec.Result != "" {
				fmt.Printf(util.Info+"Output:\n%s\n", exec.Result)
			}
		} else {
			fmt.Printf(util.Info+"Output:\n%s", exec.Result)
		}
	}
	return
}
