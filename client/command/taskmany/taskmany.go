package taskmany

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

	"github.com/bishopfox/sliver/client/console"
	"github.com/desertbit/grumble"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/command/filesystem"
	"github.com/bishopfox/sliver/client/command/exec"
)

var (
	ErrNoSelection = errors.New("no selection")
)


// TaskmanyCmd - Task many beacons
func TaskmanyCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
    con.PrintErrorf("Not implemented yet\n")
}

// Wrap a function to run it for each beacon
func WrapFunction(con *console.SliverConsoleClient, f func(ctx *grumble.Context) error) func(ctx *grumble.Context) error {
    return func(ctx *grumble.Context) error {
        beacons, err := con.Rpc.GetBeacons(context.Background(), &commonpb.Empty{})
        if err != nil {
                con.PrintErrorf("%s\n", err)
                return nil
        }
        sessions, err := con.Rpc.GetSessions(context.Background(), &commonpb.Empty{})
        if err != nil {
                con.PrintErrorf("%s\n", err)
                return nil
        }

        n := 0
        for _, b := range beacons.Beacons {
                con.ActiveTarget.Set(nil, b)
                f(ctx)
                n += 1
        }

        for _, s := range sessions.Sessions {
                con.ActiveTarget.Set(s, nil)
                f(ctx)
                n += 1
        }
        con.PrintInfof("Tasked %d of your beacons >:D\n", n)
        return nil
    }
}
