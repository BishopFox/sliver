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
	"time"

	"github.com/bishopfox/sliver/client/command/generate"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
)

// InteractiveCmd - Beacon only command to open an interactive session
func InteractiveCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	beacon := con.ActiveTarget.GetBeaconInteractive()
	if beacon == nil {
		return
	}

	delay, err := time.ParseDuration(ctx.Flags.String("delay"))
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	c2s := []*clientpb.ImplantC2{}

	mtlsC2 := generate.ParseMTLSc2(ctx.Flags.String("mtls"))
	c2s = append(c2s, mtlsC2...)

	wgC2 := generate.ParseWGc2(ctx.Flags.String("wg"))
	c2s = append(c2s, wgC2...)

	httpC2 := generate.ParseHTTPc2(ctx.Flags.String("http"))
	c2s = append(c2s, httpC2...)

	dnsC2 := generate.ParseDNSc2(ctx.Flags.String("dns"))
	c2s = append(c2s, dnsC2...)

	namedPipeC2 := generate.ParseNamedPipec2(ctx.Flags.String("named-pipe"))
	c2s = append(c2s, namedPipeC2...)

	tcpPivotC2 := generate.ParseTCPPivotc2(ctx.Flags.String("tcp-pivot"))
	c2s = append(c2s, tcpPivotC2...)

	if len(mtlsC2) == 0 && len(wgC2) == 0 && len(httpC2) == 0 && len(dnsC2) == 0 && len(namedPipeC2) == 0 && len(tcpPivotC2) == 0 {
		con.PrintErrorf("Must specify at least one of --mtls, --wg, --http, --dns, --named-pipe, or --tcp-pivot\n")
		return
	}

	openSession := &sliverpb.OpenSession{
		Request: con.ActiveTarget.Request(ctx),
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
