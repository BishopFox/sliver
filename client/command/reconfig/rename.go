package reconfig

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

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
	"regexp"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/desertbit/grumble"
)

// RecnameCmd - Reconfigure metadata about a sessions
func RenameCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}

	// Option to change the agent name
	name := ctx.Flags.String("name")
	if name != "" {
		isAlphanumeric := regexp.MustCompile(`^[[:alnum:]]+$`).MatchString
		if !isAlphanumeric(name) {
			con.PrintErrorf("Name must be in alphanumeric only\n")
			return
		}
	}

	var beaconID string
	var sessionID string
	if beacon != nil {
		beaconID = beacon.ID
	} else if session != nil {
		sessionID = session.ID
	}
	_, err := con.Rpc.Rename(context.Background(), &clientpb.RenameReq{
		SessionID: sessionID,
		BeaconID:  beaconID,
		Name:      name,
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	con.PrintInfof("Renamed implant to %s\n", name)
	con.ActiveTarget.Set(nil, nil)
}
