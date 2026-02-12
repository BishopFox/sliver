package creds

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox
	Copyright (C) 2022 Bishop Fox

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
	"fmt"
	"strings"

	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
)

// CredsCmd - Manage credentials.
func CredsCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
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

func PrintCreds(creds []*clientpb.Credential, con *console.SliverClient) {
	collections := make(map[string][]*clientpb.Credential)
	for _, cred := range creds {
		collections[cred.Collection] = append(collections[cred.Collection], cred)
	}
	for collection, creds := range collections {
		printCollection(collection, creds, con)
		con.Println()
	}
}

func printCollection(collection string, creds []*clientpb.Credential, con *console.SliverClient) {
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	if collection != "" {
		tw.SetTitle(console.StyleBold.Render(collection))
	} else {
		tw.SetTitle(console.StyleBold.Render("Default Collection"))
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

// CredsHashTypeCompleter completes hash types.
// CredsHashTypeCompleter 完成哈希 types.
func CredsHashTypeCompleter(con *console.SliverClient) carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		results := make([]string, 0)

		for hashType, desc := range hashTypes {
			results = append(results, hashType)
			results = append(results, desc)
		}

		return carapace.ActionValuesDescribed(results...).Tag("hash types")
	})
}

// CredsHashFileFormatCompleter completes file formats for hash-files.
// CredsHashFileFormatCompleter 完成 hash__PH0__. 的文件格式
func CredsHashFileFormatCompleter(con *console.SliverClient) carapace.Action {
	return carapace.ActionValuesDescribed(
		UserColonHashNewlineFormat, "One hash per line.",
		HashNewlineFormat, "A file containing lines of 'username:hash' pairs.",
		CSVFormat, "A CSV file containing 'username,hash' pairs (additional columns ignored).",
	).Tag("hash file formats")
}

// CredsCollectionCompleter completes existing creds collection names.
// CredsCollectionCompleter 完成现有信用收集 names.
func CredsCollectionCompleter(con *console.SliverClient) carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		results := make([]string, 0)

		creds, err := con.Rpc.Creds(context.Background(), &commonpb.Empty{})
		if err != nil {
			return carapace.ActionMessage("failed to fetch credentials: %s", err.Error())
		}
		if len(creds.Credentials) == 0 {
			return carapace.Action{}
		}

		for _, cred := range creds.Credentials {
			if cred.Collection != "" {
				results = append(results, cred.Collection)
			}
		}

		return carapace.ActionValues(results...).Tag("creds collections")
	})
}

// CredsCredentialIDCompleter completes credential IDs.
// CredsCredentialIDCompleter 完成凭证 IDs.
func CredsCredentialIDCompleter(con *console.SliverClient) carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		results := make([]string, 0)

		creds, err := con.Rpc.Creds(context.Background(), &commonpb.Empty{})
		if err != nil {
			return carapace.ActionMessage("failed to fetch credentials: %s", err.Error())
		}
		if len(creds.Credentials) == 0 {
			return carapace.Action{}
		}

		for _, cred := range creds.Credentials {
			results = append(results, cred.ID)

			var hostID string
			if cred.OriginHostUUID != "" {
				if len(cred.OriginHostUUID) > 8 {
					hostID = cred.OriginHostUUID[8:]
				} else {
					hostID = cred.OriginHostUUID
				}
			} else {
				hostID = "None"
			}

			var username string
			if cred.Username != "" {
				username = fmt.Sprintf(" (user: %s)", cred.Username)
			}

			var cracked string
			if cred.IsCracked {
				cracked = "[C]"
			} else {
				cracked = "[ ]"
			}

			desc := fmt.Sprintf("[Host: %s] ( %s ) %s%s", hostID, cred.HashType.String(), cracked, username)
			results = append(results, desc)

		}

		return carapace.ActionValuesDescribed(results...).Tag("credentials")
	})
}
