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

	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
)

// MTLSListenerCmd - Start an mTLS listener
func MTLSListenerCmd(cmd *cobra.Command, con *console.SliverConsoleClient, args []string) {
	lhost, _ := cmd.Flags().GetString("lhost")
	lport, _ := cmd.Flags().GetUint32("lport")
	persistent, _ := cmd.Flags().GetBool("persistent")

	con.PrintInfof("Starting mTLS listener ...\n")
	mtls, err := con.Rpc.StartMTLSListener(context.Background(), &clientpb.MTLSListenerReq{
		Host:       lhost,
		Port:       lport,
		Persistent: persistent,
	})
	con.Println()
	if err != nil {
		con.PrintErrorf("%s\n", err)
	} else {
		con.PrintInfof("Successfully started job #%d\n", mtls.JobID)
	}
}
