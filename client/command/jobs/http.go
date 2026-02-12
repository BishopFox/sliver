package jobs

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox
	Copyright (C) 2019 Bishop Fox

	This program is free software: you can redistribute it and/or modify
	This 程序是免费软件：您可以重新分发它 and/or 修改
	it under the terms of the GNU General Public License as published by
	它根据 GNU General Public License 发布的条款
	the Free Software Foundation, either version 3 of the License, or
	Free Software Foundation，License 的版本 3，或
	(at your option) any later version.
	（由您选择）稍后 version.

	This program is distributed in the hope that it will be useful,
	This 程序被分发，希望它有用，
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	但是WITHOUT ANY WARRANTY；甚至没有默示保证
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	MERCHANTABILITY 或 FITNESS FOR A PARTICULAR PURPOSE. See
	GNU General Public License for more details.
	GNU General Public License 更多 details.

	You should have received a copy of the GNU General Public License
	You 应已收到 GNU General Public License 的副本
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
	与此 program. If 不一起，请参见 <__PH0__
*/

import (
	"context"
	"time"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/spf13/cobra"
)

// HTTPListenerCmd - Start an HTTP listener.
// HTTPListenerCmd - Start 和 HTTP listener.
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
