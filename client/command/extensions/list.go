package extensions

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

	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// ExtensionsListCmd - List all extension loaded on the active session/beacon
func ExtensionsListCmd(cmd *cobra.Command, con *console.SliverConsoleClient, args []string) {
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		return
	}

	extList, err := con.Rpc.ListExtensions(context.Background(), &sliverpb.ListExtensionsReq{
		Request: con.ActiveTarget.Request(cmd),
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	if extList.Response != nil && extList.Response.Err != "" {
		con.PrintErrorf("%s\n", extList.Response.Err)
		return
	}
	if len(extList.Names) > 0 {
		con.PrintInfof("Loaded extensions:\n")
		for _, ext := range extList.Names {
			con.Printf("- %s\n", ext)
		}
	}
}
