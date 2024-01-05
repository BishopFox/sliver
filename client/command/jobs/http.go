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
	"github.com/spf13/cobra"
)

// HTTPListenerCmd - Start an HTTP listener.
func HTTPListenerCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	domain, _ := cmd.Flags().GetString("domain")
	lhost, _ := cmd.Flags().GetString("lhost")
	lport, _ := cmd.Flags().GetUint32("lport")
	disableOTP, _ := cmd.Flags().GetBool("disable-otp")
	pollTimeout, _ := cmd.Flags().GetString("long-poll-timeout")
	pollJitter, _ := cmd.Flags().GetString("long-poll-jitter")
	website, _ := cmd.Flags().GetString("website")

	longPollTimeout, err := time.ParseDuration(pollTimeout)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	longPollJitter, err := time.ParseDuration(pollJitter)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	con.PrintInfof("Starting HTTP %s:%d listener ...\n", domain, lport)
	http, err := con.Rpc.StartHTTPListener(context.Background(), &clientpb.HTTPListenerReq{
		Domain:          domain,
		Website:         website,
		Host:            lhost,
		Port:            lport,
		Secure:          false,
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
