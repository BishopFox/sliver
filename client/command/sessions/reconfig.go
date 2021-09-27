package sessions

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
	"regexp"
	"time"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"

	"github.com/desertbit/grumble"
)

// SessionsReconfigCmd - Reconfigure metadata about a sessions
func SessionsReconfigCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
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

	// Option to change the reconnect interval
	reconnect := ctx.Flags.Int("reconnect")

	// Option to change the reconnect interval
	poll := ctx.Flags.Int("poll")

	oldSession := con.ActiveTarget.GetSession()
	session, err := con.Rpc.UpdateSession(context.Background(), &clientpb.UpdateSession{
		SessionID:         oldSession.ID,
		Name:              name,
		ReconnectInterval: int64(reconnect) * int64(time.Second),
		PollInterval:      int64(poll) * int64(time.Second),
	})

	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	con.ActiveTarget.Set(session, nil)
}
