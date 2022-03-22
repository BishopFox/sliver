package operator

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

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/prelude"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/desertbit/grumble"
)

func ConnectCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	url := ctx.Args.String("connection-string")
	aesKey := ctx.Flags.String("aes-key")
	agentRange := ctx.Flags.String("range")
	skipExisting := ctx.Flags.Bool("skip-existing")

	config := &prelude.OperatorConfig{
		Range:       agentRange,
		OperatorURL: url,
		RPC:         con.Rpc,
		AESKey:      aesKey,
	}

	implantMapper := prelude.InitImplantMapper(config)

	con.PrintInfof("Connected to Operator at %s\n", url)
	if !skipExisting {
		sessions, err := con.Rpc.GetSessions(context.Background(), &commonpb.Empty{})
		if err != nil {
			con.PrintErrorf("Could not get session list: %s", err)
			return
		}
		if len(sessions.Sessions) > 0 {
			con.PrintInfof("Adding existing sessions ...\n")
			for _, session := range sessions.Sessions {
				if !session.IsDead {
					err = implantMapper.AddImplant(session, nil)
					if err != nil {
						con.PrintErrorf("Could not add session %s to implant mapper: %s", session.Name, err)
					}
				}
			}
			con.PrintInfof("Done !\n")
		}
		beacons, err := con.Rpc.GetBeacons(context.Background(), &commonpb.Empty{})
		if err != nil {
			con.PrintErrorf("Could not get beacon list: %s", err)
			return
		}
		if len(beacons.Beacons) > 0 {
			con.PrintInfof("Adding existing beacons ...\n")
			for _, beacon := range beacons.Beacons {
				err = implantMapper.AddImplant(beacon, func(taskID string, cb func(task *clientpb.BeaconTask)) {
					con.AddBeaconCallback(taskID, cb)
				})
				if err != nil {
					con.PrintErrorf("Could not add beacon %s to implant mapper: %s", beacon.Name, err)
				}
			}
			con.PrintInfof("Done !\n")
		}
	}
}
