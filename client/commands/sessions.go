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

	"github.com/bishopfox/sliver/client/connection"
	cctx "github.com/bishopfox/sliver/client/context"
	"github.com/bishopfox/sliver/client/util"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// Interact - Interact with a Sliver implant. This commands changes the console
// context, with different commands and completions.
type Interact struct {
	Positional struct {
		Implant string `description:"Session ID or name"` // Name or ID, command will say.
	} `positional-args:"yes" required:"yes"`
}

// Execute - Interact with a Sliver implant.
func (i *Interact) Execute(args []string) (err error) {

	session := GetSession(i.Positional.Implant)
	if session != nil {
		cctx.Context.Sliver = &cctx.Session{Session: session} // This will be noticed by all components in need.
		cctx.Context.Menu = cctx.Sliver                       // Except this one.
		fmt.Printf(util.Info+"Active session %s (%d)\n", session.Name, session.ID)
	} else {
		fmt.Printf(util.Error+"Invalid session name or session number '%s'\n", i.Positional.Implant)
	}

	// For the moment, we ask the current working directory to implant...
	pwd, err := connection.RPC.Pwd(context.Background(), &sliverpb.PwdReq{
		Request: cctx.Context.Sliver.Request(10),
	})
	if err != nil {
		fmt.Printf(util.Error+"%s\n", err)
	} else {
		cctx.Context.Sliver.WorkingDir = pwd.Path
	}

	return
}

// Background - Exit from implant context.
type Background struct{}

// Execute - Exit from implant context.
func (b *Background) Execute(args []string) (err error) {
	cctx.Context.Menu = cctx.Server // Coming back to server main menu
	cctx.Context.Sliver = nil
	fmt.Printf(util.Info + "Background ...\n")
	return
}

// GetSession - Get session by session ID or name
func GetSession(arg string) *clientpb.Session {
	sessions, err := connection.RPC.GetSessions(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(util.Error+"%s\n", err)
		return nil
	}
	for _, session := range sessions.GetSessions() {
		if session.Name == arg || fmt.Sprintf("%d", session.ID) == arg {
			return session
		}
	}
	return nil
}
