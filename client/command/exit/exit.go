package exit

/*
	Sliver Implant Framework
	Copyright (C) 2023  Bishop Fox

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
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/spf13/cobra"
)

// ExitCmd - Exit the console.
func ExitCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	fmt.Println("Exiting...")
	if con.IsServer {
		sessions, err := con.Rpc.GetSessions(context.Background(), &commonpb.Empty{})
		if err != nil {
			os.Exit(1)
		}
		beacons, err := con.Rpc.GetBeacons(context.Background(), &commonpb.Empty{})
		if err != nil {
			os.Exit(1)
		}
		if 0 < len(sessions.Sessions) || 0 < len(beacons.Beacons) {
			con.Printf("There are %d active sessions and %d active beacons.\n", len(sessions.Sessions), len(beacons.Beacons))
			confirm := false
			prompt := &survey.Confirm{Message: "Are you sure you want to exit?"}
			survey.AskOne(prompt, &confirm)
			if !confirm {
				return
			}
		}
	}
	os.Exit(0)
}

// Commands returns the `exit` command.
func Command(con *console.SliverClient) []*cobra.Command {
	return []*cobra.Command{{
		Use:   "exit",
		Short: "Exit the program",
		Run: func(cmd *cobra.Command, args []string) {
			ExitCmd(cmd, con, args)
		},
		GroupID: constants.GenericHelpGroup,
	}}
}
