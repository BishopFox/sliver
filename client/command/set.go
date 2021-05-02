package command

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
	"regexp"

	//consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"

	"github.com/desertbit/grumble"
)

func updateSessionCmd(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {

	// Option to change the agent name
	name := ctx.Flags.String("name")
	if name != "" {
		isAlphanumeric := regexp.MustCompile(`^[[:alnum:]]+$`).MatchString
		if !isAlphanumeric(name) {
			fmt.Printf(Warn + "Name must be in alphanumeric only\n")
			return
		}
	}

	// Option to change the reconnect interval
	reconnect := ctx.Flags.Int("reconnect")

	// Option to change the reconnect interval
	poll := ctx.Flags.Int("poll")

	session, err := rpc.UpdateSession(context.Background(), &clientpb.UpdateSession{
		SessionID:         ActiveSession.session.ID,
		Name:              name,
		ReconnectInterval: int32(reconnect),
		PollInterval:      int32(poll),
	})

	if err != nil {
		fmt.Printf(Warn+"Error: %v", err)
		return
	}

	ActiveSession.Set(session)

}
