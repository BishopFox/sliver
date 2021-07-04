package filesystem

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
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util"

	"github.com/desertbit/grumble"
)

// LsCmd - List the contents of a remote directory
func LsCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveSession.GetInteractive()
	if session == nil {
		return
	}

	remotePath := ctx.Args.String("path")

	ls, err := con.Rpc.Ls(context.Background(), &sliverpb.LsReq{
		Request: con.ActiveSession.Request(ctx),
		Path:    remotePath,
	})
	if err != nil {
		con.PrintWarnf("%s\n", err)
	} else {
		PrintLs(con.App.Stdout(), ls)
	}
}

func PrintLs(stdout io.Writer, ls *sliverpb.Ls) {
	fmt.Fprintf(stdout, "%s\n", ls.Path)
	fmt.Fprintf(stdout, "%s\n", strings.Repeat("=", len(ls.Path)))

	table := tabwriter.NewWriter(stdout, 0, 2, 2, ' ', 0)
	for _, fileInfo := range ls.Files {
		if fileInfo.IsDir {
			fmt.Fprintf(table, "%s\t<dir>\t\n", fileInfo.Name)
		} else {
			fmt.Fprintf(table, "%s\t%s\t\n", fileInfo.Name, util.ByteCountBinary(fileInfo.Size))
		}
	}
	table.Flush()
}
