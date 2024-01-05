package sessions

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
	"net/url"
	"time"

	"github.com/bishopfox/sliver/client/command/generate"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/spf13/cobra"
)

// InteractiveCmd - Beacon only command to open an interactive session.
func InteractiveCmd(cmd *cobra.Command, con *console.SliverClient, _ []string) {
	beacon := con.ActiveTarget.GetBeaconInteractive()
	if beacon == nil {
		return
	}

	delayF, _ := cmd.Flags().GetString("delay")
	delay, err := time.ParseDuration(delayF)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	// Parse C2 Flags
	c2s := []*clientpb.ImplantC2{}

	mtlsC2F, _ := cmd.Flags().GetString("mtls")
	mtlsC2, err := generate.ParseMTLSc2(mtlsC2F)
	if err != nil {
		con.PrintErrorf("%s\n", err.Error())
		return
	}
	c2s = append(c2s, mtlsC2...)

	wgC2F, _ := cmd.Flags().GetString("wg")
	wgC2, err := generate.ParseWGc2(wgC2F)
	if err != nil {
		con.PrintErrorf("%s\n", err.Error())
		return
	}
	c2s = append(c2s, wgC2...)

	httpC2F, _ := cmd.Flags().GetString("http")
	httpC2, err := generate.ParseHTTPc2(httpC2F)
	if err != nil {
		con.PrintErrorf("%s\n", err.Error())
		return
	}
	c2s = append(c2s, httpC2...)

	dnsC2F, _ := cmd.Flags().GetString("dns")
	dnsC2, err := generate.ParseDNSc2(dnsC2F)
	if err != nil {
		con.PrintErrorf("%s\n", err.Error())
		return
	}
	c2s = append(c2s, dnsC2...)

	namedPipeC2F, _ := cmd.Flags().GetString("named-pipe")
	namedPipeC2, err := generate.ParseNamedPipec2(namedPipeC2F)
	if err != nil {
		con.PrintErrorf("%s\n", err.Error())
		return
	}
	c2s = append(c2s, namedPipeC2...)

	tcpPivotC2F, _ := cmd.Flags().GetString("tcp-pivot")
	tcpPivotC2, err := generate.ParseTCPPivotc2(tcpPivotC2F)
	if err != nil {
		con.PrintErrorf("%s\n", err.Error())
		return
	}
	c2s = append(c2s, tcpPivotC2...)

	// No flags, parse the current beacon's ActiveC2 instead
	if len(c2s) == 0 {
		con.PrintInfof("Using beacon's active C2 endpoint: %s\n", beacon.ActiveC2)
		c2url, err := url.Parse(beacon.ActiveC2)
		if err != nil {
			con.PrintErrorf("%s\n", err.Error())
			return
		}
		switch c2url.Scheme {
		case "mtls":
			mtlsC2, err = generate.ParseMTLSc2(beacon.ActiveC2)
			if err != nil {
				con.PrintErrorf("%s\n", err.Error())
				return
			}
			c2s = append(c2s, mtlsC2...)
		case "wg":
			wgC2, err = generate.ParseWGc2(beacon.ActiveC2)
			if err != nil {
				con.PrintErrorf("%s\n", err.Error())
				return
			}
			c2s = append(c2s, wgC2...)
		case "https":
			fallthrough
		case "http":
			httpC2, err = generate.ParseHTTPc2(beacon.ActiveC2)
			if err != nil {
				con.PrintErrorf("%s\n", err.Error())
				return
			}
			c2s = append(c2s, httpC2...)
		case "dns":
			dnsC2, err = generate.ParseDNSc2(beacon.ActiveC2)
			if err != nil {
				con.PrintErrorf("%s\n", err.Error())
				return
			}
			c2s = append(c2s, dnsC2...)
		case "namedpipe":
			namedPipeC2, err = generate.ParseNamedPipec2(beacon.ActiveC2)
			if err != nil {
				con.PrintErrorf("%s\n", err.Error())
				return
			}
			c2s = append(c2s, namedPipeC2...)
		case "tcppivot":
			tcpPivotC2, err = generate.ParseTCPPivotc2(beacon.ActiveC2)
			if err != nil {
				con.PrintErrorf("%s\n", err.Error())
				return
			}
			c2s = append(c2s, tcpPivotC2...)
		default:
			con.PrintErrorf("Unsupported C2 scheme: %s\n", c2url.Scheme)
			return
		}
	}

	openSession := &sliverpb.OpenSession{
		Request: con.ActiveTarget.Request(cmd),
		C2S:     []string{},
		Delay:   int64(delay),
	}
	for _, c2 := range c2s {
		openSession.C2S = append(openSession.C2S, c2.URL)
	}

	openSession, err = con.Rpc.OpenSession(context.Background(), openSession)
	if err != nil {
		con.PrintErrorf("%s\n", err)
	}
	if openSession.Response != nil && openSession.Response.Async {
		con.PrintAsyncResponse(openSession.Response)
	} else {
		con.PrintErrorf("rpc response missing!\n")
	}
}
