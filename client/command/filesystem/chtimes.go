package filesystem

/*
	Copyright (C) 2023 b0yd

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
	"strconv"
	"time"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

const chtimesDefaultLayout = "2006-01-02 15:04:05"

type chtimesTimeFormat struct {
	name  string
	parse func(string) (int64, error)
}

func chtimesParseUnixSeconds(value string) (int64, error) {
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, err
	}
	return parsed, nil
}

func chtimesParseUnixMillis(value string) (int64, error) {
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, err
	}
	return time.UnixMilli(parsed).Unix(), nil
}

func chtimesParseLayout(layout string) func(string) (int64, error) {
	return func(value string) (int64, error) {
		parsed, err := time.Parse(layout, value)
		if err != nil {
			return 0, err
		}
		return parsed.Unix(), nil
	}
}

func chtimesFormatFromFlags(cmd *cobra.Command) (chtimesTimeFormat, error) {
	formatFlags := []struct {
		flag   string
		format chtimesTimeFormat
	}{
		{flag: "unix", format: chtimesTimeFormat{name: "unix", parse: chtimesParseUnixSeconds}},
		{flag: "unix-ms", format: chtimesTimeFormat{name: "unix-ms", parse: chtimesParseUnixMillis}},
		{flag: "rfc3339", format: chtimesTimeFormat{name: "rfc3339", parse: chtimesParseLayout(time.RFC3339)}},
		{flag: "rfc1123", format: chtimesTimeFormat{name: "rfc1123", parse: chtimesParseLayout(time.RFC1123)}},
	}

	var selected *chtimesTimeFormat
	selectedFlag := ""
	for _, candidate := range formatFlags {
		enabled, _ := cmd.Flags().GetBool(candidate.flag)
		if !enabled {
			continue
		}
		if selected != nil {
			return chtimesTimeFormat{}, fmt.Errorf("only one time format flag can be used (--%s and --%s are both set)", selectedFlag, candidate.flag)
		}
		selected = &candidate.format
		selectedFlag = candidate.flag
	}

	if selected == nil {
		return chtimesTimeFormat{name: "datetime", parse: chtimesParseLayout(chtimesDefaultLayout)}, nil
	}

	return *selected, nil
}

// ChtimesCmd - Change the access and modified time of a file on the remote file system.
func ChtimesCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}
	filePath := args[0]

	if filePath == "" {
		con.PrintErrorf("Missing parameter: file or directory name\n")
		return
	}

	format, err := chtimesFormatFromFlags(cmd)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	atime := args[1]
	if atime == "" {
		con.PrintErrorf("Missing parameter: Last accessed time id\n")
		return
	}

	unixAtime, err := format.parse(atime)
	if err != nil {
		con.PrintErrorf("Invalid access time (%s): %s\n", format.name, err)
		return
	}

	mtime := args[2]
	if mtime == "" {
		con.PrintErrorf("Missing parameter: Last modified time id\n")
		return
	}

	unixMtime, err := format.parse(mtime)
	if err != nil {
		con.PrintErrorf("Invalid modified time (%s): %s\n", format.name, err)
		return
	}

	chtimes, err := con.Rpc.Chtimes(context.Background(), &sliverpb.ChtimesReq{
		Request: con.ActiveTarget.Request(cmd),
		Path:    filePath,
		ATime:   unixAtime,
		MTime:   unixMtime,
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if chtimes.Response != nil && chtimes.Response.Async {
		con.AddBeaconCallback(chtimes.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, chtimes)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintChtimes(chtimes, con)
		})
		con.PrintAsyncResponse(chtimes.Response)
	} else {
		PrintChtimes(chtimes, con)
	}
}

// PrintChtimes - Print the Chtimes response.
func PrintChtimes(chtimes *sliverpb.Chtimes, con *console.SliverClient) {
	if chtimes.Response != nil && chtimes.Response.Err != "" {
		con.PrintErrorf("%s\n", chtimes.Response.Err)
		return
	}
	con.PrintInfof("%s\n", chtimes.Path)
}
