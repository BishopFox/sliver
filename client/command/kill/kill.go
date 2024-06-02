package kill

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
	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// KillCmd - Kill the active session (not to be confused with TerminateCmd)
func KillCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	// Confirm with the user, just in case they confused kill with terminate
	confirm := false
	con.PrintWarnf("WARNING: This will kill the remote implant process\n\n")
	if session != nil {
		survey.AskOne(&survey.Confirm{Message: "Kill the active session?"}, &confirm, nil)
		if !confirm {
			return
		}
		err := KillSession(session, cmd, con)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		con.PrintInfof("Killed %s (%s)\n", session.Name, session.ID)
		con.ActiveTarget.Background()
		return
	} else if beacon != nil {
		survey.AskOne(&survey.Confirm{Message: "Kill the active beacon?"}, &confirm, nil)
		if !confirm {
			return
		}
		err := KillBeacon(beacon, cmd, con)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		con.PrintInfof("Killed %s (%s)\n", beacon.Name, beacon.ID)
		con.ActiveTarget.Background()
		return
	}
	con.PrintErrorf("No active session or beacon\n")
}

func KillSession(session *clientpb.Session, cmd *cobra.Command, con *console.SliverClient) error {
	if session == nil {
		return errors.New("session does not exist")
	}
	timeout, _ := cmd.Flags().GetInt64("timeout")
	force, _ := cmd.Flags().GetBool("force")

	// remove any active socks proxies
	socks := core.SocksProxies.List()
	if len(socks) != 0 {
		for _, p := range socks {
			if p.SessionID == session.ID {
				core.SocksProxies.Remove(p.ID)
			}
		}
	}

	_, err := con.Rpc.Kill(context.Background(), &sliverpb.KillReq{
		Request: &commonpb.Request{
			SessionID: session.ID,
			Timeout:   timeout,
		},
		Force: force,
	})
	return err
}

func KillBeacon(beacon *clientpb.Beacon, cmd *cobra.Command, con *console.SliverClient) error {
	if beacon == nil {
		return errors.New("session does not exist")
	}

	timeout, _ := cmd.Flags().GetInt64("timeout")
	force, _ := cmd.Flags().GetBool("force")

	_, err := con.Rpc.Kill(context.Background(), &sliverpb.KillReq{
		Request: &commonpb.Request{
			BeaconID: beacon.ID,
			Timeout:  timeout,
		},
		Force: force,
	})
	return err
}
