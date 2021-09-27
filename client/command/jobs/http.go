package jobs

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
	"time"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/desertbit/grumble"
)

// HTTPListenerCmd - Start an HTTP listener
func HTTPListenerCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	domain := ctx.Flags.String("domain")
	lhost := ctx.Flags.String("lhost")
	lport := uint16(ctx.Flags.Int("lport"))
	disableOTP := ctx.Flags.Bool("disable-otp")
	longPollTimeout, err := time.ParseDuration(ctx.Flags.String("long-poll-timeout"))
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	longPollJitter, err := time.ParseDuration(ctx.Flags.String("long-poll-jitter"))
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	con.PrintInfof("Starting HTTP %s:%d listener ...\n", domain, lport)
	http, err := con.Rpc.StartHTTPListener(context.Background(), &clientpb.HTTPListenerReq{
		Domain:          domain,
		Website:         ctx.Flags.String("website"),
		Host:            lhost,
		Port:            uint32(lport),
		Secure:          false,
		Persistent:      ctx.Flags.Bool("persistent"),
		EnforceOTP:      !disableOTP,
		LongPollTimeout: int64(longPollTimeout),
		LongPollJitter:  int64(longPollJitter),
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
	} else {
		con.PrintInfof("Successfully started job #%d\n", http.JobID)
	}
}
