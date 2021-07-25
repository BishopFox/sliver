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
	"strings"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/desertbit/grumble"
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
func RegReadCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	session := con.ActiveSession.GetInteractive()
	if session == nil {
		return
	}
	if session.OS != "windows" {
		con.PrintErrorf("Command not supported on this operating system.")
		return
	}

	hostname := ctx.Flags.String("hostname")
	hive := ctx.Flags.String("hive")
	if err := checkHive(hive); err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	regPath := ctx.Args.String("registry-path")
	if regPath == "" {
		con.PrintErrorf("You must provide a path")
		return
	}
	if strings.Contains(regPath, "/") {
		regPath = strings.ReplaceAll(regPath, "/", "\\")
	}
	slashIndex := strings.LastIndex(regPath, "\\")
	key := regPath[slashIndex+1:]
	regPath = regPath[:slashIndex]
	regRead, err := con.Rpc.RegistryRead(context.Background(), &sliverpb.RegistryReadReq{
		Hive:     hive,
		Path:     regPath,
		Key:      key,
		Hostname: hostname,
		Request:  con.ActiveSession.Request(ctx),
	})

	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	if regRead.Response != nil && regRead.Response.Err != "" {
		con.PrintErrorf("%s\n", regRead.Response.Err)
		return
	}
	con.Println(regRead.Value)
}
