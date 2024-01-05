package registry

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
	"strings"

	"google.golang.org/protobuf/proto"

	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// RegDeleteKeyCmd - Remove a Windows registry key
func RegDeleteKeyCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}
	targetOS := getOS(session, beacon)
	if targetOS != "windows" {
		con.PrintErrorf("Registry operations can only target Windows\n")
		return
	}

	hostname, _ := cmd.Flags().GetString("hostname")
	hive, _ := cmd.Flags().GetString("hive")
	if err := checkHive(hive); err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	regPath := args[0]
	if regPath == "" {
		con.PrintErrorf("You must provide a path\n")
		return
	}
	if strings.Contains(regPath, "/") {
		regPath = strings.ReplaceAll(regPath, "/", "\\")
	}
	pathBaseIdx := strings.LastIndex(regPath, `\`)
	if pathBaseIdx < 0 {
		con.PrintErrorf("invalid path: %s", regPath)
		return
	}
	if len(regPath) < pathBaseIdx+1 {
		con.PrintErrorf("invalid path: %s", regPath)
		return
	}
	finalPath := regPath[:pathBaseIdx]
	key := regPath[pathBaseIdx+1:]

	deleteKey, err := con.Rpc.RegistryDeleteKey(context.Background(), &sliverpb.RegistryDeleteKeyReq{
		Hive:     hive,
		Path:     finalPath,
		Key:      key,
		Hostname: hostname,
		Request:  con.ActiveTarget.Request(cmd),
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	if deleteKey.Response != nil && deleteKey.Response.Async {
		con.AddBeaconCallback(deleteKey.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, deleteKey)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintDeleteKey(deleteKey, finalPath, key, con)
		})
		con.PrintAsyncResponse(deleteKey.Response)
	} else {
		PrintDeleteKey(deleteKey, finalPath, key, con)
	}
}

// PrintDeleteKey - Print the results of the delete key command
func PrintDeleteKey(deleteKey *sliverpb.RegistryDeleteKey, regPath string, key string, con *console.SliverClient) {
	if deleteKey.Response != nil && deleteKey.Response.Err != "" {
		con.PrintErrorf("%s", deleteKey.Response.Err)
		return
	}
	con.PrintInfof("Key removed at %s\\%s", regPath, key)
}
