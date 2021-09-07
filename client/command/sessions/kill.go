package sessions

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	"errors"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
)

// KillCmd - Kill the active session (not to be confused with TerminateCmd)
func KillCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		con.PrintWarnf("No active session\n")
		return
	}
	// Confirm with the user, just in case they confused kill with terminate
	confirm := false
	prompt := &survey.Confirm{Message: "Kill the active session?"}
	survey.AskOne(prompt, &confirm, nil)
	if !confirm {
		return
	}

	err := killSession(session, ctx.Flags.Bool("force"), con)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	con.PrintInfof("Killed %s (%d)\n", session.Name, session.ID)
	con.ActiveTarget.Background()
}

func killSession(session *clientpb.Session, force bool, con *console.SliverConsoleClient) error {
	if session == nil {
		return errors.New("session does not exist")
	}
	_, err := con.Rpc.KillSession(context.Background(), &sliverpb.KillSessionReq{
		Request: &commonpb.Request{
			SessionID: session.ID,
		},
		Force: force,
	})
	return err
}
