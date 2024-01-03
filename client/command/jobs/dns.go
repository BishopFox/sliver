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
	"strings"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/spf13/cobra"
)

// DNSListenerCmd - Start a DNS lisenter.
func DNSListenerCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	domainsF, _ := cmd.Flags().GetString("domains")
	domains := strings.Split(domainsF, ",")
	for index, domain := range domains {
		if !strings.HasSuffix(domain, ".") {
			domains[index] += "."
		}
	}

	lhost, _ := cmd.Flags().GetString("lhost")
	lport, _ := cmd.Flags().GetUint32("lport")
	canaries, _ := cmd.Flags().GetBool("no-canaries")
	enforceOTP, _ := cmd.Flags().GetBool("disable-otp")

	con.PrintInfof("Starting DNS listener with parent domain(s) %v ...\n", domains)
	dns, err := con.Rpc.StartDNSListener(context.Background(), &clientpb.DNSListenerReq{
		Domains:    domains,
		Host:       lhost,
		Port:       lport,
		Canaries:   !canaries,
		EnforceOTP: !enforceOTP,
	})
	con.Println()
	if err != nil {
		con.PrintErrorf("%s\n", err)
	} else {
		con.PrintInfof("Successfully started job #%d\n", dns.JobID)
	}
}
