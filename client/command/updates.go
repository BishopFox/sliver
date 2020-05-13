package command

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"time"

	"github.com/bishopfox/sliver/client/version"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/desertbit/grumble"
)

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

func updates(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {

	verboseVersions(ctx, rpc)

	timeout := time.Duration(ctx.Flags.Int("timeout")) * time.Second
	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: timeout,
			}).Dial,
			TLSHandshakeTimeout: timeout,
		},
	}

	fmt.Printf("\nChecking for updates ... ")
	prereleases := ctx.Flags.Bool("prereleases")
	release, err := version.CheckForUpdates(client, prereleases)
	fmt.Printf("done!\n\n")
	if err != nil {
		fmt.Printf(Warn+"Update check failed %s", err)
		return
	}

	if release != nil {
		fmt.Printf("New version available: %s\n", release.TagName)
		fmt.Println(release.HTMLURL)
	} else {
		fmt.Printf(Info + "No new releases.\n")
	}
}

func verboseVersions(ctx *grumble.Context, rpc rpcpb.SliverRPCClient) {
	clientVer := version.FullVersion()
	serverVer, err := rpc.GetVersion(context.Background(), &commonpb.Empty{})
	if err != nil {
		fmt.Printf(Warn+"Failed to check server version %s", err)
		return
	}

	fmt.Printf(Info+"Client v%s - %s/%s\n", clientVer, runtime.GOOS, runtime.GOARCH)
	clientCompiledAt, _ := version.Compiled()
	fmt.Printf("    Compiled at %s\n\n", clientCompiledAt)

	fmt.Println()
	fmt.Printf(Info+"Server v%d.%d.%d - %s - %s/%s\n",
		serverVer.Major, serverVer.Minor, serverVer.Patch, serverVer.Commit,
		serverVer.OS, serverVer.Arch)
	serverCompiledAt := time.Unix(serverVer.CompiledAt, 0)
	fmt.Printf("    Compiled at %s\n", serverCompiledAt)
}
