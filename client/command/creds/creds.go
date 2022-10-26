package creds

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

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

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/desertbit/grumble"
	"github.com/jedib0t/go-pretty/v6/table"
)

// CredsCmd - Manage credentials
func CredsCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	creds, err := con.Rpc.Creds(context.Background(), &commonpb.Empty{})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if len(creds.Credentials) == 0 {
		con.PrintInfof("No credentials 🙁\n")
		return
	}
	PrintCreds(creds.Credentials, con)
}

func PrintCreds(creds []*clientpb.Credential, con *console.SliverConsoleClient) {
	collections := make(map[string][]*clientpb.Credential)
	for _, cred := range creds {
		collections[cred.Collection] = append(collections[cred.Collection], cred)
	}
	for collection, creds := range collections {
		printCollection(collection, creds, con)
		con.Println()
	}
}

func printCollection(collection string, creds []*clientpb.Credential, con *console.SliverConsoleClient) {
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	if collection != "" {
		tw.SetTitle(console.Bold + collection + console.Normal)
	} else {
		tw.SetTitle(console.Bold + "Default Collection" + console.Normal)
	}
	tw.AppendHeader(table.Row{
		"ID",
		"Username",
		"Plaintext",
		"Hash",
		"Hash Type",
		"Cracked",
	})
	for _, cred := range creds {
		tw.AppendRow(table.Row{
			strings.Split(cred.ID, "-")[0],
			cred.Username,
			cred.Plaintext,
			cred.Hash,
			cred.HashType,
			cred.IsCracked,
		})
	}
	con.Printf("%s\n", tw.Render())
}
