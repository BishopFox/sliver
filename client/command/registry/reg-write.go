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
	"encoding/hex"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
	"google.golang.org/protobuf/proto"
)

// RegWriteCmd - Write to a Windows registry key: registry write --hive HKCU --type dword "software\google\chrome\blbeacon\hello" 32
func RegWriteCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session, beacon := con.ActiveTarget.GetInteractive()
	if session == nil && beacon == nil {
		return
	}
	targetOS := getOS(session, beacon)
	if targetOS != "windows" {
		con.PrintErrorf("Registry operations can only target Windows\n")
		return
	}

	var (
		dwordValue  uint32
		qwordValue  uint64
		stringValue string
		binaryValue []byte
	)

	binPath := ctx.Flags.String("path")
	hostname := ctx.Flags.String("hostname")
	flagType := ctx.Flags.String("type")
	valType, err := getType(flagType)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	hive := ctx.Flags.String("hive")
	if err := checkHive(hive); err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	regPath := ctx.Args.String("registry-path")
	value := ctx.Args.String("value")
	if regPath == "" || value == "" {
		con.PrintErrorf("You must provide a path and a value to write")
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
	switch valType {
	case sliverpb.RegistryTypeBinary:
		var (
			v   []byte
			err error
		)
		if binPath == "" {
			v, err = hex.DecodeString(value)
			if err != nil {
				con.PrintErrorf("%s\n", err)
				return
			}
		} else {
			v, err = ioutil.ReadFile(binPath)
			if err != nil {
				con.PrintErrorf("%s\n", err)
				return
			}
		}
		binaryValue = v
	case sliverpb.RegistryTypeDWORD:
		v, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		dwordValue = uint32(v)
	case sliverpb.RegistryTypeQWORD:
		v, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			con.PrintErrorf("%s\n", err)
			return
		}
		qwordValue = v
	case sliverpb.RegistryTypeString:
		stringValue = value
	default:
		con.PrintErrorf("Invalid type")
		return
	}
	regWrite, err := con.Rpc.RegistryWrite(context.Background(), &sliverpb.RegistryWriteReq{
		Request:     con.ActiveTarget.Request(ctx),
		Hostname:    hostname,
		Hive:        hive,
		Path:        finalPath,
		Type:        valType,
		Key:         key,
		StringValue: stringValue,
		DWordValue:  dwordValue,
		QWordValue:  qwordValue,
		ByteValue:   binaryValue,
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	if regWrite.Response != nil && regWrite.Response.Async {
		con.AddBeaconCallback(regWrite.Response.TaskID, func(task *clientpb.BeaconTask) {
			err = proto.Unmarshal(task.Response, regWrite)
			if err != nil {
				con.PrintErrorf("Failed to decode response %s\n", err)
				return
			}
			PrintRegWrite(regWrite, con)
		})
		con.PrintAsyncResponse(regWrite.Response)
	} else {
		PrintRegWrite(regWrite, con)
	}
}

// PrintRegWrite - Print the registry write operation
func PrintRegWrite(regWrite *sliverpb.RegistryWrite, con *console.SliverConsoleClient) {
	if regWrite.Response != nil && regWrite.Response.Err != "" {
		con.PrintErrorf("%s", regWrite.Response.Err)
		return
	}
	con.PrintInfof("Value written to registry\n")
}
